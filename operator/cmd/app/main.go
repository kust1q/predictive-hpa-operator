package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	autoscalingv1alpha1 "github.com/kust1q/predictive-hpa-operator/api/v1alpha1"
	"github.com/kust1q/predictive-hpa-operator/internal/config"
	"github.com/kust1q/predictive-hpa-operator/internal/controller"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(autoscalingv1alpha1.AddToScheme(scheme))
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// --- Configs ---
	if err := config.InitConfig(); err != nil {
		logrus.WithError(err).Fatal("error initializing config")
	}
	cfg := config.Get()

	// --- Infrastructure ---
	logrus.Info("Starting Predictive HPA Operator")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: cfg.MetricsAddr,
		},
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         cfg.EnableLeaderElection,
		LeaderElectionID:       "b920b44e.predictive-hpa.io",
	})
	if err != nil {
		logrus.WithError(err).Fatal("unable to start manager")
	}

	// --- Connection Logic ---
	if err := (&controller.PredictiveHPAReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logrus.WithError(err).Fatal("unable to create controller")
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logrus.WithError(err).Fatal("unable to set up health check")
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logrus.WithError(err).Fatal("unable to set up ready check")
	}

	// Graceful shutdown setup (controller-runtime handles this, but we can be explicit)
	ctx := ctrl.SetupSignalHandler()

	go func() {
		logrus.Info("Starting manager")
		if err := mgr.Start(ctx); err != nil {
			logrus.WithError(err).Fatal("problem running manager")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down operator...")
}
