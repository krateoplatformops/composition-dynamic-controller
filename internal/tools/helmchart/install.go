package helmchart

import (
	"context"
	"fmt"

	"github.com/krateoplatformops/composition-dynamic-controller/internal/helmclient"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/meta"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Credentials struct {
	Username string
	Password string
}

type InstallOptions struct {
	CheckResourceOptions
	HelmClient  helmclient.Client
	ChartName   string
	Resource    *unstructured.Unstructured
	Repo        string
	Version     string
	Credentials *Credentials
}

func Install(ctx context.Context, opts InstallOptions) (*release.Release, int64, error) {
	chartSpec := helmclient.ChartSpec{
		ReleaseName:     meta.GetReleaseName(opts.Resource),
		Namespace:       opts.Resource.GetNamespace(),
		Version:         opts.Version,
		Repo:            opts.Repo,
		ChartName:       opts.ChartName,
		CreateNamespace: true,
		UpgradeCRDs:     true,
		Wait:            false,
	}
	if opts.Credentials != nil {
		chartSpec.Username = opts.Credentials.Username
		chartSpec.Password = opts.Credentials.Password
	}

	dat, err := ExtractValuesFromSpec(opts.Resource)
	if err != nil {
		return nil, 0, err
	}
	if len(dat) == 0 {
		return nil, 0, nil
	}
	uid := opts.Resource.GetUID()

	if opts.DiscoveryClient == nil {
		return nil, 0, fmt.Errorf("discovery client is required")
	}
	gvr, err := tools.GVKtoGVR(opts.DiscoveryClient, opts.Resource.GetObjectKind().GroupVersionKind())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get GVR: %w", err)
	}

	dat, err = AddOrUpdateFieldInValues(dat, uid, "global", "compositionId")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to add compositionId to values: %w", err)
	}
	dat, err = AddOrUpdateFieldInValues(dat, opts.Resource.GetAPIVersion(), "global", "compositionApiVersion")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to add compositionApiVersion to values: %w", err)
	}
	dat, err = AddOrUpdateFieldInValues(dat, gvr.Resource, "global", "compositionResource")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to add compositionResource to values: %w", err)
	}

	claimGen := opts.Resource.GetGeneration()
	chartSpec.ValuesYaml = string(dat)

	helmOpts := &helmclient.GenericHelmOptions{
		PostRenderer: &labelsPostRender{
			UID:                   uid,
			CompositionAPIVersion: opts.Resource.GetAPIVersion(),
			CompositionName:       opts.Resource.GetName(),
			CompositionNamespace:  opts.Resource.GetNamespace(),
			CompositionResource:   gvr.Resource,
		},
	}
	rel, err := opts.HelmClient.InstallOrUpgradeChart(ctx, &chartSpec, helmOpts)
	return rel, claimGen, err
}
