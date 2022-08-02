package main

import (
	"reflect"
	"testing"
	"time"
)

func TestFeatureFlagExistsTooLong(t *testing.T) {
	tests := []struct {
		name  string
		until time.Time
	}{
		{
			name:  "ProjectAOn",
			until: time.Date(2023, 6, 6, 23, 59, 59, 0, time.Local),
		},
		{
			name:  "ProjectBOn",
			until: time.Date(2022, 6, 20, 23, 59, 59, 0, time.Local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !hasMethodName(tt.name) {
				t.Fatalf("flag %v no longer exists. you may forgot to delete the test code", tt.name)
			}
			if tt.until.Before(time.Now()) {
				t.Fatalf("flag %v should be deleted", tt.name)
			}
		})
	}
}

func TestFeatureFlagExistTooMany(t *testing.T) {
	t.Run("Count of FeatureFlag", func(t *testing.T) {
		it := reflect.TypeOf((*FeatureConfig)(nil)).Elem()
		if it.NumMethod() > 2 {
			t.Fatalf("too many feature flags")
		}
	})
}

func hasMethodName(name string) bool {
	it := reflect.TypeOf((*FeatureConfig)(nil)).Elem()
	for i := 0; i < it.NumMethod(); i++ {
		if it.Method(i).Name == name {
			return true
		}
	}
	return false
}
