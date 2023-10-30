package main

import (
	"os"

	hs "github.com/iljarotar/hybrid-scaler/internal/controller"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log    = logf.Log.WithName("hybridscaler-controller")
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(appsv1.AddToScheme(scheme))
	logf.SetLogger(zap.New())
}

func main() {
	log.Info("starting hybridscaler controller")

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		os.Exit(1)
	}

	hybridScalerController, err := controller.New("hybridscaler-controller", mgr, controller.Options{
		Reconciler: &hs.HybridScalerReconciler{
			Client: mgr.GetClient(),
		},
	})
	if err != nil {
		log.Error(err, "unable to register controller")
		os.Exit(1)
	}

	err = hybridScalerController.Watch(source.Kind(mgr.GetCache(), &appsv1.Deployment{}), &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Error(err, "unable to watch deployments")
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to continue running manager")
		os.Exit(1)
	}
}
