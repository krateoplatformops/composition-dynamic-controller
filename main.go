package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/krateoplatformops/composition-dynamic-controller/internal/composition"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/support"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/helmchart/archive"
	genctrl "github.com/krateoplatformops/unstructured-runtime"
	"github.com/krateoplatformops/unstructured-runtime/pkg/controller"
	"github.com/krateoplatformops/unstructured-runtime/pkg/eventrecorder"
	"github.com/krateoplatformops/unstructured-runtime/pkg/logging"
	"github.com/krateoplatformops/unstructured-runtime/pkg/pluralizer"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	serviceName             = "composition-dynamic-controller"
	compositionVersionLabel = "krateo.io/composition-parent-version"
)

var (
	Build string
)

func main() {
	// Flags
	kubeconfig := flag.String("kubeconfig", support.EnvString("KUBECONFIG", ""),
		"absolute path to the kubeconfig file")
	debug := flag.Bool("debug",
		support.EnvBool("COMPOSITION_CONTROLLER_DEBUG", false), "dump verbose output")
	workers := flag.Int("workers", support.EnvInt("COMPOSITION_CONTROLLER_WORKERS", 1), "number of workers")
	resyncInterval := flag.Duration("resync-interval",
		support.EnvDuration("COMPOSITION_CONTROLLER_RESYNC_INTERVAL", time.Minute*3), "resync interval")
	resourceGroup := flag.String("group",
		support.EnvString("COMPOSITION_CONTROLLER_GROUP", ""), "resource api group")
	resourceVersion := flag.String("version",
		support.EnvString("COMPOSITION_CONTROLLER_VERSION", ""), "resource api version")
	resourceName := flag.String("resource",
		support.EnvString("COMPOSITION_CONTROLLER_RESOURCE", ""), "resource plural name")
	namespace := flag.String("namespace",
		support.EnvString("COMPOSITION_CONTROLLER_NAMESPACE", "default"), "namespace")
	chart := flag.String("chart",
		support.EnvString("COMPOSITION_CONTROLLER_CHART", ""), "chart")
	urlplurals := flag.String("urlplurals",
		support.EnvString("URL_PLURALS", "http://bff.krateo-system.svc.cluster.local:8081/api-info/names"), "url plurals")

	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Flags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName(serviceName))
	// Kubernetes configuration
	var cfg *rest.Config
	var err error
	if len(*kubeconfig) > 0 {
		cfg, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		log.Debug("Building kubeconfig.", "error", err)
	}

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Debug("Creating dynamic client.", "error", err)
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		log.Debug("Creating discovery client.", "error", err)
	}

	cachedDisc := memory.NewMemCacheClient(discovery)

	rec, err := eventrecorder.Create(cfg)
	if err != nil {
		log.Debug("Creating event recorder.", "error", err)
	}

	var pig archive.Getter
	if len(*chart) > 0 {
		pig = archive.Static(*chart)
	} else {
		pig, err = archive.Dynamic(cfg, *debug, log)
		if err != nil {
			log.Debug("Creating chart url info getter.", "error", err)
		}
	}

	log.WithValues("build", Build).
		WithValues("debug", *debug).
		WithValues("resyncInterval", *resyncInterval).
		WithValues("group", *resourceGroup).
		WithValues("version", *resourceVersion).
		WithValues("resource", *resourceName).
		Info("Starting composition dynamic controller.")

	// Create a label requirement for the composition version
	labelreq, err := labels.NewRequirement(compositionVersionLabel, selection.Equals, []string{*resourceVersion})
	if err != nil {
		log.Debug("Creating label requirement.", "error", err)
	}
	labelselector := labels.NewSelector().Add(*labelreq)

	pluralizer := pluralizer.New(urlplurals, http.DefaultClient)
	handler := composition.NewHandler(cfg, log, pig, rec, dyn, cachedDisc, *pluralizer)

	controller := genctrl.New(genctrl.Options{
		Discovery:      cachedDisc,
		Client:         dyn,
		ResyncInterval: *resyncInterval,
		GVR: schema.GroupVersionResource{
			Group:    *resourceGroup,
			Version:  *resourceVersion,
			Resource: *resourceName,
		},
		Namespace:    *namespace,
		Config:       cfg,
		Debug:        *debug,
		Logger:       log,
		ProviderName: serviceName,
		ListWatcher: controller.ListWatcherConfiguration{
			LabelSelector: ptr.To(labelselector.String()),
		},
		Pluralizer: *pluralizer,
	})
	controller.SetExternalClient(handler)

	ctx, cancel := signal.NotifyContext(context.Background(), []os.Signal{
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGHUP,
		syscall.SIGQUIT,
	}...)
	defer cancel()

	err = controller.Run(ctx, *workers)
	if err != nil {
		log.Debug("Running controller.", "error", err)
	}
}
