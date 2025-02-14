//go:build integration
// +build integration

package composition

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	genctrl "github.com/krateoplatformops/unstructured-runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"

	"github.com/gobuffalo/flect"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/chartinspector"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/rbacgen"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/helmchart/archive"
	"github.com/krateoplatformops/unstructured-runtime/pkg/controller"
	"github.com/krateoplatformops/unstructured-runtime/pkg/eventrecorder"
	"github.com/krateoplatformops/unstructured-runtime/pkg/logging"
	"github.com/krateoplatformops/unstructured-runtime/pkg/pluralizer"

	"github.com/krateoplatformops/snowplow/plumbing/e2e"
	xenv "github.com/krateoplatformops/snowplow/plumbing/env"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/support/kind"
)

type FakePluralizer struct {
}

var _ pluralizer.PluralizerInterface = &FakePluralizer{}

func (p FakePluralizer) GVKtoGVR(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: flect.Pluralize(strings.ToLower(gvk.Kind)),
	}, nil
}

type FakeChartInspector struct {
	mock.Mock
}

func (m *FakeChartInspector) Resources(compositionDefinitionUID, compositionDefinitionNamespace, compositionUID, compositionNamespace string) ([]chartinspector.Resource, error) {
	args := m.Called(compositionDefinitionUID, compositionDefinitionNamespace, compositionUID, compositionNamespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]chartinspector.Resource), args.Error(1)
}

var _ chartinspector.ChartInspectorInterface = &FakeChartInspector{}

var (
	testenv     env.Environment
	clusterName string
)

const (
	testdataPath = "../../../testdata"
	namespace    = "demo-system"
	altNamespace = "krateo-system"
)

func TestMain(m *testing.M) {
	xenv.SetTestMode(true)

	clusterName = "krateo"
	testenv = env.New()

	testenv.Setup(
		envfuncs.CreateCluster(kind.NewProvider(), clusterName),
		e2e.CreateNamespace(namespace),
		e2e.CreateNamespace(altNamespace),

		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {

			return ctx, nil
		},
	).Finish(
		envfuncs.DeleteNamespace(namespace),
		envfuncs.DestroyCluster(clusterName),
	)

	os.Exit(testenv.Run(m))
}

func TestController(t *testing.T) {

	var handler controller.ExternalClient
	var labelselector labels.Selector
	var c *rest.Config
	f := features.New("Setup").
		Setup(e2e.Logger("test")).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c = SetSAToken(ctx, t, cfg)
			r, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Error("Creating resource client.", "error", err)
				return ctx
			}

			clientset, err := kubernetes.NewForConfig(c)
			if err != nil {
				t.Fatal(err)
			}

			_, err = clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, "test-sa", metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			log := logging.NewNopLogger()

			var pig archive.Getter

			pig, err = archive.Dynamic(cfg.Client().RESTConfig(), log)
			if err != nil {
				t.Error("Creating chart url info getter.", "error", err)
				return ctx
			}

			compositionVersionLabel := "core.krateo.io/composition-version"
			resourceVersion := "v1alpha1"

			// Create a label requirement for the composition version
			labelreq, err := labels.NewRequirement(compositionVersionLabel, selection.Equals, []string{resourceVersion})
			if err != nil {
				t.Error("Creating label requirement.", "error", err)
				return ctx
			}
			labelselector = labels.NewSelector().Add(*labelreq)

			dyn, err := dynamic.NewForConfig(cfg.Client().RESTConfig())
			if err != nil {
				t.Error("Creating dynamic client.", "error", err)
				return ctx
			}

			discovery, err := discovery.NewDiscoveryClientForConfig(cfg.Client().RESTConfig())
			if err != nil {
				t.Error("Creating discovery client.", "error", err)
				return ctx
			}

			cachedDisc := memory.NewMemCacheClient(discovery)

			rec, err := eventrecorder.Create(cfg.Client().RESTConfig())
			if err != nil {
				t.Error("Creating event recorder.", "error", err)
				return ctx
			}

			pluralizer := FakePluralizer{}
			chartInspector := FakeChartInspector{}
			rbacgen := rbacgen.NewRBACGen("test-sa", altNamespace, &chartInspector)
			handler = NewHandler(cfg.Client().RESTConfig(), log, pig, rec, dyn, cachedDisc, pluralizer, rbacgen)

			controller := genctrl.New(genctrl.Options{
				Discovery:      cachedDisc,
				Client:         dyn,
				ResyncInterval: 30 * time.Second,
				GVR: schema.GroupVersionResource{
					Group:    "composition.krateo.io",
					Version:  "v1-1-10",
					Resource: "fireworkapps",
				},
				Namespace:    namespace,
				Config:       cfg.Client().RESTConfig(),
				Debug:        true,
				Logger:       log,
				ProviderName: "composition-dynamic-controller",
				ListWatcher: controller.ListWatcherConfiguration{
					LabelSelector: ptr.To(labelselector.String()),
				},
				Pluralizer: pluralizer,
			})
			controller.SetExternalClient(handler)

			r.WithNamespace(namespace)

			go controller.Run(ctx, 1)

			return ctx
		}).Assess("Install", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		fmt.Println("Assessing")
		return ctx
	}).Feature()

	testenv.Test(t, f)
}

func SetSAToken(ctx context.Context, t *testing.T, cfg *envconf.Config) *rest.Config {
	clientset, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
	if err != nil {
		t.Fatal(err)
	}

	_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Create a Role and RoleBinding for the ServiceAccount
	_, err = clientset.RbacV1().Roles(namespace).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa-role",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = clientset.RbacV1().RoleBindings(namespace).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa-role-binding",
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "test-sa",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "test-sa-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa-token",
			Namespace: namespace,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": "test-sa",
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	tokenSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, "test-sa-token", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	saToken := string(tokenSecret.Data["token"])
	cacrt := tokenSecret.Data["ca.crt"]

	if saToken == "" {
		t.Fatal("ServiceAccount token not found")
	}

	// Create a new REST config with the ServiceAccount token
	restConfig := cfg.Client().RESTConfig()
	restConfig.BearerToken = saToken
	restConfig.BearerTokenFile = ""

	config := &rest.Config{
		Host:        restConfig.Host,
		BearerToken: string(saToken),
		TLSClientConfig: rest.TLSClientConfig{
			CertData: cacrt,
			Insecure: true,
		},
	}

	return config
}
