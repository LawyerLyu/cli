// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package contentsafety

import (
	"bytes"
	"encoding/json"
)

func normalize(v any) any {
	switch v.(type) {
	case map[string]any, []any, string, json.Number, bool, nil:
		return v
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return v
	}
	return out
}
