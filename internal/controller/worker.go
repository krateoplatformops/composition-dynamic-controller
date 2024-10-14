package controller

import (
	"context"
	"fmt"

	"github.com/krateoplatformops/composition-dynamic-controller/internal/controller/objectref"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/meta"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools"
	unstructuredtools "github.com/krateoplatformops/composition-dynamic-controller/internal/tools/unstructured"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/unstructured/condition"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/runtime"
)

const (
	maxRetries = 5
)

func (c *Controller) runWorker(ctx context.Context) {
	for {
		obj, shutdown := c.queue.Get()
		if shutdown {
			break
		}
		defer c.queue.Done(obj)

		err := c.processItem(ctx, obj)
		c.handleErr(err, obj)
	}
}

func (c *Controller) handleErr(err error, obj interface{}) {
	if err == nil {
		c.queue.Forget(obj)
		return
	}

	if retries := c.queue.NumRequeues(obj); retries < maxRetries {
		c.logger.Warn().Int("retries", retries).
			Str("obj", fmt.Sprintf("%v", obj)).
			Msgf("error processing event: %v, retrying", err)
		c.queue.AddRateLimited(obj)
		return
	}

	c.logger.Err(err).Msg("error processing event (max retries reached)")
	c.queue.Forget(obj)
	runtime.HandleError(err)
}

func (c *Controller) processItem(ctx context.Context, obj interface{}) error {
	evt, ok := obj.(event)
	if !ok {
		c.logger.Error().Msgf("unexpected event: %v", obj)
		return nil
	}

	c.logger.Debug().Str("event", string(evt.eventType)).Str("ref", evt.objectRef.String()).Msg("processing")
	switch evt.eventType {
	case Create:
		return c.handleCreate(ctx, evt.objectRef)
	case Update:
		return c.handleUpdateEvent(ctx, evt.objectRef)
	case Delete:
		return c.handleDeleteEvent(ctx, evt.objectRef)
	default:
		return c.handleObserve(ctx, evt.objectRef)
	}
}

func (c *Controller) handleObserve(ctx context.Context, ref objectref.ObjectRef) error {
	if c.externalClient == nil {
		c.logger.Warn().
			Str("eventType", string(Observe)).
			Msg("No event handler registered.")
		return nil
	}

	el, err := c.fetch(ctx, ref, false)
	if err != nil {
		c.logger.Err(err).
			Str("objectRef", ref.String()).
			Msg("Resolving unstructured object.")
		return err
	}

	if meta.IsPaused(el) {
		c.logger.Debug().Msgf("Reconciliation is paused via the pause annotation %s: %s; %s: %s", "annotation", meta.AnnotationKeyReconciliationPaused, "value", "true")
		c.recorder.Event(el, corev1.EventTypeNormal, reasonReconciliationPaused, "Reconciliation is paused via the pause annotation")
		err = unstructuredtools.SetCondition(el, condition.ReconcilePaused())
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: setting condition.")
			return err
		}

		_, err := c.dynamicClient.Resource(c.gvr).Namespace(el.GetNamespace()).UpdateStatus(context.Background(), el, metav1.UpdateOptions{})
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: updating status.")
			return err
		}
		// if the pause annotation is removed, we will have a chance to reconcile again and resume
		// and if status update fails, we will reconcile again to retry to update the status
		return nil
	}

	exists, actionErr := c.externalClient.Observe(ctx, el)
	if actionErr != nil {
		if apierrors.IsNotFound(actionErr) {
			c.queue.Add(event{
				eventType: Update,
				objectRef: ref,
			})
			return nil
		}

		e, err := c.fetch(ctx, ref, false)
		if err != nil {
			c.logger.Err(err).
				Str("objectRef", ref.String()).
				Msg("Resolving unstructured object.")
			return err
		}

		err = unstructuredtools.SetCondition(e, condition.FailWithReason(fmt.Sprintf("failed to observe object: %s", actionErr)))
		if err != nil {
			c.logger.Error().Err(err).Msg("Observe: setting condition.")
			return err
		}

		_, err = tools.UpdateStatus(ctx, e, tools.UpdateOptions{
			DiscoveryClient: c.discoveryClient,
			DynamicClient:   c.dynamicClient,
		})
		if err != nil {
			c.logger.Error().Err(err).Msg("Observe: updating status.")
			return err
		}

		return actionErr
	}

	if !exists {
		// return c.externalClient.Create(ctx, el)
		// fmt.Println("Creating object")

		// rateLimiter := workqueue.NewMaxOfRateLimiter(
		// 	workqueue.NewItemExponentialFailureRateLimiter(3*time.Second, 180*time.Second),
		// 	// 10 qps, 100 bucket size.  This is only for retry speed and its only the overall factor (not per item)
		// 	&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
		// )

		// c.queue = workqueue.NewRateLimitingQueue(rateLimiter)
		// c.queue.AddRateLimited(event{
		// 	eventType: Create,
		// 	objectRef: ref,
		// })

		c.handleCreate(ctx, ref)

		// len := c.queue.Len()
		// for {
		// 	if len < c.queue.Len() {
		// 		// fmt.Println("Queue length changed", len, c.queue.Len())
		// 		break
		// 	}
		// 	// fmt.Println("Adding object to queue", len, c.queue.Len())
		// 	c.queue.AddRateLimited(event{
		// 		eventType: Create,
		// 		objectRef: ref,
		// 	})
		// }

		// fmt.Println("Object created", c.queue.Len())

	}

	return nil
}

func (c *Controller) handleCreate(ctx context.Context, ref objectref.ObjectRef) error {
	if c.externalClient == nil {
		c.logger.Warn().
			Str("eventType", string(Create)).
			Msg("No event handler registered.")
		return nil
	}

	// fmt.Println("Creating object - handleCreate")

	el, err := c.fetch(ctx, ref, false)
	if err != nil {
		c.logger.Err(err).
			Str("objectRef", ref.String()).
			Msg("Resolving unstructured object.")
		return err
	}

	if meta.IsPaused(el) {
		c.logger.Debug().Msgf("Reconciliation is paused via the pause annotation %s: %s; %s: %s", "annotation", meta.AnnotationKeyReconciliationPaused, "value", "true")
		// opts.Recorder.Event(newUns, corev1.EventTypeNormal, reasonReconciliationPaused, "Reconciliation is paused via the pause annotation")
		err = unstructuredtools.SetCondition(el, condition.ReconcilePaused())
		if err != nil {
			c.logger.Error().Err(err).Msg("CreateFunc: setting condition.")
			return err
		}

		_, err := c.dynamicClient.Resource(c.gvr).Namespace(el.GetNamespace()).UpdateStatus(context.Background(), el, metav1.UpdateOptions{})
		if err != nil {
			c.logger.Error().Err(err).Msg("CreateFunc: updating status.")
			return err
		}
		// if the pause annotation is removed, we will have a chance to reconcile again and resume
		// and if status update fails, we will reconcile again to retry to update the status
		return nil
	}

	actionErr := c.externalClient.Create(ctx, el)

	if actionErr != nil {
		e, err := c.fetch(ctx, ref, false)
		if err != nil {
			c.logger.Err(err).
				Str("objectRef", ref.String()).
				Msg("Resolving unstructured object.")
			return err
		}

		err = unstructuredtools.SetCondition(e, condition.FailWithReason(fmt.Sprintf("failed to create object: %s", actionErr)))
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: setting condition.")
			return err
		}

		_, err = tools.UpdateStatus(ctx, e, tools.UpdateOptions{
			DiscoveryClient: c.discoveryClient,
			DynamicClient:   c.dynamicClient,
		})
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: updating status.")
			return err
		}

	}

	return actionErr
}

func (c *Controller) handleUpdateEvent(ctx context.Context, ref objectref.ObjectRef) error {
	if c.externalClient == nil {
		c.logger.Warn().
			Str("eventType", string(Update)).
			Msg("No event handler registered.")
		return nil
	}

	el, err := c.fetch(ctx, ref, false)
	if err != nil {
		c.logger.Err(err).
			Str("objectRef", ref.String()).
			Msg("Resolving unstructured object.")
		return err
	}

	if meta.IsPaused(el) {
		c.logger.Debug().Msgf("Reconciliation is paused via the pause annotation %s: %s; %s: %s", "annotation", meta.AnnotationKeyReconciliationPaused, "value", "true")
		// opts.Recorder.Event(newUns, corev1.EventTypeNormal, reasonReconciliationPaused, "Reconciliation is paused via the pause annotation")
		err = unstructuredtools.SetCondition(el, condition.ReconcilePaused())
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: setting condition.")
			return err
		}

		_, err := c.dynamicClient.Resource(c.gvr).Namespace(el.GetNamespace()).UpdateStatus(context.Background(), el, metav1.UpdateOptions{})
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: updating status.")
			return err
		}
		// if the pause annotation is removed, we will have a chance to reconcile again and resume
		// and if status update fails, we will reconcile again to retry to update the status
		return nil
	}

	actionErr := c.externalClient.Update(ctx, el)

	if actionErr != nil {
		e, err := c.fetch(ctx, ref, false)
		if err != nil {
			c.logger.Err(err).
				Str("objectRef", ref.String()).
				Msg("Resolving unstructured object.")
			return err
		}

		err = unstructuredtools.SetCondition(e, condition.FailWithReason(fmt.Sprintf("failed to update object: %s", actionErr)))
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: setting condition.")
			return err
		}

		_, err = tools.UpdateStatus(ctx, e, tools.UpdateOptions{
			DiscoveryClient: c.discoveryClient,
			DynamicClient:   c.dynamicClient,
		})
		if err != nil {
			c.logger.Error().Err(err).Msg("UpdateFunc: updating status.")
			return err
		}

	}

	return actionErr
}

func (c *Controller) handleDeleteEvent(ctx context.Context, ref objectref.ObjectRef) error {
	if c.externalClient == nil {
		c.logger.Warn().
			Str("eventType", string(Delete)).
			Msg("No event handler registered.")
		return nil
	}

	return c.externalClient.Delete(ctx, ref)
}

func (c *Controller) fetch(ctx context.Context, ref objectref.ObjectRef, clean bool) (*unstructured.Unstructured, error) {
	res, err := c.dynamicClient.Resource(c.gvr).
		Namespace(ref.Namespace).
		Get(ctx, ref.Name, metav1.GetOptions{})
	if err == nil {
		if clean && res != nil {
			unstructured.RemoveNestedField(res.Object,
				"metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")
			unstructured.RemoveNestedField(res.Object, "metadata", "creationTimestamp")
			unstructured.RemoveNestedField(res.Object, "metadata", "generation")
			unstructured.RemoveNestedField(res.Object, "metadata", "uid")
		}
	}
	return res, err
}
