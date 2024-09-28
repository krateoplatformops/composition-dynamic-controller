package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/controller/objectref"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/listwatcher"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/meta"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/shortid"
	unstructuredtools "github.com/krateoplatformops/composition-dynamic-controller/internal/tools/unstructured"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/unstructured/condition"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const (
	reasonReconciliationPaused string = "ReconciliationPaused"
)

type Options struct {
	Client         dynamic.Interface
	Discovery      discovery.DiscoveryInterface
	GVR            schema.GroupVersionResource
	Namespace      string
	ResyncInterval time.Duration
	Recorder       record.EventRecorder
	Logger         *zerolog.Logger
	ExternalClient ExternalClient
}

type Controller struct {
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	gvr             schema.GroupVersionResource
	queue           workqueue.RateLimitingInterface
	indexer         cache.Indexer
	informer        cache.Controller
	recorder        record.EventRecorder
	logger          *zerolog.Logger
	externalClient  ExternalClient
}

func New(sid *shortid.Shortid, opts Options) *Controller {
	rateLimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(3*time.Second, 180*time.Second),
		// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)

	queue := workqueue.NewRateLimitingQueue(rateLimiter)

	lw, err := listwatcher.Create(listwatcher.CreateOptions{
		Discovery: opts.Discovery,
		Client:    opts.Client,
		GVR:       opts.GVR,
		Namespace: opts.Namespace,
	})
	if err != nil {
		opts.Logger.Error().Err(err).Msg("Failed to create listwatcher.")
		return nil
	}
	indexer, informer := cache.NewIndexerInformer(
		lw,
		&unstructured.Unstructured{},
		opts.ResyncInterval,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				el, ok := obj.(*unstructured.Unstructured)
				if !ok {
					opts.Logger.Warn().Msg("AddFunc: object is not an unstructured.")
					return
				}

				id, err := sid.Generate()
				if err != nil {
					opts.Logger.Error().Err(err).Msg("AddFunc: generating short id.")
					return
				}

				queue.Add(event{
					id:        id,
					eventType: Observe,
					objectRef: objectref.ObjectRef{
						APIVersion: el.GetAPIVersion(),
						Kind:       el.GetKind(),
						Name:       el.GetName(),
						Namespace:  el.GetNamespace(),
					},
				})
			},
			UpdateFunc: func(old, new interface{}) {
				oldUns, ok := old.(*unstructured.Unstructured)
				if !ok {
					opts.Logger.Warn().Msg("UpdateFunc: object is not an unstructured.")
					return
				}

				newUns, ok := new.(*unstructured.Unstructured)
				if !ok {
					opts.Logger.Warn().Msg("UpdateFunc: object is not an unstructured.")
					return
				}

				id, err := sid.Generate()
				if err != nil {
					opts.Logger.Error().Err(err).Msg("UpdateFunc: generating short id.")
					return
				}

				newSpec, _, err := unstructured.NestedMap(newUns.Object, "spec")
				if err != nil {
					opts.Logger.Error().Err(err).Msg("UpdateFunc: getting new object spec.")
					return
				}

				oldSpec, _, err := unstructured.NestedMap(oldUns.Object, "spec")
				if err != nil {
					opts.Logger.Error().Err(err).Msg("UpdateFunc: getting old object spec.")
				}

				diff := cmp.Diff(newSpec, oldSpec)
				opts.Logger.Debug().Str("diff", diff).Msg("UpdateFunc: comparing current spec with desired spec")
				if len(diff) > 0 {
					queue.Add(event{
						id:        id,
						eventType: Update,
						objectRef: objectref.ObjectRef{
							APIVersion: newUns.GetAPIVersion(),
							Kind:       newUns.GetKind(),
							Name:       newUns.GetName(),
							Namespace:  newUns.GetNamespace(),
						},
					})
				} else {
					queue.AddAfter(event{
						id:        id,
						eventType: Observe,
						objectRef: objectref.ObjectRef{
							APIVersion: newUns.GetAPIVersion(),
							Kind:       newUns.GetKind(),
							Name:       newUns.GetName(),
							Namespace:  newUns.GetNamespace(),
						},
					}, opts.ResyncInterval)
				}
			},
			DeleteFunc: func(obj interface{}) {
				el, ok := obj.(*unstructured.Unstructured)
				if !ok {
					opts.Logger.Warn().Msg("DeleteFunc: object is not an unstructured.")
					return
				}

				if meta.IsPaused(el) {
					opts.Logger.Debug().Msgf("Reconciliation is paused via the pause annotation %s: %s; %s: %s", "annotation", meta.AnnotationKeyReconciliationPaused, "value", "true")
					opts.Recorder.Event(el, corev1.EventTypeNormal, reasonReconciliationPaused, "Reconciliation is paused via the pause annotation")
					unstructuredtools.SetCondition(el, condition.ReconcilePaused())
					// if the pause annotation is removed, we will have a chance to reconcile again and resume
					// and if status update fails, we will reconcile again to retry to update the status
					return
				}

				id, err := sid.Generate()
				if err != nil {
					opts.Logger.Error().Err(err).Msg("DeleteFunc: generating short id.")
					return
				}

				queue.Add(event{
					id:        id,
					eventType: Delete,
					objectRef: objectref.ObjectRef{
						APIVersion: el.GetAPIVersion(),
						Kind:       el.GetKind(),
						Name:       el.GetName(),
						Namespace:  el.GetNamespace(),
					},
				})
			},
		},
		cache.Indexers{},
	)

	return &Controller{
		dynamicClient:   opts.Client,
		discoveryClient: opts.Discovery,
		gvr:             opts.GVR,
		recorder:        opts.Recorder,
		logger:          opts.Logger,
		informer:        informer,
		indexer:         indexer,
		queue:           queue,
		externalClient:  opts.ExternalClient,
	}
}

func (c *Controller) SetExternalClient(ec ExternalClient) {
	c.externalClient = ec
}

// Run begins watching and syncing.
func (c *Controller) Run(ctx context.Context, numWorkers int) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info().Msg("Starting controller")
	go c.informer.Run(ctx.Done())

	// Wait for all involved caches to be synced, before
	// processing items from the queue is started
	c.logger.Info().Msg("waiting for informer caches to sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		err := fmt.Errorf("failed to wait for informers caches to sync")
		utilruntime.HandleError(err)
		return err
	}

	c.logger.Info().Int("workers", numWorkers).Msg("Starting workers.")
	for i := 0; i < numWorkers; i++ {
		go wait.Until(func() {
			c.runWorker(ctx)
		}, 2*time.Second, ctx.Done())
	}
	c.logger.Info().Msg("Controller ready.")

	<-ctx.Done()
	c.logger.Info().Msg("Stopping controller.")

	return nil
}
