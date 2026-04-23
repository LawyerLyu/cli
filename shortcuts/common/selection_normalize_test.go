// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import "testing"

func TestNormalizeSelectionWithEllipsis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		want        string
		wantChanged bool
	}{
		{
			name:        "empty passes through",
			input:       "",
			want:        "",
			wantChanged: false,
		},
		{
			name:        "ascii-only selection is untouched",
			input:       "欢迎大家多给反馈",
			want:        "欢迎大家多给反馈",
			wantChanged: false,
		},
		{
			name:        "curly single quotes normalized",
			input:       "\u2018That\u2019s All\u2019",
			want:        "'That's All'",
			wantChanged: true,
		},
		{
			name:        "curly double quotes normalized",
			input:       "he said \u201Chello\u201D",
			want:        "he said \"hello\"",
			wantChanged: true,
		},
		{
			name:        "mixed curly + straight normalized",
			input:       "start\u2019s...end",
			want:        "start's...end",
			wantChanged: true,
		},
		{
			name:        "crlf collapsed to lf",
			input:       "line1\r\nline2",
			want:        "line1\nline2",
			wantChanged: true,
		},
		{
			name:        "standalone cr collapsed to lf",
			input:       "line1\rline2",
			want:        "line1\nline2",
			wantChanged: true,
		},
		{
			name:        "already lf is untouched",
			input:       "line1\nline2",
			want:        "line1\nline2",
			wantChanged: false,
		},
		{
			name:        "chinese punctuation deliberately untouched",
			input:       "你好，世界",
			want:        "你好，世界",
			wantChanged: false,
		},
		{
			name:        "fullwidth latin deliberately untouched",
			input:       "ＡＢＣ",
			want:        "ＡＢＣ",
			wantChanged: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, changed := NormalizeSelectionWithEllipsis(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeSelectionWithEllipsis(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if changed != tt.wantChanged {
				t.Errorf("NormalizeSelectionWithEllipsis(%q) changed=%v, want %v", tt.input, changed, tt.wantChanged)
			}
		})
	}
}
