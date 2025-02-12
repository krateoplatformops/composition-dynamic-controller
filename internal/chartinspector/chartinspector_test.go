package chartinspector

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestChartInspector_Resources(t *testing.T) {
	mockResources := []Resource{
		{Group: "group1", Version: "v1", Resource: "resource1", Name: "name1", Namespace: "namespace1"},
		{Group: "group2", Version: "v2", Resource: "resource2", Name: "name2", Namespace: "namespace2"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("compositionDefinitionUID") != "uid1" ||
			r.URL.Query().Get("compositionDefinitionNamespace") != "namespace1" ||
			r.URL.Query().Get("compositionUID") != "uid2" ||
			r.URL.Query().Get("compositionNamespace") != "namespace2" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResources)
	}))
	defer server.Close()

	inspector := NewChartInspector(server.URL)

	resources, err := inspector.Resources("uid1", "namespace1", "uid2", "namespace2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(resources, mockResources) {
		t.Errorf("expected %v, got %v", mockResources, resources)
	}
}

func TestChartInspector_Resources_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	inspector := NewChartInspector(server.URL)

	_, err := inspector.Resources("uid1", "namespace1", "uid2", "namespace2")
	if err == nil {
		t.Fatalf("expected error, got none")
	}
}
