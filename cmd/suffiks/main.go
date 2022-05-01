package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/logr"
	"github.com/suffiks/suffiks"
	suffiksv1 "github.com/suffiks/suffiks/api/v1"
	"github.com/suffiks/suffiks/base"
	"github.com/suffiks/suffiks/base/tracing"
	"github.com/suffiks/suffiks/controllers"
	"github.com/suffiks/suffiks/docparser"
	"github.com/suffiks/suffiks/extension/protogen"
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
	var configFile string
	flag.StringVar(&configFile, "config-file", "", "Path to the configuration file.")

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

	var err error
	ctrlConfig := suffiksv1.ProjectConfig{}
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(&ctrlConfig))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

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

	if !ctrlConfig.WebhooksDisabled {
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
	err = tracing.Provider(ctx, tracerLog, ctrlConfig.Tracing)
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

func documentationServer(ctx context.Context, addr string, mgr *base.ExtensionManager, log logr.Logger) {
	ctrl := docparser.NewController()
	go updateDocs(ctx, ctrl, mgr, log)

	mux := http.NewServeMux()

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

	log.V(3).Info("serving documentation", "addr", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Error(err, "problem running server")
	}
}

func updateDocs(ctx context.Context, ctrl *docparser.Controller, mgr *base.ExtensionManager, log logr.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute * 3):
			for _, ext := range mgr.All() {
				pages, err := ext.Client().Documentation(ctx, &protogen.DocumentationRequest{})
				if err != nil {
					log.V(5).Error(err, "unable to get documentation")
					continue
				}
				ctrl.Parse(ext.Name, pages.GetPages())
			}
		}
	}
}
