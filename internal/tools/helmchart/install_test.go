package helmchart

import (
	"context"
	"testing"

	"helm.sh/helm/v3/pkg/release"
)

func TestInstall(t *testing.T) {
	helmClient := newHelmClient()
	ctx := context.TODO()

	// Create a dummy resource
	res := createDummyResource()

	// Set up the install options
	opts := InstallOptions{
		HelmClient: helmClient,
		ChartName:  "https://charts.bitnami.com/bitnami",
		Resource:   res,
		Repo:       "postgresql",
		Version:    "12.8.3",
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
