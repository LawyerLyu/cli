// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package contentsafety

import (
	"encoding/json"
	"testing"
)

func TestNormalize_GenericTypes(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{"nil", nil},
		{"string", "hello"},
		{"bool", true},
		{"json.Number", json.Number("42")},
		{"map", map[string]any{"key": "val"}},
		{"slice", []any{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalize(tt.input)
			if got == nil && tt.input != nil {
				t.Errorf("normalize(%v) = nil, want non-nil", tt.input)
			}
		})
	}
}

func TestNormalize_TypedStruct(t *testing.T) {
	type inner struct {
		Name string `json:"name"`
	}
	got := normalize(inner{Name: "test"})
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("normalize(struct) = %T, want map[string]any", got)
	}
	if m["name"] != "test" {
		t.Errorf("m[\"name\"] = %v, want %q", m["name"], "test")
	}
}

func TestNormalize_PreservesJsonNumber(t *testing.T) {
	type data struct {
		Count int64 `json:"count"`
	}
	got := normalize(data{Count: 9007199254740993})
	m := got.(map[string]any)
	num, ok := m["count"].(json.Number)
	if !ok {
		t.Fatalf("count is %T, want json.Number", m["count"])
	}
	if num.String() != "9007199254740993" {
		t.Errorf("count = %s, want 9007199254740993", num.String())
	}
}

func TestNormalize_UnmarshalableValue(t *testing.T) {
	ch := make(chan int)
	got := normalize(ch)
	if got != any(ch) {
		t.Error("unmarshalable value should return original")
	}
}
