package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/suffiks/suffiks"
	suffiksv1 "github.com/suffiks/suffiks/api/v1"
	"github.com/suffiks/suffiks/base"
	"github.com/suffiks/suffiks/base/tracing"
	"github.com/suffiks/suffiks/controllers"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	oteltrace "go.opentelemetry.io/otel/trace"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	//+kubebuilder:scaffold:imports
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(suffiksv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	tracingAddr := "traces.grafana:4317"
	if ta := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); ta != "" {
		tracingAddr = ta
	}
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8090", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8091", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
		Level:       zapcore.InfoLevel,
		// Encoder: zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		Encoder: zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			EncodeCaller: zapcore.ShortCallerEncoder,
			// StacktraceKey: "stacktrace",
			CallerKey: "caller",
			// FunctionKey: "function",
			// TimeKey:       "time",
		}),
		ZapOpts: []uzap.Option{
			uzap.AddCaller(),
			uzap.AddCallerSkip(0),
			// uzap.AddStacktrace(zapcore.InfoLevel),
		},
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog := ctrl.Log.WithName("setup")

	cfg := ctrl.GetConfigOrDie()
	cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(rt, otelhttp.WithFilter(func(r *http.Request) bool {
			span := oteltrace.SpanFromContext(r.Context())
			return span.IsRecording()
		}))
	}
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0ff08fbf.suffiks.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	grpcOptions := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	}

	crdMgr, err := base.NewExtensionManager(suffiks.CRDFiles, grpcOptions)
	if err != nil {
		setupLog.Error(err, "unable to create CRD manager")
		os.Exit(1)
	}

	extController := base.NewExtensionController(crdMgr)

	if err := extController.RegisterMetrics(metrics.Registry); err != nil {
		setupLog.Error(err, "unable to register CRD metrics")
		os.Exit(1)
	}

	extRec := &controllers.ExtensionReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		KubeConfig: cfg,
		CRDManager: crdMgr,
	}
	if err = (extRec).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Extension")
		os.Exit(1)
	}

	appRec := &controllers.ReconcilerWrapper[*base.Application]{
		Client: mgr.GetClient(),
		Child: &controllers.AppReconciler{
			Scheme: mgr.GetScheme(),
			Client: mgr.GetClient(),
		},
		CRDController: extController,
	}
	if err = appRec.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}

	workRec := &controllers.ReconcilerWrapper[*base.Work]{
		Client: mgr.GetClient(),
		Child: &controllers.JobReconciler{
			Scheme: mgr.GetScheme(),
			Client: mgr.GetClient(),
		},
		CRDController: extController,
	}
	if err = workRec.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&suffiksv1.Extension{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Extension")
			os.Exit(1)
		}

		if err := appRec.SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Application")
			os.Exit(1)
		}

		if err := workRec.SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Work")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	if err := extRec.RefreshCRD(ctx); err != nil {
		panic(err)
	}

	tracerLog := ctrl.Log.WithName("tracing")
	err = tracing.Provider(ctx, tracerLog, "github.com/suffiks/suffiks", "v0.0.1", "local", tracingAddr)
	if err != nil {
		setupLog.Error(err, "unable to create tracer provider")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

	if err := tracing.Shutdown(context.Background()); err != nil {
		setupLog.Error(err, "problem shutting down tracer provider")
		os.Exit(1)
	}
}
