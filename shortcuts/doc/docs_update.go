// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"context"
	"fmt"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

var validModes = map[string]bool{
	"append":        true,
	"overwrite":     true,
	"replace_range": true,
	"replace_all":   true,
	"insert_before": true,
	"insert_after":  true,
	"delete_range":  true,
}

var needsSelection = map[string]bool{
	"replace_range": true,
	"replace_all":   true,
	"insert_before": true,
	"insert_after":  true,
	"delete_range":  true,
}

var DocsUpdate = common.Shortcut{
	Service:     "docs",
	Command:     "+update",
	Description: "Update a Lark document",
	Risk:        "write",
	Scopes:      []string{"docx:document:write_only", "docx:document:readonly"},
	AuthTypes:   []string{"user", "bot"},
	Flags: []common.Flag{
		{Name: "doc", Desc: "document URL or token", Required: true},
		{Name: "mode", Desc: "update mode: append | overwrite | replace_range | replace_all | insert_before | insert_after | delete_range", Required: true},
		{Name: "markdown", Desc: "new content (Lark-flavored Markdown; create blank whiteboards with <whiteboard type=\"blank\"></whiteboard>, repeat to create multiple boards)", Input: []string{common.File, common.Stdin}},
		{Name: "selection-with-ellipsis", Desc: "content locator (e.g. 'start...end')"},
		{Name: "selection-by-title", Desc: "title locator (e.g. '## Section')"},
		{Name: "new-title", Desc: "also update document title"},
		{Name: "show-diff", Type: "bool", Desc: "fetch the document before and after the update and print a unified diff of the affected region to stderr (skipped for append / overwrite; adds two fetch-doc calls)"},
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		mode := runtime.Str("mode")
		if !validModes[mode] {
			return common.FlagErrorf("invalid --mode %q, valid: append | overwrite | replace_range | replace_all | insert_before | insert_after | delete_range", mode)
		}

		if mode != "delete_range" && runtime.Str("markdown") == "" {
			return common.FlagErrorf("--%s mode requires --markdown", mode)
		}

		selEllipsis := runtime.Str("selection-with-ellipsis")
		selTitle := runtime.Str("selection-by-title")
		if selEllipsis != "" && selTitle != "" {
			return common.FlagErrorf("--selection-with-ellipsis and --selection-by-title are mutually exclusive")
		}

		if needsSelection[mode] && selEllipsis == "" && selTitle == "" {
			return common.FlagErrorf("--%s mode requires --selection-with-ellipsis or --selection-by-title", mode)
		}
		if err := validateSelectionByTitle(selTitle); err != nil {
			return err
		}

		return nil
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		args := map[string]interface{}{
			"doc_id": runtime.Str("doc"),
			"mode":   runtime.Str("mode"),
		}
		if v := runtime.Str("markdown"); v != "" {
			args["markdown"] = v
		}
		if v := runtime.Str("selection-with-ellipsis"); v != "" {
			args["selection_with_ellipsis"] = v
		}
		if v := runtime.Str("selection-by-title"); v != "" {
			args["selection_by_title"] = v
		}
		if v := runtime.Str("new-title"); v != "" {
			args["new_title"] = v
		}
		return common.NewDryRunAPI().
			POST(common.MCPEndpoint(runtime.Config.Brand)).
			Desc("MCP tool: update-doc").
			Body(map[string]interface{}{"method": "tools/call", "params": map[string]interface{}{"name": "update-doc", "arguments": args}}).
			Set("mcp_tool", "update-doc").Set("args", args)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		mode := runtime.Str("mode")
		markdown := runtime.Str("markdown")

		// Static semantic checks run before the MCP call so users see
		// warnings even if the subsequent request fails. They never block
		// execution — the update still proceeds.
		for _, w := range docsUpdateWarnings(mode, markdown) {
			fmt.Fprintf(runtime.IO().ErrOut, "warning: %s\n", w)
		}

		args := map[string]interface{}{
			"doc_id": runtime.Str("doc"),
			"mode":   mode,
		}
		if markdown != "" {
			args["markdown"] = markdown
		}
		if v := runtime.Str("selection-with-ellipsis"); v != "" {
			args["selection_with_ellipsis"] = v
		}
		if v := runtime.Str("selection-by-title"); v != "" {
			args["selection_by_title"] = v
		}
		if v := runtime.Str("new-title"); v != "" {
			args["new_title"] = v
		}

		// Optional diff capture: fetch the document before the update so we
		// can render a unified diff after the update settles. Kept off the
		// default path because it doubles read-side MCP calls for every
		// update. append/overwrite don't get a diff — append has no "before"
		// context worth showing, and overwrite marks every line as changed,
		// defeating the purpose.
		showDiff := runtime.Bool("show-diff") &&
			mode != "append" && mode != "overwrite"
		var beforeMarkdown string
		var beforeErr error
		if showDiff {
			beforeMarkdown, beforeErr = fetchMarkdownForDiff(runtime, runtime.Str("doc"))
			if beforeErr != nil {
				fmt.Fprintf(runtime.IO().ErrOut,
					"warning: --show-diff pre-fetch failed (%v); update will proceed without a diff\n",
					beforeErr)
			}
		}

		result, err := common.CallMCPTool(runtime, "update-doc", args)
		if err != nil {
			return err
		}

		normalizeDocsUpdateResult(result, runtime.Str("markdown"))

		// Post-fetch and emit the diff. Any failure here is advisory only —
		// the update already succeeded, so degrade gracefully instead of
		// making the caller re-run.
		if showDiff && beforeErr == nil {
			afterMarkdown, afterErr := fetchMarkdownForDiff(runtime, runtime.Str("doc"))
			switch {
			case afterErr != nil:
				fmt.Fprintf(runtime.IO().ErrOut,
					"warning: --show-diff post-fetch failed (%v); update succeeded but no diff available\n",
					afterErr)
			default:
				diff := computeMarkdownDiff(
					fixExportedMarkdown(beforeMarkdown),
					fixExportedMarkdown(afterMarkdown),
				)
				if diff == "" {
					fmt.Fprintln(runtime.IO().ErrOut,
						"note: --show-diff found no textual change after the update (server may have normalized the markdown)")
				} else {
					fmt.Fprintf(runtime.IO().ErrOut, "--- before\n+++ after\n%s", diff)
				}
			}
		}

		runtime.Out(result, nil)
		return nil
	},
}

func normalizeDocsUpdateResult(result map[string]interface{}, markdown string) {
	if !isWhiteboardCreateMarkdown(markdown) {
		return
	}
	result["board_tokens"] = normalizeBoardTokens(result["board_tokens"])
}

func isWhiteboardCreateMarkdown(markdown string) bool {
	lower := strings.ToLower(markdown)
	if strings.Contains(lower, "```mermaid") || strings.Contains(lower, "```plantuml") {
		return true
	}
	return strings.Contains(lower, "<whiteboard") &&
		(strings.Contains(lower, `type="blank"`) || strings.Contains(lower, `type='blank'`))
}

func normalizeBoardTokens(raw interface{}) []string {
	switch v := raw.(type) {
	case nil:
		return []string{}
	case []string:
		return v
	case []interface{}:
		tokens := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				tokens = append(tokens, s)
			}
		}
		return tokens
	case string:
		if v == "" {
			return []string{}
		}
		return []string{v}
	default:
		return []string{}
	}
}

func validateSelectionByTitle(title string) error {
	if title == "" {
		return nil
	}
	trimmed := strings.TrimSpace(title)
	if strings.Contains(trimmed, "\n") || strings.Contains(trimmed, "\r") {
		return common.FlagErrorf("--selection-by-title must be a single heading line (for example: '## Section')")
	}
	if strings.HasPrefix(trimmed, "#") {
		return nil
	}
	return common.FlagErrorf("--selection-by-title must include markdown heading prefix '#'. Example: --selection-by-title '## Section'")
}
