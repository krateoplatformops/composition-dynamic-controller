package helmchart

import (
	"context"
	"fmt"

	"github.com/krateoplatformops/composition-dynamic-controller/internal/helmclient"
	"github.com/krateoplatformops/unstructured-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type UpdateOptions struct {
	CheckResourceOptions
	HelmClient  helmclient.Client
	ChartName   string
	Version     string
	Resource    *unstructured.Unstructured
	Repo        string
	Credentials *Credentials
}

func Update(ctx context.Context, opts UpdateOptions) error {
	chartSpec := helmclient.ChartSpec{
		ReleaseName:     meta.GetReleaseName(opts.Resource),
		Namespace:       opts.Resource.GetNamespace(),
		Repo:            opts.Repo,
		ChartName:       opts.ChartName,
		Version:         opts.Version,
		CreateNamespace: true,
		UpgradeCRDs:     true,
		Replace:         true,
		CleanupOnFail:   true,
		Install:         true,
	}
	if opts.Credentials != nil {
		chartSpec.Username = opts.Credentials.Username
		chartSpec.Password = opts.Credentials.Password
	}

	dat, err := ExtractValuesFromSpec(opts.Resource)
	if err != nil {
		return err
	}
	if len(dat) > 0 {
		chartSpec.ResetValues = true
		chartSpec.ValuesYaml = string(dat)
	}

	uid := opts.Resource.GetUID()

	gvr, err := opts.Pluralizer.GVKtoGVR(opts.Resource.GetObjectKind().GroupVersionKind())
	if err != nil {
		return fmt.Errorf("failed to get GVR: %w", err)
	}

	dat, err = AddOrUpdateFieldInValues(dat, uid, "global", "compositionId")
	if err != nil {
		return fmt.Errorf("failed to add compositionId to values: %w", err)
	}
	dat, err = AddOrUpdateFieldInValues(dat, opts.Resource.GetAPIVersion(), "global", "compositionApiVersion")
	if err != nil {
		return fmt.Errorf("failed to add compositionApiVersion to values: %w", err)
	}
	dat, err = AddOrUpdateFieldInValues(dat, gvr.Resource, "global", "compositionResource")
	if err != nil {
		return fmt.Errorf("failed to add compositionResource to values: %w", err)
	}

	chartSpec.ValuesYaml = string(dat)
	helmOpts := &helmclient.GenericHelmOptions{
		PostRenderer: &labelsPostRender{
			UID:                  uid,
			CompositionName:      opts.Resource.GetName(),
			CompositionNamespace: opts.Resource.GetNamespace(),
			CompositionGVR:       gvr,
		},
	}

	_, err = opts.HelmClient.UpgradeChart(ctx, &chartSpec, helmOpts)
	return err
}
