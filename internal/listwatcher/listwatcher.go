package listwatcher

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

const (
	CompositionVersionLabel = "krateo.io/composition-version"
)

type CreateOptions struct {
	Client    dynamic.Interface
	Discovery discovery.DiscoveryInterface
	GVR       schema.GroupVersionResource
	Namespace string
}

func Create(opts CreateOptions) (*cache.ListWatch, error) {
	// Create a label requirement for the composition version
	labelreq, err := labels.NewRequirement(CompositionVersionLabel, selection.Equals, []string{opts.GVR.Version})
	if err != nil {
		return nil, err
	}
	selector := labels.NewSelector()
	selector = selector.Add(*labelreq)

	return &cache.ListWatch{
		ListFunc: func(lo metav1.ListOptions) (runtime.Object, error) {
			lo.LabelSelector = selector.String()
			return opts.Client.Resource(opts.GVR).
				Namespace(opts.Namespace).
				List(context.Background(), lo)
		},
		WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
			lo.LabelSelector = selector.String()
			return opts.Client.Resource(opts.GVR).
				Namespace(opts.Namespace).
				Watch(context.Background(), lo)
		},
	}, nil
}
