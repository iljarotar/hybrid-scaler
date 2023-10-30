package main

import (
	"os"

	v1 "github.com/iljarotar/hybrid-scaler/api/v1"
	hs "github.com/iljarotar/hybrid-scaler/internal/controller"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	log    = ctrl.Log.WithName("hybridscaler-controller")
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
}

func main() {
	ctrl.SetLogger(zap.New())
	log.Info("starting hybridscaler controller")

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		os.Exit(1)
	}

	err = (&hs.HybridScalerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		log.Error(err, "unable to register controller")
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "unable to continue running manager")
		os.Exit(1)
	}
}
