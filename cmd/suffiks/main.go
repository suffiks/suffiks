package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/logr"
	"github.com/suffiks/suffiks"
	"github.com/suffiks/suffiks/internal/controller"
	"github.com/suffiks/suffiks/internal/docparser"
	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/tracing"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	oteltrace "go.opentelemetry.io/otel/trace"
	uzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer stop()

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
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
			CallerKey:  "c",
			MessageKey: "m",
			LevelKey:   "l",
			TimeKey:    "t",
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

	var err error
	// TODO: Re-introduce config file support
	// ctrlConfig := suffiksv1.ProjectConfig{}
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "operator.suffiks.com",
	}
	// if configFile != "" {
	// 	options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
	// 	if err != nil {
	// 		setupLog.Error(err, "unable to load the config file")
	// 		os.Exit(1)
	// 	}
	// }

	cfg := ctrl.GetConfigOrDie()
	cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(rt, otelhttp.WithFilter(func(r *http.Request) bool {
			span := oteltrace.SpanFromContext(r.Context())
			return span.IsRecording()
		}))
	}
	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	grpcOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	}

	dynClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create dynamic client")
		os.Exit(1)
	}

	crdMgr, err := extension.NewExtensionManager(ctx, suffiks.CRDFiles, dynClient, extension.WithGRPCOptions(grpcOptions...))
	if err != nil {
		setupLog.Error(err, "unable to create CRD manager")
		os.Exit(1)
	}

	extController := controller.NewExtensionController(crdMgr)

	if err := extController.RegisterMetrics(metrics.Registry); err != nil {
		setupLog.Error(err, "unable to register CRD metrics")
		os.Exit(1)
	}

	extRec := &controller.ExtensionReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		KubeConfig: cfg,
		CRDManager: crdMgr,
	}
	if err = (extRec).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Extension")
		os.Exit(1)
	}

	appRec := &controller.ReconcilerWrapper[*suffiksv1.Application]{
		Client: mgr.GetClient(),
		Child: &controller.AppReconciler{
			Scheme: mgr.GetScheme(),
			Client: mgr.GetClient(),
			// Defaults: ctrlConfig.ApplicationDefaults,
		},
		CRDController: extController,
	}
	if err = appRec.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}

	// workRec := &controllers.ReconcilerWrapper[*base.Work]{
	// 	Client: mgr.GetClient(),
	// 	Child: &controllers.JobReconciler{
	// 		Scheme: mgr.GetScheme(),
	// 		Client: mgr.GetClient(),
	// 	},
	// 	CRDController: extController,
	// }
	// if err = workRec.SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "Application")
	// 	os.Exit(1)
	// }

	if true { //!ctrlConfig.WebhooksDisabled {
		if err = (&suffiksv1.Extension{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Extension")
			os.Exit(1)
		}

		if err := appRec.SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Application")
			os.Exit(1)
		}

		// if err := workRec.SetupWebhookWithManager(mgr); err != nil {
		// 	setupLog.Error(err, "unable to create webhook", "webhook", "Work")
		// 	os.Exit(1)
		// }
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

	if err := extRec.RefreshCRD(ctx); err != nil {
		panic(err)
	}

	tracerLog := ctrl.Log.WithName("tracing")
	err = tracing.Provider(ctx, tracerLog) // ctrlConfig.Tracing)
	if err != nil {
		setupLog.Error(err, "unable to create tracer provider")
		os.Exit(1)
	}

	go documentationServer(ctx, "" /*ctrlConfig.DocumentationAddress*/, crdMgr, setupLog)
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

func documentationServer(ctx context.Context, addr string, mgr *extension.ExtensionManager, log logr.Logger) {
	fmt.Println("################ STARTING DOCUMENTATION SERVER ################")
	ctrl := docparser.NewController()
	_ = ctrl.AddFS("_suffiks", suffiks.DocFiles)
	go updateDocs(ctx, ctrl, mgr, log)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cats := ctrl.GetAll()
		_ = json.NewEncoder(w).Encode(cats)
	})

	if addr == "" {
		addr = ":8084"
	}

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(ctx); err != nil {
			log.Error(err, "problem shutting down server")
		}
	}()

	log.Info("serving documentation", "addr", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Error(err, "problem running server")
	}
}

func updateDocs(ctx context.Context, ctrl *docparser.Controller, mgr *extension.ExtensionManager, log logr.Logger) {
	for {
		for _, ext := range mgr.All() {
			fmt.Println("Start update docs for", ext.Name())
			pages, err := ext.Documentation(ctx)
			if err != nil {
				log.V(5).Error(err, "unable to get documentation")
				continue
			}

			fmt.Printf("Got %v pages from %v\n", len(pages.Pages), ext.Name())
			_ = ctrl.Parse(ext.Name(), pages.GetPages())
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute * 3):
		}
	}
}
