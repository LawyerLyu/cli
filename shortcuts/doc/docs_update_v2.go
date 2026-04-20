// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

var validCommandsV2 = map[string]bool{
	"str_replace":             true,
	"str_delete":              true,
	"block_delete":            true,
	"block_insert_after":      true,
	"block_copy_insert_after": true,
	"block_replace":           true,
	"block_move_after":        true,
	"overwrite":               true,
	"append":                  true,
	// Table operations
	"table_insert_rows":     true,
	"table_insert_cols":     true,
	"table_delete_rows":     true,
	"table_delete_cols":     true,
	"table_merge_cells":     true,
	"table_unmerge_cells":   true,
	"table_update_property": true,
}

// v2UpdateFlags returns the flag definitions for the v2 (OpenAPI) update path.
func v2UpdateFlags() []common.Flag {
	return []common.Flag{
		{Name: "command", Desc: "operation: " + validCommandsV2Description(), Hidden: true, Enum: validCommandsV2Keys()},
		{Name: "doc-format", Desc: "content format (prefer XML)", Hidden: true, Default: "xml", Enum: []string{"xml", "markdown"}},
		{Name: "content", Desc: "new content (XML or Markdown)", Hidden: true, Input: []string{common.File, common.Stdin}},
		{Name: "pattern", Desc: "regex pattern for str_replace / str_delete", Hidden: true},
		{Name: "block-id", Desc: "target block ID for block_* operations", Hidden: true},
		{Name: "src-block-ids", Desc: "source block IDs (comma-separated) for block_copy_insert_after / block_move_after", Hidden: true},
		{Name: "revision-id", Desc: "base revision (-1 = latest)", Hidden: true, Type: "int", Default: "-1"},
	}
}

// validCommandsV2Keys returns the sorted list of v2 --command values, derived from
// validCommandsV2 so the enum never drifts when new commands are added.
func validCommandsV2Keys() []string {
	keys := make([]string, 0, len(validCommandsV2))
	for cmd := range validCommandsV2 {
		keys = append(keys, cmd)
	}
	sort.Strings(keys)
	return keys
}

// validCommandsV2Description renders the key list as a pipe-separated string for
// help and error text, keeping both in lock-step with validCommandsV2.
func validCommandsV2Description() string {
	return strings.Join(validCommandsV2Keys(), " | ")
}

func validateUpdateV2(_ context.Context, runtime *common.RuntimeContext) error {
	cmd := runtime.Str("command")
	if cmd == "" {
		return common.FlagErrorf("--command is required")
	}
	if !validCommandsV2[cmd] {
		return common.FlagErrorf("invalid --command %q, valid: %s", cmd, validCommandsV2Description())
	}
	content := runtime.Str("content")
	pattern := runtime.Str("pattern")
	blockID := runtime.Str("block-id")
	srcBlockIDs := runtime.Str("src-block-ids")

	switch cmd {
	case "str_replace":
		if pattern == "" {
			return common.FlagErrorf("--command str_replace requires --pattern")
		}
		if content == "" {
			return common.FlagErrorf("--command str_replace requires --content")
		}
	case "str_delete":
		if pattern == "" {
			return common.FlagErrorf("--command str_delete requires --pattern")
		}
	case "block_delete":
		if blockID == "" {
			return common.FlagErrorf("--command block_delete requires --block-id")
		}
	case "block_insert_after":
		if blockID == "" {
			return common.FlagErrorf("--command block_insert_after requires --block-id")
		}
		if content == "" {
			return common.FlagErrorf("--command block_insert_after requires --content")
		}
	case "block_copy_insert_after":
		if blockID == "" {
			return common.FlagErrorf("--command block_copy_insert_after requires --block-id")
		}
		if srcBlockIDs == "" {
			return common.FlagErrorf("--command block_copy_insert_after requires --src-block-ids")
		}
	case "block_move_after":
		if blockID == "" {
			return common.FlagErrorf("--command block_move_after requires --block-id")
		}
		if content == "" && srcBlockIDs == "" {
			return common.FlagErrorf("--command block_move_after requires --content or --src-block-ids")
		}
	case "block_replace":
		if blockID == "" {
			return common.FlagErrorf("--command block_replace requires --block-id")
		}
		if content == "" {
			return common.FlagErrorf("--command block_replace requires --content")
		}
	case "overwrite":
		if content == "" {
			return common.FlagErrorf("--command overwrite requires --content")
		}
	case "append":
		if content == "" {
			return common.FlagErrorf("--command append requires --content")
		}
	}
	return nil
}

func dryRunUpdateV2(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	ref, err := parseDocumentRef(runtime.Str("doc"))
	if err != nil {
		return common.NewDryRunAPI().Desc(fmt.Sprintf("error: %v", err))
	}
	body := buildUpdateBody(runtime)
	apiPath := fmt.Sprintf("/open-apis/docs_ai/v1/documents/%s", ref.Token)
	return common.NewDryRunAPI().
		PUT(apiPath).
		Desc("OpenAPI: update document").
		Body(body).
		Set("document_id", ref.Token)
}

func executeUpdateV2(_ context.Context, runtime *common.RuntimeContext) error {
	ref, err := parseDocumentRef(runtime.Str("doc"))
	if err != nil {
		return err
	}

	apiPath := fmt.Sprintf("/open-apis/docs_ai/v1/documents/%s", ref.Token)
	body := buildUpdateBody(runtime)

	data, err := doDocAPI(runtime, "PUT", apiPath, body)
	if err != nil {
		return err
	}

	runtime.OutRaw(data, nil)
	return nil
}

func buildUpdateBody(runtime *common.RuntimeContext) map[string]interface{} {
	cmd := runtime.Str("command")

	// append is a shorthand for block_insert_after with block_id "-1" (end of document)
	blockID := runtime.Str("block-id")
	if cmd == "append" {
		cmd = "block_insert_after"
		blockID = "-1"
	}

	body := map[string]interface{}{
		"format":  runtime.Str("doc-format"),
		"command": cmd,
	}
	if v := runtime.Int("revision-id"); v != 0 {
		body["revision_id"] = v
	}
	if v := runtime.Str("content"); v != "" {
		body["content"] = v
	}
	if v := runtime.Str("pattern"); v != "" {
		body["pattern"] = v
	}
	if blockID != "" {
		body["block_id"] = blockID
	}
	if v := runtime.Str("src-block-ids"); v != "" {
		body["src_block_ids"] = v
	}
	return body
}
