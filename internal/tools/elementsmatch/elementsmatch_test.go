package elementsmatch

import (
	"testing"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect bool
	}{
		{"nil", nil, true},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1, 2, 3}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"key": 1}, false},
		{"empty array", [0]int{}, true},
		{"non-empty array", [3]int{1, 2, 3}, false},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer to zero value", new(int), true},
		{"non-nil pointer to non-zero value", func() *int { v := 42; return &v }(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmpty(tt.input); got != tt.expect {
				t.Errorf("isEmpty(%v) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestObjectsAreEqual(t *testing.T) {
	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
		expect   bool
	}{
		{"both nil", nil, nil, true},
		{"one nil", nil, 42, false},
		{"equal ints", 42, 42, true},
		{"different ints", 42, 43, false},
		{"equal strings", "hello", "hello", true},
		{"different strings", "hello", "world", false},
		{"equal byte slices", []byte{1, 2, 3}, []byte{1, 2, 3}, true},
		{"different byte slices", []byte{1, 2, 3}, []byte{4, 5, 6}, false},
		{"nil byte slice", nil, []byte{1, 2, 3}, false},
		{"both nil byte slices", []byte(nil), []byte(nil), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ObjectsAreEqual(tt.expected, tt.actual); got != tt.expect {
				t.Errorf("ObjectsAreEqual(%v, %v) = %v, want %v", tt.expected, tt.actual, got, tt.expect)
			}
		})
	}
}

type ManagedResource struct {
	APIVersion string `json:"apiVersion"`
	Resource   string `json:"resource"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
}

func TestElementsMatch(t *testing.T) {
	tests := []struct {
		name    string
		listA   interface{}
		listB   interface{}
		expect  bool
		wantErr bool
	}{
		{"both nil", nil, nil, true, false},
		{"both empty slices", []int{}, []int{}, true, false},
		{"equal slices", []int{1, 2, 3}, []int{3, 2, 1}, true, false},
		{"different length slices", []int{1, 2, 3}, []int{1, 2}, false, false},
		{"different elements", []int{1, 2, 3}, []int{4, 5, 6}, false, false},
		{"unsupported type listA", 42, []int{1, 2, 3}, false, true},
		{"unsupported type listB", []int{1, 2, 3}, 42, false, true},
		{"check managed", []ManagedResource{
			{
				Name:       "foo",
				APIVersion: "v1",
				Resource:   "foos",
				Namespace:  "default",
			},
			{
				Name:       "bar",
				APIVersion: "v1",
				Resource:   "bars",
				Namespace:  "default",
			},
		},
			[]ManagedResource{
				{
					Name:       "bar",
					APIVersion: "v1",
					Resource:   "bars",
					Namespace:  "default",
				},
				{
					Name:       "foo",
					APIVersion: "v1",
					Resource:   "foos",
					Namespace:  "default",
				},
			},
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ElementsMatch(tt.listA, tt.listB)
			if (err != nil) != tt.wantErr {
				t.Errorf("ElementsMatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expect {
				t.Errorf("ElementsMatch() = %v, want %v", got, tt.expect)
			}
		})
	}
}
