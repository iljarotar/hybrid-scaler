/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/inf.v0"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	scalingv1 "github.com/iljarotar/hybrid-scaler/api/v1"
	"github.com/iljarotar/hybrid-scaler/internal/reinforcement"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
)

var (
	ownerKey = ".metadata.controller"
)

// HybridScalerReconciler reconciles a HybridScaler object
type HybridScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=scaling.autoscaling.custom,resources=hybridscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scaling.autoscaling.custom,resources=hybridscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scaling.autoscaling.custom,resources=hybridscalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HybridScaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *HybridScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	requeuePeriod := 15 * time.Second
	result := ctrl.Result{RequeueAfter: requeuePeriod}

	logger.Info("Reconcile", "req", req)

	var scaler scalingv1.HybridScaler
	if err := r.Get(ctx, req.NamespacedName, &scaler); err != nil {
		if errors.IsNotFound(err) {
			return result, client.IgnoreNotFound(err)
		}

		return result, err
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: scaler.Spec.ScaleTargetRef.Name}, &deployment); err != nil {
		if errors.IsNotFound(err) {
			return result, nil
		}

		return result, err
	}

	scaler.Status.Replicas = deployment.Status.Replicas
	scaler.Status.ContainerResources = getContainerResources(deployment.Spec.Template.Spec.Containers)

	var replicaSets appsv1.ReplicaSetList
	if err := r.List(ctx, &replicaSets, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: deployment.Name}); err != nil {
		return result, err
	}

	pods := make([]corev1.Pod, 0)

	for _, rs := range replicaSets.Items {
		var podList corev1.PodList
		if err := r.List(ctx, &podList, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: rs.Name}); err != nil {
			return result, err
		}

		pods = append(pods, podList.Items...)
	}

	// FIXME: getting pod metrics works now, but shows error: not allowed to watch; where is the watcher?
	for _, pod := range pods {
		var podMetrics v1beta1.PodMetrics
		if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: pod.Name}, &podMetrics, &client.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				return result, client.IgnoreNotFound(err)
			}

			return result, err
		}

		for _, metrics := range podMetrics.Containers {
			scaler.Status.ContainerMetrics = append(scaler.Status.ContainerMetrics, metrics)
		}
	}

	state, err := prepareState(scaler.Status, scaler.Spec)
	if err != nil {
		return result, err
	}

	scalingStrategy := getScalingStrategy(scaler.Spec.LearningType, scaler.Spec.QLearningParams)

	decision, learningState, err := scalingStrategy.MakeDecision(state, scaler.Status.LearningState)
	if err != nil {
		return result, err
	}

	scaler.Status.LearningState = learningState

	// FIXME: if there are 0 replicas, an error is thrown.
	if err := r.Status().Update(ctx, &scaler); err != nil {
		logger.Error(err, "unable to update scaler status")
		return result, err
	}

	newResources := interpretResourceScaling(decision)
	newContainers := make([]corev1.Container, 0)

	deployment.Spec.Replicas = &decision.Replicas

	for _, container := range deployment.Spec.Template.Spec.Containers {
		resources, ok := newResources[container.Name]
		if !ok {
			return result, fmt.Errorf("unable to find new resources for container %v", container.Name)
		}

		container.Resources.Requests = resources.Requests
		container.Resources.Limits = resources.Limits

		newContainers = append(newContainers, container)
	}

	deployment.Spec.Template.Spec.Containers = newContainers

	if err := r.Update(ctx, &deployment); err != nil {
		logger.Error(err, "unable to update deployment spec")
		return result, err
	}

	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HybridScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := addIndices(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1.HybridScaler{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func addIndices(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.ReplicaSet{}, ownerKey, func(rawObj client.Object) []string {
		replicaSet := rawObj.(*appsv1.ReplicaSet)
		owner := metav1.GetControllerOf(replicaSet)
		if owner == nil {
			return nil
		}

		if owner.Kind != "Deployment" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, ownerKey, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}

		if owner.Kind != "ReplicaSet" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return nil
}

func getContainerResources(containers []corev1.Container) map[string]scalingv1.ContainerResources {
	resources := make(map[string]scalingv1.ContainerResources)

	for _, container := range containers {
		requests, limits := container.Resources.Requests, container.Resources.Limits

		containerResources := scalingv1.ContainerResources{
			Requests: requests,
			Limits:   limits,
		}

		resources[container.Name] = containerResources
	}

	return resources
}

func prepareState(status scalingv1.HybridScalerStatus, spec scalingv1.HybridScalerSpec) (*strategy.State, error) {
	podCpuUsage := inf.NewDec(0, 0)
	podMemoryUsage := inf.NewDec(0, 0)

	podCpuRequests := inf.NewDec(0, 0)
	podMemoryRequests := inf.NewDec(0, 0)
	podCpuLimits := inf.NewDec(0, 0)
	podMemoryLimits := inf.NewDec(0, 0)

	containerResources := make(strategy.ContainerResources)

	for _, metrics := range status.ContainerMetrics {
		cpuUsage := metrics.Usage.Cpu().AsDec()
		memoryUsage := metrics.Usage.Cpu().AsDec()

		r, ok := status.ContainerResources[metrics.Name]
		if !ok {
			return nil, fmt.Errorf("unable to calculate resource usage of container %s", metrics.Name)
		}

		cpuRequests := r.Requests.Cpu().AsDec()
		memoryRequests := r.Requests.Memory().AsDec()
		cpuLimits := r.Limits.Cpu().AsDec()
		memoryLimits := r.Limits.Memory().AsDec()

		resources := strategy.Resources{
			Requests: strategy.ResourcesList{
				CPU:    cpuRequests,
				Memory: memoryRequests,
			},
			Limits: strategy.ResourcesList{
				CPU:    cpuLimits,
				Memory: memoryLimits,
			},
		}

		containerResources[metrics.Name] = resources

		podCpuRequests.Add(podCpuRequests, cpuRequests)
		podCpuLimits.Add(podCpuLimits, cpuLimits)
		podMemoryRequests.Add(podMemoryRequests, memoryRequests)
		podMemoryLimits.Add(podMemoryLimits, memoryLimits)

		podCpuUsage.Add(podCpuUsage, cpuUsage)
		podMemoryUsage.Add(podMemoryUsage, memoryUsage)
	}

	if status.Replicas == 0 {
		return nil, fmt.Errorf("number of replicas should not be zero")
	}

	replicas := inf.NewDec(int64(status.Replicas), 0)
	averagePodCpuUsage := new(inf.Dec).QuoRound(podCpuUsage, replicas, 8, inf.RoundHalfUp)
	averagePodMemoryUsage := new(inf.Dec).QuoRound(podMemoryUsage, replicas, 8, inf.RoundHalfUp)

	// TODO: also add pod overhead to average metrics

	podMetrics := strategy.PodMetrics{
		ResourceUsage: strategy.ResourcesList{
			CPU:    averagePodCpuUsage,
			Memory: averagePodMemoryUsage,
		},
		Resources: strategy.Resources{
			Requests: strategy.ResourcesList{
				CPU:    podCpuRequests,
				Memory: podMemoryRequests,
			},
			Limits: strategy.ResourcesList{
				CPU:    podCpuLimits,
				Memory: podMemoryLimits,
			},
		},
		// TODO: get from prometheus metrics
		LatencyThresholdExceeded: false,
	}

	constraints := strategy.Constraints{
		MinReplicas: ptr.Deref(spec.MinReplicas, 0),
		MaxReplicas: ptr.Deref(spec.MaxReplicas, 0),
		MinResources: strategy.ResourcesList{
			CPU:    spec.ResourcePolicy.MinAllowed.Cpu().AsDec(),
			Memory: spec.ResourcePolicy.MinAllowed.Memory().AsDec(),
		},
		MaxResources: strategy.ResourcesList{
			CPU:    spec.ResourcePolicy.MaxAllowed.Cpu().AsDec(),
			Memory: spec.ResourcePolicy.MaxAllowed.Memory().AsDec(),
		},
	}

	targetCpuUtilization, ok := spec.ResourcePolicy.TargetUtilization[corev1.ResourceCPU]
	if !ok {
		return nil, fmt.Errorf("cannot find target cpu utilization")
	}

	targetMemoryUtilization, ok := spec.ResourcePolicy.TargetUtilization[corev1.ResourceMemory]
	if !ok {
		return nil, fmt.Errorf("cannot find target memory utilization")
	}

	targetUtilization := strategy.ResourcesList{
		CPU:    inf.NewDec(int64(targetCpuUtilization), 2),
		Memory: inf.NewDec(int64(targetMemoryUtilization), 2),
	}

	state := &strategy.State{
		Replicas:           status.Replicas,
		Constraints:        constraints,
		ContainerResources: containerResources,
		PodMetrics:         podMetrics,
		TargetUtilization:  targetUtilization,
	}

	return state, nil
}

func interpretResourceScaling(decision *strategy.ScalingDecision) map[string]scalingv1.ContainerResources {
	containerResources := make(map[string]scalingv1.ContainerResources)

	for name, resources := range decision.ContainerResources {
		requests := make(corev1.ResourceList)
		limits := make(corev1.ResourceList)

		requests[corev1.ResourceCPU] = *resource.NewDecimalQuantity(*resources.Requests.CPU, resource.DecimalSI)
		requests[corev1.ResourceMemory] = *resource.NewDecimalQuantity(*resources.Requests.Memory, resource.DecimalSI)
		limits[corev1.ResourceCPU] = *resource.NewDecimalQuantity(*resources.Limits.CPU, resource.DecimalSI)
		limits[corev1.ResourceMemory] = *resource.NewDecimalQuantity(*resources.Limits.Memory, resource.DecimalSI)

		containerResources[name] = scalingv1.ContainerResources{
			Requests: requests,
			Limits:   limits,
		}
	}

	return containerResources
}

func getScalingStrategy(learningType scalingv1.LearningType, qParams scalingv1.QLearningParams) strategy.ScalingStrategy {
	switch learningType {
	case scalingv1.LearningTypeQLearning:
		cpuCost := qParams.CpuCost.AsDec()
		memoryCost := qParams.MemoryCost.AsDec()
		performancePenalty := qParams.PerformancePenalty.AsDec()
		alpha := qParams.LearningRate.AsDec()
		gamma := qParams.DiscountFactor.AsDec()

		return reinforcement.NewQAgent(cpuCost, memoryCost, performancePenalty, alpha, gamma)
	default:
		return &strategy.NoOp{}
	}
}
