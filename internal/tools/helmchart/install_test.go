//go:build integration
// +build integration

package helmchart

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	cacheddiscovery "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

func TestInstall(t *testing.T) {
	helmClient := newHelmClient()
	ctx := context.TODO()
	rc, err := newRestConfig()
	if err != nil {
		t.Fatal(err)
	}

	uns, err := getUnstructured(rc, getUnstructuredOptions{
		gvk:       schema.FromAPIVersionAndKind("composition.krateo.io/v0-1-0", "FireworksApp"),
		name:      "test-1",
		namespace: "demo-system",
	})

	// Create a dummy resource
	res := createDummyResource()

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(rc)
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(
		cacheddiscovery.NewMemCacheClient(discoveryClient),
	)

	// Set up the install options
	opts := InstallOptions{
		HelmClient: helmClient,
		ChartName:  "https://charts.bitnami.com/bitnami",
		Resource:   res,
		Repo:       "postgresql",
		Version:    "12.8.3",
		CheckResourceOptions: CheckResourceOptions{
			DiscoveryClient: nil,
		},
	}

	// Call the Install function
	rel, _, err := Install(ctx, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check the returned release
	expectedRelease := &release.Release{
		Name:      "demo",
		Namespace: "demo-system",
		Version:   1,
	}

	if rel.Name != expectedRelease.Name || rel.Namespace != expectedRelease.Namespace || rel.Version != expectedRelease.Version {
		t.Fatalf("expected release %v, got %v", expectedRelease, rel)
	}
}

func newRestConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
}
