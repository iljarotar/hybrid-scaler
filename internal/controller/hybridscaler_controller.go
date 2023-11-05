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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	scalingv1 "github.com/iljarotar/hybrid-scaler/api/v1"
)

var (
	ownerKey    = ".metadata.controller"
	apiGVString = scalingv1.GroupVersion.String()
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

	logger.Info("reconcile", "req", req)

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

	requests, limits := getResourcesFromContainers(deployment.Spec.Template.Spec.Containers)
	scaler.Status.Requests = requests
	scaler.Status.Limits = limits

	// TODO: get current cpu and memory utilization of all pods and calculate their entropy

	// TODO: find a way to get an average of the response time of all pods or maybe find a better benchmark for measuring performance

	logger.Info("reconcile", "scaler status", scaler.Status)

	return ctrl.Result{RequeueAfter: requeuePeriod}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HybridScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&scalingv1.HybridScaler{}).
		Complete(r)
}

func getResourcesFromContainers(containers []corev1.Container) (requests, limits corev1.ResourceList) {
	var cpuRequests, cpuLimits, memoryRequests, memoryLimits resource.Quantity

	for _, container := range containers {
		resources := container.Resources

		cpuRequests.Add(*resources.Requests.Cpu())
		memoryRequests.Add(*resources.Requests.Memory())
		cpuLimits.Add(*resources.Limits.Cpu())
		memoryLimits.Add(*resources.Limits.Memory())
	}

	requests, limits = make(corev1.ResourceList), make(corev1.ResourceList)

	requests[corev1.ResourceCPU] = cpuRequests
	requests[corev1.ResourceMemory] = memoryRequests
	limits[corev1.ResourceCPU] = cpuLimits
	limits[corev1.ResourceMemory] = memoryLimits

	return requests, limits
}
