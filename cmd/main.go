package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/lhns/kube-traefik-reflector/internal/controller"
)

// Stamped via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		metricsAddr   string
		probeAddr     string
		enableLeader  bool
		excludedNS    string
		sweepInterval time.Duration
		showVersion   bool
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Address the metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "Address the health probe endpoint binds to.")
	flag.BoolVar(&enableLeader, "leader-elect", false, "Enable leader election for controller manager.")
	flag.StringVar(&excludedNS, "excluded-namespaces", strings.Join(controller.DefaultExcludedNamespaces, ","),
		"Comma-separated namespaces skipped during '*'/selector expansion (explicit names still honored).")
	flag.DurationVar(&sweepInterval, "orphan-sweep-interval", 10*time.Minute,
		"How often to sweep for mirrors whose source no longer exists.")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit.")
	opts := zap.Options{Development: false}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	if showVersion {
		fmt.Printf("traefik-reflector %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	setupLog := ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	for _, rt := range controller.ReflectedTypes {
		scheme.AddKnownTypeWithName(rt.GVK, &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(rt.GVK.GroupVersion().WithKind(rt.GVK.Kind+"List"), &unstructured.UnstructuredList{})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeader,
		LeaderElectionID:       "kube-traefik-reflector.lhns.de",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	excluded := map[string]bool{}
	for _, n := range strings.Split(excludedNS, ",") {
		if n = strings.TrimSpace(n); n != "" {
			excluded[n] = true
		}
	}

	// One reconciler per Traefik CRD that is actually installed. A missing CRD is
	// logged and skipped rather than crashing the manager.
	var active []controller.ReflectedType
	for _, rt := range controller.ReflectedTypes {
		if _, err := mgr.GetRESTMapper().RESTMapping(rt.GVK.GroupKind(), rt.GVK.Version); err != nil {
			setupLog.Info("Traefik CRD not installed; skipping", "kind", rt.GVK.Kind)
			continue
		}
		r := &controller.ReflectorReconciler{
			Client:    mgr.GetClient(),
			APIReader: mgr.GetAPIReader(),
			Recorder:  mgr.GetEventRecorderFor("traefik-reflector"),
			Type:      rt,
			Excluded:  excluded,
		}
		if err := r.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "kind", rt.GVK.Kind)
			os.Exit(1)
		}
		active = append(active, rt)
	}
	if len(active) == 0 {
		setupLog.Error(fmt.Errorf("no Traefik CRDs found"), "nothing to reconcile; are the Traefik CRDs installed?")
		os.Exit(1)
	}

	if err := mgr.Add(&controller.OrphanSweeper{
		Client:   mgr.GetClient(),
		Types:    active,
		Interval: sweepInterval,
	}); err != nil {
		setupLog.Error(err, "unable to add orphan sweeper")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting traefik-reflector", "version", version, "reflectedKinds", kinds(active))
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func kinds(types []controller.ReflectedType) []string {
	out := make([]string, 0, len(types))
	for _, t := range types {
		out = append(out, t.GVK.Kind)
	}
	return out
}
