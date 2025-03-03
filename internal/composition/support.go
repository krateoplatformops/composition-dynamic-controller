package composition

import (
	"fmt"

	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/elementsmatch"
	"github.com/krateoplatformops/composition-dynamic-controller/internal/tools/helmchart/archive"
	"github.com/krateoplatformops/unstructured-runtime/pkg/controller/objectref"
	"github.com/krateoplatformops/unstructured-runtime/pkg/pluralizer"
	unstructuredtools "github.com/krateoplatformops/unstructured-runtime/pkg/tools/unstructured"
	"github.com/krateoplatformops/unstructured-runtime/pkg/tools/unstructured/condition"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ManagedResource struct {
	APIVersion string `json:"apiVersion"`
	Resource   string `json:"resource"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
}

func setAvaibleStatus(mg *unstructured.Unstructured, pkg *archive.Info, message string) error {
	if pkg == nil {
		return fmt.Errorf("package info is nil")
	}

	unstructured.SetNestedField(mg.Object, pkg.Version, "status", "helmChartVersion")
	unstructured.SetNestedField(mg.Object, pkg.URL, "status", "helmChartUrl")

	currentCondition := unstructuredtools.GetCondition(mg, condition.Available().Type, condition.Available().Reason)

	if currentCondition != nil && currentCondition.Message == message {
		return nil
	}
	cond := condition.Available()
	cond.Message = message
	err := unstructuredtools.SetCondition(mg, cond)
	if err != nil {
		return fmt.Errorf("setting condition: %w", err)
	}
	return nil
}

func setManagedResources(mg *unstructured.Unstructured, managed []interface{}) {
	status := mg.Object["status"]
	if status == nil {
		status = map[string]interface{}{}
	}
	mapstatus := status.(map[string]interface{})
	mapstatus["managed"] = managed
	mg.Object["status"] = mapstatus
}

func checkManaged(mg *unstructured.Unstructured, managed []interface{}) (bool, error) {
	status := mg.Object["status"]
	if status == nil {
		return false, fmt.Errorf("status not found")
	}
	mapstatus := status.(map[string]interface{})
	managedStatus := mapstatus["managed"]

	if managedStatus == nil {
		return false, nil
	}

	return elementsmatch.ElementsMatch(managedStatus, managed)
}

func populateManagedResources(pluralizer pluralizer.PluralizerInterface, resources []objectref.ObjectRef) ([]interface{}, error) {
	var managed []interface{}
	for _, ref := range resources {
		gvr, err := pluralizer.GVKtoGVR(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind))
		if err != nil {
			return nil, fmt.Errorf("getting GVR for %s: %w", ref.String(), err)
		}
		managed = append(managed, ManagedResource{
			APIVersion: ref.APIVersion,
			Resource:   gvr.Resource,
			Name:       ref.Name,
			Namespace:  ref.Namespace,
		})
	}

	return managed, nil
}
