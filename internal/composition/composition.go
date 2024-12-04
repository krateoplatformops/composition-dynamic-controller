package composition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/krateoplatformops/composition-dynamic-controller/internal/helmclient"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/helmchart"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/helmchart/archive"
	"github.com/krateoplatformops/unstructured-runtime/pkg/controller"
	"github.com/krateoplatformops/unstructured-runtime/pkg/logging"
	"github.com/krateoplatformops/unstructured-runtime/pkg/meta"
	"github.com/krateoplatformops/unstructured-runtime/pkg/pluralizer"
	"github.com/krateoplatformops/unstructured-runtime/pkg/tools"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	unstructuredtools "github.com/krateoplatformops/unstructured-runtime/pkg/tools/unstructured"
	"github.com/krateoplatformops/unstructured-runtime/pkg/tools/unstructured/condition"
)

var (
	errReleaseNotFound  = errors.New("helm release not found")
	errCreateIncomplete = "cannot determine creation result - remove the " + meta.AnnotationKeyExternalCreatePending + " annotation if it is safe to proceed"
)

const (
	reasonCreated   = "CompositionCreated"
	reasonDeleted   = "CompositionDeleted"
	reasonReady     = "CompositionReady"
	reasonNotReady  = "CompositionNotReady"
	reasonUpdated   = "CompositionUpdated"
	reasonInstalled = "CompositionInstalled"
)

const (
	eventTypeNormal  = "Normal"
	eventTypeWarning = "Warning"
)

var _ controller.ExternalClient = (*handler)(nil)

func NewHandler(cfg *rest.Config, log logging.Logger, pig archive.Getter, event record.EventRecorder, dyn dynamic.Interface, disc discovery.CachedDiscoveryInterface, pluralizer pluralizer.Pluralizer) controller.ExternalClient {
	return &handler{
		pluralizer:        pluralizer,
		logger:            log,
		dynamicClient:     dyn,
		discoveryClient:   disc,
		packageInfoGetter: pig,
		eventRecorder:     event,
	}
}

type handler struct {
	logger            logging.Logger
	pluralizer        pluralizer.Pluralizer
	dynamicClient     dynamic.Interface
	discoveryClient   discovery.CachedDiscoveryInterface
	packageInfoGetter archive.Getter
	eventRecorder     record.EventRecorder
}

func (h *handler) Observe(ctx context.Context, mg *unstructured.Unstructured) (controller.ExternalObservation, error) {
	log := h.logger.WithValues("op", "Observe").
		WithValues("apiVersion", mg.GetAPIVersion()).
		WithValues("kind", mg.GetKind()).
		WithValues("name", mg.GetName()).
		WithValues("namespace", mg.GetNamespace())

	meta.SetReleaseName(mg, mg.GetName())

	if h.packageInfoGetter == nil {
		return controller.ExternalObservation{}, fmt.Errorf("helm chart package info getter must be specified")
	}

	pkg, err := h.packageInfoGetter.Get(mg)
	if err != nil {
		log.Debug("Getting package info", "error", err)
		return controller.ExternalObservation{}, err
	}

	hc, err := h.helmClientForResource(mg, pkg.RegistryAuth)
	if err != nil {
		log.Debug("Getting helm client", "error", err)
		return controller.ExternalObservation{}, err
	}

	rel, err := helmchart.FindRelease(hc, meta.GetReleaseName(mg))
	if err != nil {
		if !errors.Is(err, errReleaseNotFound) {
			return controller.ExternalObservation{}, err
		}
	}
	if rel == nil {
		log.Debug("Composition not found.")
		return controller.ExternalObservation{
			ResourceExists:   false,
			ResourceUpToDate: false,
		}, nil
	}

	if meta.ExternalCreateIncomplete(mg) {
		meta.RemoveAnnotations(mg, meta.AnnotationKeyExternalCreatePending)
		meta.SetExternalCreateSucceeded(mg, time.Now())
		mg, err = tools.Update(ctx, mg, tools.UpdateOptions{
			Pluralizer:    h.pluralizer,
			DynamicClient: h.dynamicClient,
		})
		if err != nil {
			log.Debug("Setting meta create succeeded annotation.", "error", err)
			return controller.ExternalObservation{}, err
		}
	}

	renderOpts := helmchart.RenderTemplateOptions{
		HelmClient:     hc,
		Resource:       mg,
		PackageUrl:     pkg.URL,
		PackageVersion: pkg.Version,
		Repo:           pkg.Repo,
	}
	if pkg.RegistryAuth != nil {
		renderOpts.Credentials = &helmchart.Credentials{
			Username: pkg.RegistryAuth.Username,
			Password: pkg.RegistryAuth.Password,
		}
	}

	renderedRel, all, err := helmchart.RenderTemplate(ctx, renderOpts)
	if err != nil {
		log.Debug("Rendering helm chart template", "error", err)
		return controller.ExternalObservation{}, fmt.Errorf("rendering helm chart template: %w", err)
	}
	if len(all) == 0 {
		return controller.ExternalObservation{}, nil
	}

	if rel.Chart.Metadata.Version != renderedRel.Chart.Metadata.Version {
		log.Debug("Composition package version mismatch.", "package", pkg.URL, "installed", rel.Chart.Metadata.Version, "expected", pkg.Version)
		return controller.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	log.Debug("Setting managed array", "package", pkg.URL)

	managed, err := populateManagedResources(h.pluralizer, all)
	if err != nil {
		log.Debug("Populating managed resources", "error", err)
		return controller.ExternalObservation{}, err
	}
	ok, err := checkManaged(mg, managed)
	if err != nil {
		log.Debug("Checking managed resources", "error", err)
		return controller.ExternalObservation{}, err
	}
	if !ok {
		log.Debug("Composition resources mismatch")
		return controller.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	setManagedResources(mg, managed)

	unstructured.SetNestedField(mg.Object, pkg.Version, "status", "helmChartVersion")
	unstructured.SetNestedField(mg.Object, pkg.URL, "status", "helmChartUrl")
	_ = unstructuredtools.SetCondition(mg, condition.Available())
	_, err = tools.UpdateStatus(ctx, mg, tools.UpdateOptions{
		// DiscoveryClient: h.discoveryClient,
		Pluralizer:    h.pluralizer,
		DynamicClient: h.dynamicClient,
	})
	if err != nil {
		log.Debug("Updating cr status with condition", "error", err, "condition", condition.Available())
		return controller.ExternalObservation{}, err
	}

	log.Debug("Composition Observed - installed", "package", pkg.URL)
	return controller.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (h *handler) Create(ctx context.Context, mg *unstructured.Unstructured) error {
	log := h.logger.WithValues("op", "Create").
		WithValues("apiVersion", mg.GetAPIVersion()).
		WithValues("kind", mg.GetKind()).
		WithValues("name", mg.GetName()).
		WithValues("namespace", mg.GetNamespace())

	meta.RemoveAnnotations(mg, meta.AnnotationKeyExternalCreatePending)
	mg, err := tools.Update(ctx, mg, tools.UpdateOptions{
		Pluralizer:    h.pluralizer,
		DynamicClient: h.dynamicClient,
	})
	if err != nil {
		log.Debug("Removing Create pending annotation", "error", err)
		return err
	}

	if meta.ExternalCreateIncomplete(mg) {
		log.Debug(errCreateIncomplete)
		err := unstructuredtools.SetCondition(mg, condition.Creating())
		if err != nil {
			return err
		}
		_, err = tools.UpdateStatus(ctx, mg, tools.UpdateOptions{
			Pluralizer:    h.pluralizer,
			DynamicClient: h.dynamicClient,
		})
		return err
	}

	meta.SetReleaseName(mg, mg.GetName())
	mg, err = tools.Update(ctx, mg, tools.UpdateOptions{
		Pluralizer:    h.pluralizer,
		DynamicClient: h.dynamicClient,
	})
	if err != nil {
		log.Debug("Updating composition", "error", err)
		return err
	}

	if h.packageInfoGetter == nil {
		return fmt.Errorf("helm chart package info getter must be specified")
	}

	pkg, err := h.packageInfoGetter.Get(mg)
	if err != nil {
		log.Debug("Getting package info", "error", err)
		return err
	}

	hc, err := h.helmClientForResource(mg, pkg.RegistryAuth)
	if err != nil {
		log.Debug("Getting helm client", "error", err)
		return err
	}

	opts := helmchart.InstallOptions{
		CheckResourceOptions: helmchart.CheckResourceOptions{
			DynamicClient: h.dynamicClient,
			Pluralizer:    h.pluralizer,
		},
		HelmClient: hc,
		ChartName:  pkg.URL,
		Resource:   mg,
		Repo:       pkg.Repo,
		Version:    pkg.Version,
	}
	if pkg.RegistryAuth != nil {
		opts.Credentials = &helmchart.Credentials{
			Username: pkg.RegistryAuth.Username,
			Password: pkg.RegistryAuth.Password,
		}
	}

	rel, _, err := helmchart.Install(ctx, opts)
	if err != nil {
		log.Debug("Installing helm chart", "package", pkg.URL, "error", err)
		meta.SetExternalCreateFailed(mg, time.Now())

		tools.Update(ctx, mg, tools.UpdateOptions{
			Pluralizer:    h.pluralizer,
			DynamicClient: h.dynamicClient,
		})

		return fmt.Errorf("installing helm chart: %w", err)
	}
	log.Debug("Installing composition package", "package", pkg.URL)

	meta.SetExternalCreatePending(mg, time.Now())

	mg, err = tools.Update(ctx, mg, tools.UpdateOptions{
		Pluralizer:    h.pluralizer,
		DynamicClient: h.dynamicClient,
	})
	if err != nil {
		log.Debug("Setting meta create pending annotation.", "error", err)
		return err
	}

	all, err := helmchart.GetResourcesRefFromRelease(rel, mg.GetNamespace())
	if err != nil {
		log.Debug("Getting resources from release", "error", err)
		return fmt.Errorf("getting resources from release: %w", err)
	}

	managed, err := populateManagedResources(h.pluralizer, all)
	if err != nil {
		log.Debug("Populating managed resources", "error", err)
		return err
	}
	setManagedResources(mg, managed)

	unstructuredtools.SetCondition(mg, condition.Available())
	h.eventRecorder.Eventf(mg, eventTypeNormal, reasonCreated, "Created composition: %s", mg.GetName())
	_, err = tools.UpdateStatus(ctx, mg, tools.UpdateOptions{
		Pluralizer:    h.pluralizer,
		DynamicClient: h.dynamicClient,
	})
	return err
}

func (h *handler) Update(ctx context.Context, mg *unstructured.Unstructured) error {
	log := h.logger.WithValues("op", "Update").
		WithValues("apiVersion", mg.GetAPIVersion()).
		WithValues("kind", mg.GetKind()).
		WithValues("name", mg.GetName()).
		WithValues("namespace", mg.GetNamespace())

	log.Debug("Handling composition values update.")

	// If we started but never completed creation of an external resource we
	// may have lost critical information.The safest thing to
	// do is to refuse to proceed.
	if meta.ExternalCreateIncomplete(mg) {
		log.Debug(errCreateIncomplete)
		_ = unstructuredtools.SetCondition(mg, condition.Creating())

		_, err := tools.UpdateStatus(ctx, mg, tools.UpdateOptions{
			Pluralizer:    h.pluralizer,
			DynamicClient: h.dynamicClient,
		})
		return err
	}

	if h.packageInfoGetter == nil {
		return fmt.Errorf("helm chart package info getter must be specified")
	}

	pkg, err := h.packageInfoGetter.Get(mg)
	if err != nil {
		log.Debug("Getting package info", "error", err)
		return err
	}

	hc, err := h.helmClientForResource(mg, pkg.RegistryAuth)
	if err != nil {
		log.Debug("Getting helm client", "error", err)
		return err
	}

	opts := helmchart.UpdateOptions{
		CheckResourceOptions: helmchart.CheckResourceOptions{
			DynamicClient: h.dynamicClient,
			Pluralizer:    h.pluralizer,
		},
		HelmClient: hc,
		ChartName:  pkg.URL,
		Resource:   mg,
		Repo:       pkg.Repo,
		Version:    pkg.Version,
	}
	if pkg.RegistryAuth != nil {
		opts.Credentials = &helmchart.Credentials{
			Username: pkg.RegistryAuth.Username,
			Password: pkg.RegistryAuth.Password,
		}
	}

	err = helmchart.Update(ctx, opts)
	if err != nil {
		log.Debug("Performing helm chart update", "error", err)
		return err
	}

	meta.SetExternalCreatePending(mg, time.Now())
	mg, err = tools.Update(ctx, mg, tools.UpdateOptions{
		Pluralizer:    h.pluralizer,
		DynamicClient: h.dynamicClient,
	})
	if err != nil {
		log.Debug("Setting meta create pending annotation.", "error", err)
		return err
	}

	// Update the composition values in the status.
	renderOpts := helmchart.RenderTemplateOptions{
		Pluralizer:     h.pluralizer,
		HelmClient:     hc,
		Resource:       mg,
		PackageUrl:     pkg.URL,
		PackageVersion: pkg.Version,
		Repo:           pkg.Repo,
	}
	if pkg.RegistryAuth != nil {
		renderOpts.Credentials = &helmchart.Credentials{
			Username: pkg.RegistryAuth.Username,
			Password: pkg.RegistryAuth.Password,
		}
	}
	_, all, err := helmchart.RenderTemplate(ctx, renderOpts)
	if err != nil {
		log.Debug("Rendering helm chart template", "error", err)
		return fmt.Errorf("rendering helm chart template: %w", err)
	}
	if len(all) == 0 {
		return nil
	}

	managed, err := populateManagedResources(h.pluralizer, all)
	if err != nil {
		log.Debug("Populating managed resources", "error", err)
		return err
	}
	setManagedResources(mg, managed)

	log.Debug("Composition values updated.", "package", pkg.URL)
	h.eventRecorder.Eventf(mg, eventTypeNormal, reasonUpdated, "Updated composition: %s", mg.GetName())

	cond := condition.Creating()
	cond.Message = "Composition values updated."
	unstructuredtools.SetCondition(mg, cond)

	_, err = tools.UpdateStatus(ctx, mg, tools.UpdateOptions{
		Pluralizer:    h.pluralizer,
		DynamicClient: h.dynamicClient,
	})
	return err
}

func (h *handler) Delete(ctx context.Context, mg *unstructured.Unstructured) error {
	log := h.logger.WithValues("op", "Delete").
		WithValues("apiVersion", mg.GetAPIVersion()).
		WithValues("kind", mg.GetKind()).
		WithValues("name", mg.GetName()).
		WithValues("namespace", mg.GetNamespace())

	if h.packageInfoGetter == nil {
		return fmt.Errorf("helm chart package info getter must be specified")
	}

	hc, err := h.helmClientForResource(mg, nil)
	if err != nil {
		log.Debug("Getting helm client", "error", err)
		return err
	}

	pkg, err := h.packageInfoGetter.Get(mg)
	if err != nil {
		log.Debug("Getting package info", "error", err)
		return err
	}

	chartSpec := helmclient.ChartSpec{
		ReleaseName: mg.GetName(),
		Namespace:   mg.GetNamespace(),
		ChartName:   pkg.URL,
		Version:     pkg.Version,
		Wait:        true,
		Timeout:     time.Minute * 3,
	}

	err = hc.UninstallRelease(&chartSpec)
	if err != nil {
		log.Debug("Uninstalling helm chart", "error", err)
		return err
	}

	h.eventRecorder.Eventf(mg, eventTypeNormal, reasonDeleted, "Deleted composition: %s", mg.GetName())
	log.Debug("Composition package removed.", "package", pkg.URL)

	return nil
}

func (h *handler) helmClientForResource(mg *unstructured.Unstructured, registryAuth *helmclient.RegistryAuth) (helmclient.Client, error) {
	log := h.logger.WithValues("apiVersion", mg.GetAPIVersion()).
		WithValues("kind", mg.GetKind()).
		WithValues("name", mg.GetName()).
		WithValues("namespace", mg.GetNamespace())

	opts := &helmclient.Options{
		Namespace:        mg.GetNamespace(),
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          false,
		DebugLog: func(format string, v ...interface{}) {
			if !meta.IsVerbose(mg) {
				return
			}

			if len(v) > 0 {
				log.Debug(fmt.Sprintf(format, v))
			} else {
				log.Debug(format)
			}
		},
		RegistryAuth: (registryAuth),
	}

	return helmclient.New(opts)
}
