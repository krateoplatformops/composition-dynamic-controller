//go:build integration
// +build integration

package helmchart

import (
	"context"
	"fmt"
	"testing"

	"github.com/krateoplatformops/composition-dynamic-controller/internal/helmclient"
	"github.com/krateoplatformops/plumbing/e2e"
	"github.com/krateoplatformops/unstructured-runtime/pkg/meta"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func ExampleExtractValuesFromSpec() {
	res := createDummyResource()

	dat, err := ExtractValuesFromSpec(res)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(dat))
	// Output:
	// data:
	//   counter: 1
	//   greeting: Hello World!
	//   like: false
}

func TestRenderTemplate(t *testing.T) {
	f := features.New("RenderTemplate").
		Setup(e2e.Logger("test")).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("RenderTemplate", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

		helmClient, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
			RestConfig: cfg.Client().RESTConfig(),
			Options: &helmclient.Options{
				Namespace: namespace,
			},
		})
		if err != nil {
			t.Fatalf("failed to create helm client: %v", err)
		}

		res := createDummyResource()

		opts := RenderTemplateOptions{
			PackageUrl:     "oci://registry-1.docker.io/bitnamicharts/postgresql",
			PackageVersion: "12.8.3",
			HelmClient:     helmClient,
			Resource:       res,
		}

		_, all, err := RenderTemplate(ctx, opts)
		if err != nil {
			t.Fatalf("RenderTemplate failed: %v", err)
		}

		assert.Equal(t, "v1", all[0].APIVersion)
		assert.Equal(t, "Secret", all[0].Kind)
		assert.Equal(t, "demo-postgresql", all[0].Name)
		assert.Equal(t, "demo-system", all[0].Namespace)

		assert.Equal(t, "v1", all[1].APIVersion)
		assert.Equal(t, "Service", all[1].Kind)
		assert.Equal(t, "demo-postgresql-hl", all[1].Name)
		assert.Equal(t, "demo-system", all[1].Namespace)

		return ctx
	}).Feature()

	testenv.Test(t, f)
}

func TestFindRelease(t *testing.T) {
	f := features.New("FindRelease").
		Setup(e2e.Logger("test")).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("FindRelease", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

		helmClient, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
			RestConfig: cfg.Client().RESTConfig(),
			Options: &helmclient.Options{
				Namespace: namespace,
			},
		})
		if err != nil {
			t.Fatalf("failed to create helm client: %v", err)
		}

		releaseName := "my-release"

		// Call the FindRelease function
		actualRelease, err := FindRelease(helmClient, releaseName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check if the actual release matches the expected release (should be nil for non-existent release)
		if actualRelease != nil {
			t.Fatalf("expected release %v, got %v", nil, actualRelease)
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}

func TestFindAllReleases(t *testing.T) {
	f := features.New("FindAllReleases").
		Setup(e2e.Logger("test")).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("FindAllReleases", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

		helmClient, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
			RestConfig: cfg.Client().RESTConfig(),
			Options: &helmclient.Options{
				Namespace: namespace,
			},
		})
		if err != nil {
			t.Fatalf("failed to create helm client: %v", err)
		}

		// First, check that we start with no releases
		releases, err := FindAllReleases(helmClient)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assert.NotNil(t, releases)
		assert.Len(t, releases, 0, "Expected no releases initially")

		dynamicClient, err := dynamic.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatalf("failed to create dynamic client: %v", err)
		}

		// Install a successful test release using Install function
		successfulReleaseName := "test-release-success"
		successfulRes := createDummyResource()

		ls := successfulRes.GetLabels()
		if ls == nil {
			ls = make(map[string]string)
		}
		ls[meta.ReleaseNameLabel] = successfulReleaseName
		successfulRes.SetLabels(ls)
		successfulRes.SetName("success-resource")
		successfulRes.SetUID("12345678-1234-1234-1234-123456789001")

		// Create invalid configuration that will cause the PostgreSQL chart to fail
		validData := map[string]interface{}{
			"service": map[string]interface{}{
				"type": "NodePort",
				"port": int64(31180),
			},
		}
		unstructured.SetNestedField(successfulRes.Object, validData, "spec")

		successfulOpts := InstallOptions{
			HelmClient: helmClient,
			ChartName:  "https://charts.krateo.io",
			Resource:   successfulRes,
			Repo:       "nginx",
			Version:    "0.1.0",
			CheckResourceOptions: CheckResourceOptions{
				DynamicClient: dynamicClient,
				Pluralizer:    FakePluralizer{},
			},
			KrateoNamespace: namespace,
		}

		// Install the successful release
		_, _, err = Install(ctx, successfulOpts)
		if err != nil {
			t.Fatalf("failed to install successful test chart: %v", err)
		}

		// Install a release that will fail using Install function with invalid configuration
		failedReleaseName := "test-release-failed"
		failedRes := createDummyResource()

		lsFailed := failedRes.GetLabels()
		if lsFailed == nil {
			lsFailed = make(map[string]string)
		}
		lsFailed[meta.ReleaseNameLabel] = failedReleaseName
		failedRes.SetLabels(lsFailed)
		failedRes.SetName("failed-resource")
		failedRes.SetUID("12345678-1234-1234-1234-123456789002")
		// Create invalid configuration that will cause the PostgreSQL chart to fail
		invalidData := map[string]interface{}{
			"service": map[string]interface{}{
				"type": "NodePort",
				"port": int64(31180),
			},
		}
		unstructured.SetNestedField(failedRes.Object, invalidData, "spec")

		failedOpts := InstallOptions{
			HelmClient: helmClient,
			ChartName:  "https://charts.krateo.io",
			Resource:   failedRes,
			Repo:       "nginx",
			Version:    "0.1.0",
			CheckResourceOptions: CheckResourceOptions{
				DynamicClient: dynamicClient,
				Pluralizer:    FakePluralizer{},
			},
			KrateoNamespace: namespace,
		}

		// Try to install the potentially failing release
		_, _, err = Install(ctx, failedOpts)
		// We don't treat this as fatal since it might fail by design
		if err != nil {
			t.Logf("Expected potential failure when installing failing release: %v", err)
		}

		// Now check that FindAllReleases finds the installed releases
		releases, err = FindAllReleases(helmClient)
		if err != nil {
			t.Fatalf("unexpected error after installing releases: %v", err)
		}

		assert.NotNil(t, releases)
		// We should have at least 1 release (successful one), possibly 2 if the failed one was also created
		assert.GreaterOrEqual(t, len(releases), 1, "Expected to find at least 1 release after installations")

		// Log all found releases and their statuses
		t.Logf("Found %d releases:", len(releases))
		for _, release := range releases {
			t.Logf("Release '%s' with status: %s", release.Name, release.Info.Status.String())
		}

		// Verify we can find the successful release
		var successfulRelease *release.Release
		var failedRelease *release.Release

		for _, rel := range releases {
			if rel.Name == successfulReleaseName {
				successfulRelease = rel
			}
			if rel.Name == failedReleaseName {
				failedRelease = rel
			}
		}

		// Check successful release
		assert.NotNil(t, successfulRelease, "Should find the successful release")
		if successfulRelease != nil {
			assert.Equal(t, successfulReleaseName, successfulRelease.Name)
			assert.Equal(t, namespace, successfulRelease.Namespace)
			assert.NotEmpty(t, successfulRelease.Info.Status.String(), "Successful release should have a status")
		}

		// Check if failed release was created and found
		if failedRelease != nil {
			t.Logf("Found failed release '%s' with status: %s", failedRelease.Name, failedRelease.Info.Status.String())
			assert.Equal(t, failedReleaseName, failedRelease.Name)
			assert.Equal(t, namespace, failedRelease.Namespace)
			assert.NotEmpty(t, failedRelease.Info.Status.String(), "Failed release should have a status")
		} else {
			t.Logf("Failed release was not created or not found - this might be expected depending on the failure mode")
		}

		// Clean up test releases
		err = helmClient.UninstallRelease(&helmclient.ChartSpec{
			ReleaseName: successfulReleaseName,
			Namespace:   namespace,
		})
		if err != nil {
			t.Logf("Warning: failed to clean up successful test release: %v", err)
		}

		if failedRelease != nil {
			err = helmClient.UninstallRelease(&helmclient.ChartSpec{
				ReleaseName: failedReleaseName,
				Namespace:   namespace,
			})
			if err != nil {
				t.Logf("Warning: failed to clean up failed test release: %v", err)
			}
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}

func TestFindAnyRelease(t *testing.T) {
	f := features.New("FindAnyRelease").
		Setup(e2e.Logger("test")).
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			return ctx
		}).Assess("FindAnyRelease", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {

		helmClient, err := helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
			RestConfig: cfg.Client().RESTConfig(),
			Options: &helmclient.Options{
				Namespace: namespace,
			},
		})
		if err != nil {
			t.Fatalf("failed to create helm client: %v", err)
		}

		// Test finding non-existent release
		nonExistentReleaseName := "non-existent-release"
		actualRelease, err := FindAnyRelease(helmClient, nonExistentReleaseName)
		assert.Nil(t, actualRelease)
		dynamicClient, err := dynamic.NewForConfig(cfg.Client().RESTConfig())
		if err != nil {
			t.Fatalf("failed to create dynamic client: %v", err)
		}

		// Install a test release
		testReleaseName := "test-findany-release"
		testRes := createDummyResource()

		ls := testRes.GetLabels()
		if ls == nil {
			ls = make(map[string]string)
		}
		ls[meta.ReleaseNameLabel] = testReleaseName
		testRes.SetLabels(ls)
		testRes.SetName("findany-resource")
		testRes.SetUID("12345678-1234-1234-1234-123456789003")

		validData := map[string]interface{}{
			"service": map[string]interface{}{
				"type": "NodePort",
				"port": int64(31180),
			},
		}
		unstructured.SetNestedField(testRes.Object, validData, "spec")

		installOpts := InstallOptions{
			HelmClient: helmClient,
			ChartName:  "https://charts.krateo.io",
			Resource:   testRes,
			Repo:       "nginx",
			Version:    "0.1.0",
			CheckResourceOptions: CheckResourceOptions{
				DynamicClient: dynamicClient,
				Pluralizer:    FakePluralizer{},
			},
			KrateoNamespace: namespace,
		}

		// Install the test release
		_, _, err = Install(ctx, installOpts)
		if err != nil {
			t.Fatalf("failed to install test chart: %v", err)
		}

		// Test finding existing release
		foundRelease, err := FindAnyRelease(helmClient, testReleaseName)
		assert.NoError(t, err)
		assert.NotNil(t, foundRelease)
		assert.Equal(t, testReleaseName, foundRelease.Name)
		assert.Equal(t, namespace, foundRelease.Namespace)

		// Clean up test release
		err = helmClient.UninstallRelease(&helmclient.ChartSpec{
			ReleaseName: testReleaseName,
			Namespace:   namespace,
		})
		if err != nil {
			t.Logf("Warning: failed to clean up test release: %v", err)
		}

		return ctx
	}).Feature()

	testenv.Test(t, f)
}

// func TestGetResourcesRefFromRelease(t *testing.T) {
// 	f := features.New("GetResourcesRefFromRelease").
// 		Setup(e2e.Logger("test")).
// 		Assess("SingleFileManifest", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
// 			// Initialize Helm Client to get the RESTMapper from the environment

// 			// _, err = helmclient.NewCachedClientFromRestConf(&helmclient.RestConfClientOptions{
// 			// 	RestConfig: cfg.Client().RESTConfig(),
// 			// }, clientSet)
// 			// if err != nil {
// 			// 	t.Fatalf("failed to create helm client: %v", err)
// 			// }

// 			// Use Case: Chart with only one file/resource to render
// 			rel := &release.Release{
// 				Namespace: "application-group",
// 				Name:      "single-resource-rel",
// 				Manifest: `
// ---
// # Source: application-group/templates/argocd/serviceaccount.argocd.yaml
// apiVersion: v1
// kind: ServiceAccount
// metadata:
//   # Using .Values.applicationGroupName since Microservice template need to reference
//   #   this resource (and in Microservice template cannot obtain .Release.Name of application-group)
//   name: "ag-test-au-260-argocd"
//   namespace: "krateo-system"
//   annotations:
//     helm.sh/resource-policy: keep
//     krateo.io/description: "Create Service Account to access ArgoCD pipelines"
//   labels:
//     helm.sh/chart: "application-group-1.6.0-rc.2"
//     app.kubernetes.io/name: "test-au-260"
//     app.kubernetes.io/instance: "test"
//     app.kubernetes.io/managed-by: "Helm"
// ...
// ---
// # Source: application-group/templates/argocd/secret.argocd-sa.yaml
// apiVersion: v1
// kind: Secret
// metadata:
//   name: "ag-test-argocd-sa"
//   namespace: "krateo-system"
//   annotations:
//     helm.sh/resource-policy: keep
//     krateo.io/description: "Create Service Account token to access ArgoCD pipelines"
//     kubernetes.io/service-account.name: "ag-test-au-260-argocd"
//   labels:
//     helm.sh/chart: "application-group-1.6.0-rc.2"
//     app.kubernetes.io/name: "test-au-260"
//     app.kubernetes.io/instance: "test"
//     app.kubernetes.io/managed-by: "Helm"
// type: "kubernetes.io/service-account-token"
// ...
// ---
// # Source: application-group/templates/argocd/appproject.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 1240
// ---
// # Source: application-group/templates/argocd/endpoint.argocd.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 3780
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/argocd/pipelinepermission.argocd.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 3877
// # [INFO] @dependsOn "endpoint/ag-test-argocd"
// ---
// # Source: application-group/templates/argocd/secret.argocd-sa.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 3647
// ---
// # Source: application-group/templates/argocd/serviceaccount.argocd.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 3604
// ---
// # Source: application-group/templates/endpoint/endpoint.to-share.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 701,776,850
// # [INFO] @dependsOn "teamproject/ag-test"

// ciao
// ---
// # Source: application-group/templates/permission/pipelinepermission.endpoint-to-share.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 3960,4042,4124
//     # [INFO] @dependsOn "ag-test-test-connection-docker"
//     # [INFO] @dependsOn "ag-test-test-connection-pbi-user"
//     # [INFO] @dependsOn "ag-test-test-connection-pbi-sp"
// ---
// # Source: application-group/templates/permission/pipelinepermission.gitrepository-pipeline-templates.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 5202
// # [INFO] @dependsOn "gitrepository/ag-test-pipeline-templates"
// ---
// # Source: application-group/templates/permission/pipelinepermission.gitrepository-pipeline.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 1672
// # [INFO] @dependsOn "gitrepository/ag-test-pipeline"
// ---
// # Source: application-group/templates/pipeline/environment.environments.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 188,258,328,398
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/repository/configmap.pipeline.global.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 1754
// # [INFO] @dependsOn "gitrepository/ag-test-pipeline"
// ---
// # Source: application-group/templates/repository/feed.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 468
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/repository/feedpermission.bsearchitettura.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 1088
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/repository/feedpermission.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 1163
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/repository/gitrepository.pipeline-templates.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 924
// # [INFO] @dependsOn "teamproject/ag-test"
// Test
// ---
// # Source: application-group/templates/repository/gitrepository.pipeline.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 1288
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/repository/repo.pipeline.global.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 1804
// # [INFO] @dependsOn "gitrepository/ag-test-pipeline"
// ---
// # Source: application-group/templates/repository/repositorypermission.pipeline-templates.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 997
// # [INFO] @dependsOn "gitrepository/ag-test-pipeline-templates"
// ---
// # Source: application-group/templates/teamproject/teamproject.bsearchitettura.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 572
// ---
// # Source: application-group/templates/teamproject/teamproject.bsetoolchaindevops.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 637
// ---
// # Source: application-group/templates/teamproject/teamproject.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 120
// ---
// # Source: application-group/templates/user/checkconfiguration.group-approvers.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4700
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.add-user-to-team-developer.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 5128
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.admin.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4208
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.approvers.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4632
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.backbone-default-approvers.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4482
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.bsearchitetturait-readers.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 5288
// ---
// # Source: application-group/templates/user/group.bsebackbone-admin.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4276
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.bsebackbone-operations.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4855
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.bsebackbone-support.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4482
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.contributors.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4788
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.developers.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4554
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.pr-approvers.yaml
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.readers.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4348
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/group.toolchaindevops-team.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4415
// ---
// # Source: application-group/templates/user/groups.add-aad-to-approvers-group.yaml
// # End of if eq $group.description "developers"      # End of with $relativeContext
//   # [INFO] @dependsOn "group/AG_TEST_AU_260_APPROVERS"  # End of if $entraIdGroup    # End of if eq $group.description "developers"      # End of with $relativeContext    # End of if eq $group.description "developers"      # End of with $relativeContext        # End of range $index, $group
// ---
// # Source: application-group/templates/user/groups.add-aad-to-developers-team.yaml
// # [INFO] @dependsOn "team/AG_TEST_AU_260_DEVELOPERS"  # End of if $entraIdGroup    # End of if eq $group.description "developers"      # End of with $relativeContext    # End of if eq $group.description "developers"      # End of with $relativeContext    # End of if eq $group.description "developers"      # End of with $relativeContext        # End of range $index, $group
// ---
// # Source: application-group/templates/user/groups.add-aad-to-pr-approvers-team.yaml
// # End of if eq $group.description "developers"      # End of with $relativeContext    # End of if eq $group.description "developers"      # End of with $relativeContext
//   # [INFO] @dependsOn "team/AG_TEST_AU_260_PR_APPROVERS"  # End of if $entraIdGroup    # End of if eq $group.description "developers"      # End of with $relativeContext        # End of range $index, $group
// ---
// # Source: application-group/templates/user/team.developers.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 5045
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/team.pr-approver.yaml
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/user.add-azdevopsconn-to-group-approvers-admin.yaml
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/user/user.add-bs-to-group-contributors.yaml
// # [DEBUG] krateo-v1 gruppoapplicativo (@commit 955ac2f) @line 4929
// # [INFO] @dependsOn "teamproject/ag-test"
// ---
// # Source: application-group/templates/argocd/appproject.yaml
// apiVersion: argoproj.io/v1alpha1
// kind: AppProject
// metadata:
//   # Using applicationGroupName since this value need to be referenced from Microservices
//   name: "test-au-260"
//   namespace: "krateo-system"
//   annotations:
//     helm.sh/resource-policy: keep
//     backbone.cloud/description: "ArgoCD project configuration"
//     krateo.io/connector-verbose: "false"
//   labels:
//     helm.sh/chart: "application-group-1.6.0-rc.2"
//     app.kubernetes.io/name: "test-au-260"
//     app.kubernetes.io/instance: "test"
//     app.kubernetes.io/managed-by: "Helm"
// spec:
//   clusterResourceWhitelist:
//     - group: "*"
//       kind: "*"
//   destinations:
//     - name: "*"
//       namespace: "*"
//       server: "*"
//   namespaceResourceWhitelist:
//     - group: "*"
//       kind: "*"
//   sourceRepos:
//     - "*"
//   sourceNamespaces:
//     - "krateo-system"
// `,
// 			}

// 			refs, err := GetResourcesRefFromRelease(rel, "default", true)
// 			assert.NoError(t, err)
// 			// assert.Len(t, refs, 82, "Should find exactly one resource reference")

// 			b, _ := json.MarshalIndent(refs, "", "  ")
// 			t.Logf("Extracted Resource Refs: %s", string(b))

// 			return ctx
// 		}).
// 		Assess("ClusterScopedResource", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
// 			// clientSet, err := helmclient.NewCachedClients(cfg.Client().RESTConfig())
// 			// if err != nil {
// 			// 	t.Fatalf("failed to create cached clients: %v", err)
// 			// }
// 			// _, _ = helmclient.NewClientFromRestConf(&helmclient.RestConfClientOptions{
// 			// 	RestConfig: cfg.Client().RESTConfig(),
// 			// })

// 			// Testing resource scope logic (Namespace should be empty for ClusterRole)
// 			rel := &release.Release{
// 				Name: "cluster-resource-rel",
// 				Manifest: `
// apiVersion: rbac.authorization.k8s.io/v1
// kind: ClusterRole
// metadata:
//   name: my-cluster-role
// `,
// 			}

// 			refs, err := GetResourcesRefFromRelease(rel, "some-namespace", true)
// 			assert.NoError(t, err)
// 			assert.Len(t, refs, 1)

// 			if len(refs) > 0 {
// 				assert.Equal(t, "", refs[0].Namespace, "Cluster-scoped resources should have empty namespace")
// 			}

// 			return ctx
// 		}).Feature()

// 	testenv.Test(t, f)
// }

func createDummyResource() *unstructured.Unstructured {
	data := map[string]interface{}{
		"like":     false,
		"greeting": "Hello World!",
		"counter":  int64(1),
	}

	res := &unstructured.Unstructured{}
	meta.SetReleaseName(res, "demo")
	res.SetGroupVersionKind(schema.FromAPIVersionAndKind("dummy-charts.krateo.io/v0-2-0", "DummyChart"))
	res.SetName("demo")
	res.SetNamespace("demo-system")
	unstructured.SetNestedField(res.Object, data, "spec", "data")

	return res
}
