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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	scalingv1 "github.com/iljarotar/hybrid-scaler/api/v1"
	"github.com/iljarotar/hybrid-scaler/internal/strategy"
)

var (
	ownerKey = ".metadata.controller"
)

// HybridScalerReconciler reconciles a HybridScaler object
type HybridScalerReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	ScalingStrategy strategy.ScalingStrategy
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

	logger.Info("Reconcile", "req", req)

	var scaler scalingv1.HybridScaler
	if err := r.Get(ctx, req.NamespacedName, &scaler); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		return ctrl.Result{}, err
	}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: scaler.Spec.ScaleTargetRef.Name}, &deployment); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: requeuePeriod}, nil
		}

		return ctrl.Result{}, err
	}

	scaler.Status.Replicas = deployment.Status.Replicas
	scaler.Status.ContainerResources = getContainerResources(deployment.Spec.Template.Spec.Containers)

	var replicaSets appsv1.ReplicaSetList
	if err := r.List(ctx, &replicaSets, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: deployment.Name}); err != nil {
		return ctrl.Result{}, err
	}

	pods := make([]corev1.Pod, 0)

	for _, rs := range replicaSets.Items {
		var podList corev1.PodList
		if err := r.List(ctx, &podList, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: rs.Name}); err != nil {
			return ctrl.Result{}, err
		}

		pods = append(pods, podList.Items...)
	}

	// FIXME: getting pod metrics works now, but shows error: not allowed to watch; where is the watcher?
	for _, pod := range pods {
		var podMetrics v1beta1.PodMetrics
		if err := r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: pod.Name}, &podMetrics, &client.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: requeuePeriod}, client.IgnoreNotFound(err)
			}

			return ctrl.Result{}, err
		}

		for _, metrics := range podMetrics.Containers {
			scaler.Status.ContainerMetrics = append(scaler.Status.ContainerMetrics, metrics)
		}
	}

	state, err := prepareState(scaler.Status)
	if err != nil {
		return ctrl.Result{}, err
	}

	decision := r.ScalingStrategy.MakeDecision(state)
	newResources := interpretResourceScaling(decision)

	logger.Info("scaling decision for state", "state", state, "decision", decision, "newResources", newResources)

	return ctrl.Result{RequeueAfter: requeuePeriod}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HybridScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := addIndices(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1.HybridScaler{}).
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

func getContainerResources(containers []corev1.Container) []scalingv1.ContainerResources {
	resources := make([]scalingv1.ContainerResources, 0)

	for _, container := range containers {
		requests, limits := container.Resources.Requests, container.Resources.Limits

		containerResources := scalingv1.ContainerResources{
			Name:     container.Name,
			Requests: requests,
			Limits:   limits,
		}

		resources = append(resources, containerResources)
	}

	return resources
}

func prepareState(status scalingv1.HybridScalerStatus) (*strategy.State, error) {
	containerResources := make(map[string]strategy.Resources)

	for _, resources := range status.ContainerResources {
		cpuRequests := resources.Requests.Cpu().AsDec()
		memoryRequests := resources.Requests.Memory().AsDec()
		cpuLimits := resources.Limits.Cpu().AsDec()
		memoryLimits := resources.Limits.Memory().AsDec()

		containerResources[resources.Name] = strategy.Resources{
			Requests: strategy.ResourcesList{
				CPU:    cpuRequests,
				Memory: memoryRequests,
			},
			Limits: strategy.ResourcesList{
				CPU:    cpuLimits,
				Memory: memoryLimits,
			},
		}
	}

	containerMetrics := make([]strategy.ContainerMetrics, 0)

	for _, metrics := range status.ContainerMetrics {
		cpuUsage := metrics.Usage.Cpu().AsDec()
		memoryUsage := metrics.Usage.Cpu().AsDec()

		resources, ok := containerResources[metrics.Name]
		if !ok {
			return nil, fmt.Errorf("unable to calculate resource usage of container %s", metrics.Name)
		}

		cpuLimits := resources.Limits.CPU
		memoryLimits := resources.Limits.Memory
		zero := new(inf.Dec)

		cpuPercentage := &inf.Dec{}
		memoryPercentage := &inf.Dec{}

		if cpuLimits.Cmp(zero) != 0 {
			cpuPercentage.QuoRound(cpuUsage, cpuLimits, 8, inf.RoundHalfUp)
		}

		if memoryLimits.Cmp(zero) != 0 {
			memoryPercentage.QuoRound(memoryUsage, memoryLimits, 8, inf.RoundHalfUp)
		}

		containerMetrics = append(containerMetrics, strategy.ContainerMetrics{
			Name: metrics.Name,
			ResourceUsage: strategy.ResourceUsage{
				CPU:    cpuPercentage,
				Memory: memoryPercentage,
			},
		})
	}

	state := &strategy.State{
		Replicas:           status.Replicas,
		ContainerResources: containerResources,
		ContainerMetrics:   containerMetrics,
	}

	return state, nil
}

func interpretResourceScaling(decision *strategy.ScalingDecision) []scalingv1.ContainerResources {
	containerResources := make([]scalingv1.ContainerResources, 0)

	for name, resources := range decision.ContainerResources {
		requests := make(corev1.ResourceList)
		limits := make(corev1.ResourceList)

		requests[corev1.ResourceCPU] = *resource.NewDecimalQuantity(*resources.Requests.CPU, resource.DecimalSI)
		requests[corev1.ResourceMemory] = *resource.NewDecimalQuantity(*resources.Requests.Memory, resource.DecimalSI)
		limits[corev1.ResourceCPU] = *resource.NewDecimalQuantity(*resources.Limits.CPU, resource.DecimalSI)
		limits[corev1.ResourceMemory] = *resource.NewDecimalQuantity(*resources.Limits.Memory, resource.DecimalSI)

		containerResources = append(containerResources, scalingv1.ContainerResources{
			Name:     name,
			Requests: requests,
			Limits:   limits,
		})
	}

	return containerResources
}
