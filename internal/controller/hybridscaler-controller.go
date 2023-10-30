package controller

import (
	"context"

	v1 "github.com/iljarotar/hybrid-scaler/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type HybridScalerReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func (r *HybridScalerReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("reconcile", "request", request)

	res := reconcile.Result{}
	log.Info("reconcile", "response", res)
	return res, nil
}

func (r *HybridScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1.HybridScaler{}).Watches(&appsv1.Deployment{}, &handler.EnqueueRequestForObject{}).Complete(r)
}
