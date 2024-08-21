package helmchart

// import (
// 	"context"
// 	"testing"
// )

// func TestUpdate(t *testing.T) {
// 	helmClient := newHelmClient()
// 	ctx := context.TODO()

// 	// Create a dummy resource
// 	res := createDummyResource()

// 	Install(ctx, InstallOptions{
// 		HelmClient: helmClient,
// 		ChartName:  "https://charts.bitnami.com/bitnami",
// 		Resource:   res,
// 		Repo:       "postgresql",
// 		Version:    "12.8.3",
// 	})

// 	// Set up the update options
// 	opts := UpdateOptions{
// 		HelmClient: helmClient,
// 		ChartName:  "https://charts.bitnami.com/bitnami",
// 		Resource:   res,
// 		Repo:       "postgresql",
// 		Version:    "12.8.3",
// 	}

// 	// Call the Update function
// 	err := Update(ctx, opts)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// }
