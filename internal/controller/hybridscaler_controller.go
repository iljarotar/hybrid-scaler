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
	"math"
	"time"

	"gopkg.in/inf.v0"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	scalingv1 "github.com/iljarotar/hybrid-scaler/api/v1"
	"github.com/iljarotar/hybrid-scaler/internal/reinforcement"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	ownerKey = ".metadata.controller"
)

// HybridScalerReconciler reconciles a HybridScaler object
type HybridScalerReconciler struct {
	client.Client
	PromAPI promv1.API
	Scheme  *runtime.Scheme
}

//+kubebuilder:rbac:groups=scaling.autoscaling.custom,resources=hybridscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scaling.autoscaling.custom,resources=hybridscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scaling.autoscaling.custom,resources=hybridscalers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

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
			logger.Error(err, "no scaler found", "namespaced name", req.NamespacedName)
			return result, client.IgnoreNotFound(err)
		}

		logger.Error(err, "cannot fetch scaler", "namespaced name", req.NamespacedName)
		return result, nil
	}
	if scaler.Spec.Interval != nil {
		result.RequeueAfter = time.Duration(*scaler.Spec.Interval) * time.Second
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: scaler.Spec.ScaleTargetRef.Name}, &deployment); err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "no deployment found for scaler", "scaler", scaler)
			return result, nil
		}

		logger.Error(err, "cannot fetch deployment for scaler", "scaler", scaler)
		return result, nil
	}

	scaler.Status.Replicas = deployment.Status.Replicas
	scaler.Status.ContainerResources = getContainerResources(deployment.Spec.Template.Spec.Containers)

	var replicaSets appsv1.ReplicaSetList
	if err := r.List(ctx, &replicaSets, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: deployment.Name}); err != nil {
		logger.Error(err, "cannot list replica sets for deployment", "deployment", deployment)
		return result, nil
	}

	pods := make([]corev1.Pod, 0)

	for _, rs := range replicaSets.Items {
		var podList corev1.PodList
		if err := r.List(ctx, &podList, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: rs.Name}); err != nil {
			logger.Error(err, "cannot list pods for deployment", "deployment", deployment)
			return result, nil
		}

		pods = append(pods, podList.Items...)
	}

	var averageCpuUsage float64
	var averageMemoryUsage float64
	for _, pod := range pods {
		query := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{pod="%s",service="kubelet"}[5m])) by (pod)`, pod.Name)
		res, _, err := r.PromAPI.Query(ctx, query, time.Now())
		if err != nil {
			logger.Error(err, "prometheus query error", "response", res)
			return result, nil
		}

		switch resTyped := res.(type) {
		case model.Vector:
			if len(resTyped) != 1 {
				logger.Error(fmt.Errorf("unexpected result length received from prometheus"), "unable to fetch pod metrics", "pod", pod.Name, "query", query, "result", resTyped)
				return result, nil
			}
			averageCpuUsage += float64(resTyped[0].Value)
		default:
			logger.Error(fmt.Errorf("unexpected result type received from prometheus"), "unable to fetch pod metrics", "pod", pod.Name, "query", query, "result", resTyped)
			return result, nil
		}

		query = fmt.Sprintf(`sum(container_memory_working_set_bytes{pod="%s",service="kubelet"}) by (pod)`, pod.Name)
		res, _, err = r.PromAPI.Query(ctx, query, time.Now())
		if err != nil {
			logger.Error(err, "prometheus query error", "response", res)
			return result, nil
		}

		switch resTyped := res.(type) {
		case model.Vector:
			if len(resTyped) != 1 {
				logger.Error(fmt.Errorf("unexpected result length received from prometheus"), "unable to fetch pod metrics", "pod", pod.Name, "query", query, "result", resTyped)
				return result, nil
			}

			averageMemoryUsage += float64(resTyped[0].Value)
		default:
			logger.Error(fmt.Errorf("unexpected result type received from prometheus"), "unable to fetch pod metrics", "pod", pod.Name, "query", query, "result", resTyped)
			return result, nil
		}
	}

	averageCpuUsage /= float64(len(pods))
	averageMemoryUsage /= float64(len(pods))

	podMetrics := scalingv1.PodMetrics{
		ResourceUsage: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewDecimalQuantity(*float64ToDec(averageCpuUsage), resource.DecimalExponent),
			corev1.ResourceMemory: *resource.NewDecimalQuantity(*float64ToDec(averageMemoryUsage), resource.DecimalExponent),
		},
	}
	scaler.Status.PodMetrics = podMetrics

	state, err := prepareState(scaler.Status, scaler.Spec)
	if err != nil {
		logger.Error(err, "cannot prepare scaling strategy state", "status", scaler.Status, "spec", scaler.Spec)
		return result, nil
	}
	logger.Info("prepared state for scaling strategy", "state", state)

	scalingStrategy := getScalingStrategy(scaler.Spec.LearningType, scaler.Spec.QLearningParams)

	decision, learningState, err := scalingStrategy.MakeDecision(state, scaler.Status.LearningState)
	if err != nil {
		logger.Error(err, "cannot make a scaling decision", "state", state)
		return result, nil
	}
	scaler.Status.LearningState = learningState

	if err := r.Status().Update(ctx, &scaler); err != nil {
		logger.Error(err, "unable to update scaler status", "status", scaler.Status)
		return result, nil
	}

	newResources := interpretResourceScaling(decision)
	newContainers := make([]corev1.Container, 0)

	deployment.Spec.Replicas = &decision.Replicas

	for _, container := range deployment.Spec.Template.Spec.Containers {
		resources, ok := newResources[container.Name]
		if !ok {
			logger.Error(err, "unable to find new resources for container", "container", container)
			return result, nil
		}

		container.Resources.Requests = resources.Requests
		container.Resources.Limits = resources.Limits

		newContainers = append(newContainers, container)
	}

	deployment.Spec.Template.Spec.Containers = newContainers

	if err := r.Update(ctx, &deployment); err != nil {
		logger.Error(err, "unable to update deployment spec")
		return result, nil
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
	podCpuRequests := inf.NewDec(0, 0)
	podMemoryRequests := inf.NewDec(0, 0)
	podCpuLimits := inf.NewDec(0, 0)
	podMemoryLimits := inf.NewDec(0, 0)

	containerResources := make(strategy.ContainerResources)

	for name, r := range status.ContainerResources {
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

		containerResources[name] = resources

		podCpuRequests.Add(podCpuRequests, cpuRequests)
		podCpuLimits.Add(podCpuLimits, cpuLimits)
		podMemoryRequests.Add(podMemoryRequests, memoryRequests)
		podMemoryLimits.Add(podMemoryLimits, memoryLimits)
	}

	podMetrics := strategy.PodMetrics{
		ResourceUsage: strategy.ResourcesList{
			CPU:    status.PodMetrics.ResourceUsage.Cpu().AsDec(),
			Memory: status.PodMetrics.ResourceUsage.Memory().AsDec(),
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
		LimitsToRequestsRatioCPU:    spec.ResourcePolicy.LimitsToRequestsRatioCPU.AsDec(),
		LimitsToRequestsRatioMemory: spec.ResourcePolicy.LimitsToRequestsRatioMemory.AsDec(),
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

		requests[corev1.ResourceCPU] = *resource.NewDecimalQuantity(*resources.Requests.CPU, resource.DecimalExponent)
		requests[corev1.ResourceMemory] = *resource.NewDecimalQuantity(*resources.Requests.Memory, resource.DecimalSI)
		limits[corev1.ResourceCPU] = *resource.NewDecimalQuantity(*resources.Limits.CPU, resource.DecimalExponent)
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
		underprovisioningPenalty := qParams.UnderprovisioningPenalty.AsDec()
		alpha := qParams.LearningRate.AsDec()
		gamma := qParams.DiscountFactor.AsDec()

		return reinforcement.NewQAgent(cpuCost, memoryCost, underprovisioningPenalty, alpha, gamma)
	default:
		return &strategy.NoOp{}
	}
}

func float64ToDec(value float64) *inf.Dec {
	integer, frac := math.Modf(value)

	scale := 10.0
	fracUnscaled := frac * math.Pow(10, scale)

	integerDec := inf.NewDec(int64(integer), 0)
	fracDec := inf.NewDec(int64(fracUnscaled), inf.Scale(scale))

	return new(inf.Dec).Add(integerDec, fracDec)
}
