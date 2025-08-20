package meta

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetReleaseName(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "release name exists",
			labels:   map[string]string{ReleaseNameLabel: "test-release"},
			expected: "test-release",
		},
		{
			name:     "release name does not exist",
			labels:   map[string]string{"other-label": "value"},
			expected: "",
		},
		{
			name:     "nil labels",
			labels:   nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// obj := &unstructured.U{labels: tt.labels}
			obj := unstructured.Unstructured{}
			obj.SetLabels(tt.labels)
			result := GetReleaseName(&obj)
			if result != tt.expected {
				t.Errorf("GetReleaseName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSetReleaseName(t *testing.T) {
	tests := []struct {
		name           string
		initialLabels  map[string]string
		releaseName    string
		expectedLabels map[string]string
	}{
		{
			name:          "set release name on nil labels",
			initialLabels: nil,
			releaseName:   "new-release",
			expectedLabels: map[string]string{
				ReleaseNameLabel: "new-release",
			},
		},
		{
			name:          "set release name on empty labels",
			initialLabels: map[string]string{},
			releaseName:   "new-release",
			expectedLabels: map[string]string{
				ReleaseNameLabel: "new-release",
			},
		},
		{
			name: "set release name with existing labels",
			initialLabels: map[string]string{
				"existing-label": "value",
			},
			releaseName: "new-release",
			expectedLabels: map[string]string{
				"existing-label": "value",
				ReleaseNameLabel: "new-release",
			},
		},
		{
			name: "do not overwrite existing release name",
			initialLabels: map[string]string{
				ReleaseNameLabel: "existing-release",
				"other-label":    "value",
			},
			releaseName: "new-release",
			expectedLabels: map[string]string{
				ReleaseNameLabel: "existing-release",
				"other-label":    "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := unstructured.Unstructured{}
			obj.SetLabels(tt.initialLabels)
			SetReleaseName(&obj, tt.releaseName)

			labels := obj.GetLabels()
			if len(labels) != len(tt.expectedLabels) {
				t.Errorf("Expected %d labels, got %d", len(tt.expectedLabels), len(labels))
			}

			for key, expectedValue := range tt.expectedLabels {
				if actualValue, exists := labels[key]; !exists || actualValue != expectedValue {
					t.Errorf("Expected label %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

func TestSetCompositionDefinitionLabels(t *testing.T) {
	tests := []struct {
		name           string
		initialLabels  map[string]string
		cdInfo         CompositionDefinitionInfo
		expectedLabels map[string]string
	}{
		{
			name:          "set labels on nil labels",
			initialLabels: nil,
			cdInfo: CompositionDefinitionInfo{
				Namespace: "test-namespace",
				Name:      "test-name",
				GVR: schema.GroupVersionResource{
					Group:    "test.io",
					Version:  "v1",
					Resource: "testresources",
				},
			},
			expectedLabels: map[string]string{
				CompositionDefinitionNameLabel:      "test-name",
				CompositionDefinitionNamespaceLabel: "test-namespace",
				CompositionDefinitionGroupLabel:     "test.io",
				CompositionDefinitionVersionLabel:   "v1",
				CompositionDefinitionResourceLabel:  "testresources",
			},
		},
		{
			name: "set labels with existing labels",
			initialLabels: map[string]string{
				"existing-label": "value",
			},
			cdInfo: CompositionDefinitionInfo{
				Namespace: "test-namespace",
				Name:      "test-name",
				GVR: schema.GroupVersionResource{
					Group:    "test.io",
					Version:  "v1",
					Resource: "testresources",
				},
			},
			expectedLabels: map[string]string{
				"existing-label":                    "value",
				CompositionDefinitionNameLabel:      "test-name",
				CompositionDefinitionNamespaceLabel: "test-namespace",
				CompositionDefinitionGroupLabel:     "test.io",
				CompositionDefinitionVersionLabel:   "v1",
				CompositionDefinitionResourceLabel:  "testresources",
			},
		},
		{
			name: "overwrite existing composition definition labels",
			initialLabels: map[string]string{
				CompositionDefinitionNameLabel: "old-name",
				"other-label":                  "value",
			},
			cdInfo: CompositionDefinitionInfo{
				Namespace: "new-namespace",
				Name:      "new-name",
				GVR: schema.GroupVersionResource{
					Group:    "new.io",
					Version:  "v2",
					Resource: "newresources",
				},
			},
			expectedLabels: map[string]string{
				"other-label":                       "value",
				CompositionDefinitionNameLabel:      "new-name",
				CompositionDefinitionNamespaceLabel: "new-namespace",
				CompositionDefinitionGroupLabel:     "new.io",
				CompositionDefinitionVersionLabel:   "v2",
				CompositionDefinitionResourceLabel:  "newresources",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := unstructured.Unstructured{}
			obj.SetLabels(tt.initialLabels)
			SetCompositionDefinitionLabels(&obj, tt.cdInfo)

			labels := obj.GetLabels()
			if len(labels) != len(tt.expectedLabels) {
				t.Errorf("Expected %d labels, got %d", len(tt.expectedLabels), len(labels))
			}

			for key, expectedValue := range tt.expectedLabels {
				if actualValue, exists := labels[key]; !exists || actualValue != expectedValue {
					t.Errorf("Expected label %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}
func TestIsGracefullyPaused(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{
			name:        "gracefully paused annotation set to true",
			annotations: map[string]string{AnnotationKeyReconciliationGracefullyPaused: "true"},
			expected:    true,
		},
		{
			name:        "gracefully paused annotation set to false",
			annotations: map[string]string{AnnotationKeyReconciliationGracefullyPaused: "false"},
			expected:    false,
		},
		{
			name:        "gracefully paused annotation set to empty string",
			annotations: map[string]string{AnnotationKeyReconciliationGracefullyPaused: ""},
			expected:    false,
		},
		{
			name:        "gracefully paused annotation does not exist",
			annotations: map[string]string{"other-annotation": "value"},
			expected:    false,
		},
		{
			name:        "nil annotations",
			annotations: nil,
			expected:    false,
		},
		{
			name:        "empty annotations",
			annotations: map[string]string{},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := unstructured.Unstructured{}
			obj.SetAnnotations(tt.annotations)
			result := IsGracefullyPaused(&obj)
			if result != tt.expected {
				t.Errorf("IsGracefullyPaused() = %v, want %v", result, tt.expected)
			}
		})
	}
}
func TestSetGracefullyPausedTime(t *testing.T) {
	tests := []struct {
		name                string
		initialAnnotations  map[string]string
		timeToSet           time.Time
		expectedAnnotations map[string]string
	}{
		{
			name:               "set time on nil annotations",
			initialAnnotations: nil,
			timeToSet:          time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			expectedAnnotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "2023-10-15T14:30:45Z",
			},
		},
		{
			name:               "set time on empty annotations",
			initialAnnotations: map[string]string{},
			timeToSet:          time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			expectedAnnotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "2023-10-15T14:30:45Z",
			},
		},
		{
			name: "set time with existing annotations",
			initialAnnotations: map[string]string{
				"existing-annotation": "value",
			},
			timeToSet: time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			expectedAnnotations: map[string]string{
				"existing-annotation":                           "value",
				AnnotationKeyReconciliationGracefullyPausedTime: "2023-10-15T14:30:45Z",
			},
		},
		{
			name: "overwrite existing paused time",
			initialAnnotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "2023-01-01T00:00:00Z",
				"other-annotation": "value",
			},
			timeToSet: time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			expectedAnnotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "2023-10-15T14:30:45Z",
				"other-annotation": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := unstructured.Unstructured{}
			obj.SetAnnotations(tt.initialAnnotations)
			SetGracefullyPausedTime(&obj, tt.timeToSet)

			annotations := obj.GetAnnotations()
			if len(annotations) != len(tt.expectedAnnotations) {
				t.Errorf("Expected %d annotations, got %d", len(tt.expectedAnnotations), len(annotations))
			}

			for key, expectedValue := range tt.expectedAnnotations {
				if actualValue, exists := annotations[key]; !exists || actualValue != expectedValue {
					t.Errorf("Expected annotation %s=%s, got %s=%s", key, expectedValue, key, actualValue)
				}
			}
		})
	}
}

func TestGetGracefullyPausedTime(t *testing.T) {
	tests := []struct {
		name         string
		annotations  map[string]string
		expectedTime time.Time
		expectedOk   bool
	}{
		{
			name: "valid paused time annotation",
			annotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "2023-10-15T14:30:45Z",
			},
			expectedTime: time.Date(2023, 10, 15, 14, 30, 45, 0, time.UTC),
			expectedOk:   true,
		},
		{
			name: "annotation does not exist",
			annotations: map[string]string{
				"other-annotation": "value",
			},
			expectedTime: time.Time{},
			expectedOk:   false,
		},
		{
			name:         "nil annotations",
			annotations:  nil,
			expectedTime: time.Time{},
			expectedOk:   false,
		},
		{
			name:         "empty annotations",
			annotations:  map[string]string{},
			expectedTime: time.Time{},
			expectedOk:   false,
		},
		{
			name: "invalid time format",
			annotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "invalid-time-format",
			},
			expectedTime: time.Time{},
			expectedOk:   false,
		},
		{
			name: "empty time string",
			annotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "",
			},
			expectedTime: time.Time{},
			expectedOk:   false,
		},
		{
			name: "valid RFC3339 time with timezone",
			annotations: map[string]string{
				AnnotationKeyReconciliationGracefullyPausedTime: "2023-10-15T14:30:45+02:00",
			},
			expectedTime: time.Date(2023, 10, 15, 14, 30, 45, 0, time.FixedZone("", 2*60*60)),
			expectedOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := unstructured.Unstructured{}
			obj.SetAnnotations(tt.annotations)

			resultTime, resultOk := GetGracefullyPausedTime(&obj)

			if resultOk != tt.expectedOk {
				t.Errorf("GetGracefullyPausedTime() ok = %v, want %v", resultOk, tt.expectedOk)
			}

			if !resultTime.Equal(tt.expectedTime) {
				t.Errorf("GetGracefullyPausedTime() time = %v, want %v", resultTime, tt.expectedTime)
			}
		})
	}
}
