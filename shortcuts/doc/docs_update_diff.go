// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"fmt"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

// diffContextLines is the number of unchanged lines printed on either side of
// each diff hunk. Matches the `git diff -U2` convention, which is enough to
// orient a reader in most docx blocks without drowning stderr in boilerplate.
const diffContextLines = 2

// computeMarkdownDiff returns a git-style unified diff between before and
// after, focused on the single changed region between the longest common
// prefix and the longest common suffix. Returns an empty string when before
// and after are identical.
//
// The algorithm is intentionally simple — not Myers, not minimal — because
// `docs +update` replace/insert/delete modes touch a localized block range,
// so the "middle" that survives prefix/suffix trimming is already the
// user-visible change. A full LCS diff would buy better output for paired
// additions+deletions but at several hundred lines of implementation we
// don't need right now.
func computeMarkdownDiff(before, after string) string {
	if before == after {
		return ""
	}
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	// Longest common prefix.
	prefix := 0
	for prefix < len(beforeLines) && prefix < len(afterLines) &&
		beforeLines[prefix] == afterLines[prefix] {
		prefix++
	}

	// Longest common suffix, not overlapping the prefix on either side.
	suffix := 0
	for suffix < len(beforeLines)-prefix &&
		suffix < len(afterLines)-prefix &&
		beforeLines[len(beforeLines)-1-suffix] == afterLines[len(afterLines)-1-suffix] {
		suffix++
	}

	beforeEnd := len(beforeLines) - suffix
	afterEnd := len(afterLines) - suffix

	// Nothing changed (defensive; before == after already returned above).
	if prefix == beforeEnd && prefix == afterEnd {
		return ""
	}

	ctxStart := prefix - diffContextLines
	if ctxStart < 0 {
		ctxStart = 0
	}
	ctxEndBefore := beforeEnd + diffContextLines
	if ctxEndBefore > len(beforeLines) {
		ctxEndBefore = len(beforeLines)
	}
	ctxEndAfter := afterEnd + diffContextLines
	if ctxEndAfter > len(afterLines) {
		ctxEndAfter = len(afterLines)
	}

	var sb strings.Builder
	// Hunk header uses 1-based line numbers matching unified-diff convention.
	fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
		ctxStart+1, ctxEndBefore-ctxStart,
		ctxStart+1, ctxEndAfter-ctxStart,
	)
	for i := ctxStart; i < prefix; i++ {
		fmt.Fprintf(&sb, " %s\n", beforeLines[i])
	}
	for i := prefix; i < beforeEnd; i++ {
		fmt.Fprintf(&sb, "-%s\n", beforeLines[i])
	}
	for i := prefix; i < afterEnd; i++ {
		fmt.Fprintf(&sb, "+%s\n", afterLines[i])
	}
	for i := beforeEnd; i < ctxEndBefore; i++ {
		fmt.Fprintf(&sb, " %s\n", beforeLines[i])
	}
	return sb.String()
}

// fetchMarkdownForDiff calls the fetch-doc MCP tool and extracts the
// markdown payload. Errors are returned verbatim so the caller can decide
// whether to block the update on a failing snapshot (currently: no — the
// update still proceeds and the diff section is skipped).
func fetchMarkdownForDiff(runtime *common.RuntimeContext, docID string) (string, error) {
	result, err := common.CallMCPTool(runtime, "fetch-doc", map[string]interface{}{
		"doc_id":           docID,
		"skip_task_detail": true,
	})
	if err != nil {
		return "", err
	}
	md, _ := result["markdown"].(string)
	return md, nil
}
