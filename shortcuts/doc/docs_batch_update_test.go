// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"strings"
	"testing"
)

func TestParseBatchUpdateOps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr string
	}{
		{
			name:    "empty input rejected",
			input:   "",
			wantErr: "--operations is required",
		},
		{
			name:    "whitespace-only rejected",
			input:   "   \n\t",
			wantErr: "--operations is required",
		},
		{
			name:    "single object rejected with clear message",
			input:   `{"mode":"append","markdown":"x"}`,
			wantErr: "must be a JSON array",
		},
		{
			name:    "scalar rejected",
			input:   `"append"`,
			wantErr: "must be a JSON array",
		},
		{
			name:    "empty array rejected",
			input:   `[]`,
			wantErr: "at least one operation",
		},
		{
			name:    "malformed JSON surfaced",
			input:   `[{"mode":"append"`,
			wantErr: "not valid JSON",
		},
		{
			name:    "single-op array parses",
			input:   `[{"mode":"append","markdown":"hello"}]`,
			wantLen: 1,
		},
		{
			name: "multi-op array parses",
			input: `[
				{"mode":"replace_range","markdown":"x","selection_with_ellipsis":"a...b"},
				{"mode":"insert_before","markdown":"y","selection_by_title":"## H2"},
				{"mode":"delete_range","selection_with_ellipsis":"z...z"}
			]`,
			wantLen: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ops, err := parseBatchUpdateOps(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(ops) != tt.wantLen {
				t.Fatalf("got %d ops, want %d", len(ops), tt.wantLen)
			}
		})
	}
}

func TestValidateBatchUpdateOp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		op      batchUpdateOp
		wantErr string
	}{
		{
			name: "append with markdown is valid",
			op:   batchUpdateOp{Mode: "append", Markdown: "hello"},
		},
		{
			name: "replace_range with selection-with-ellipsis is valid",
			op:   batchUpdateOp{Mode: "replace_range", Markdown: "new", SelectionWithEllipsis: "a...b"},
		},
		{
			name: "replace_range with selection-by-title is valid",
			op:   batchUpdateOp{Mode: "replace_range", Markdown: "new", SelectionByTitle: "## Section"},
		},
		{
			name: "delete_range without markdown is valid",
			op:   batchUpdateOp{Mode: "delete_range", SelectionWithEllipsis: "a...b"},
		},
		{
			name:    "unknown mode rejected",
			op:      batchUpdateOp{Mode: "bogus", Markdown: "x"},
			wantErr: "invalid mode",
		},
		{
			name:    "non-delete mode without markdown rejected",
			op:      batchUpdateOp{Mode: "replace_range", SelectionWithEllipsis: "a...b"},
			wantErr: "requires markdown",
		},
		{
			name:    "both selections rejected",
			op:      batchUpdateOp{Mode: "replace_range", Markdown: "x", SelectionWithEllipsis: "a", SelectionByTitle: "## b"},
			wantErr: "mutually exclusive",
		},
		{
			name:    "selection-requiring mode without any selection rejected",
			op:      batchUpdateOp{Mode: "replace_range", Markdown: "x"},
			wantErr: "requires selection_with_ellipsis or selection_by_title",
		},
		{
			name:    "selection-by-title without leading hash rejected",
			op:      batchUpdateOp{Mode: "replace_range", Markdown: "x", SelectionByTitle: "Section"},
			wantErr: "heading prefix",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateBatchUpdateOp(tt.op)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestBuildBatchUpdateArgs(t *testing.T) {
	t.Parallel()

	t.Run("omits empty optional fields", func(t *testing.T) {
		t.Parallel()
		args := buildBatchUpdateArgs("DOC123", batchUpdateOp{Mode: "append", Markdown: "hello"})
		if _, ok := args["selection_with_ellipsis"]; ok {
			t.Errorf("expected selection_with_ellipsis omitted when empty")
		}
		if _, ok := args["selection_by_title"]; ok {
			t.Errorf("expected selection_by_title omitted when empty")
		}
		if _, ok := args["new_title"]; ok {
			t.Errorf("expected new_title omitted when empty")
		}
		if args["doc_id"] != "DOC123" {
			t.Errorf("expected doc_id DOC123, got %v", args["doc_id"])
		}
		if args["mode"] != "append" {
			t.Errorf("expected mode append, got %v", args["mode"])
		}
	})

	t.Run("carries all set fields", func(t *testing.T) {
		t.Parallel()
		args := buildBatchUpdateArgs("DOC123", batchUpdateOp{
			Mode:                  "replace_range",
			Markdown:              "new",
			SelectionWithEllipsis: "a...b",
			NewTitle:              "Renamed",
		})
		if args["selection_with_ellipsis"] != "a...b" {
			t.Errorf("expected selection_with_ellipsis a...b")
		}
		if args["new_title"] != "Renamed" {
			t.Errorf("expected new_title Renamed")
		}
	})

	t.Run("delete_range without markdown omits the key", func(t *testing.T) {
		t.Parallel()
		args := buildBatchUpdateArgs("DOC123", batchUpdateOp{
			Mode:                  "delete_range",
			SelectionWithEllipsis: "a...b",
		})
		if _, ok := args["markdown"]; ok {
			t.Errorf("expected markdown omitted for delete_range with empty markdown")
		}
	})
}
