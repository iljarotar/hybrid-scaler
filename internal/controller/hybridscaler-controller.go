package controller

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type HybridScalerReconciler struct {
	Client client.Client
}

func (r *HybridScalerReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	res := reconcile.Result{}
	return res, nil
}
