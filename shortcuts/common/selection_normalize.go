// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import "strings"

// NormalizeSelectionWithEllipsis returns a canonical form of a user-typed
// --selection-with-ellipsis value suitable for server-side matching, along
// with a flag indicating whether any rewrite happened.
//
// The Lark docx store keeps punctuation in a canonical shape — straight ASCII
// quotes, LF line endings — while user-provided selection strings often come
// from pasted prose that has been auto-corrected to curly quotes, CRLF, or
// other typographic variants. Matching is strict byte-level, so a curly/
// straight mismatch on a single character is enough to defeat the whole
// selection.
//
// The normalization set is deliberately conservative: only transformations
// that are virtually always safe (typographic quotes and CR line endings)
// are applied. Full/half-width Latin punctuation or CJK punctuation is left
// alone, since those can legitimately appear verbatim in the document body.
func NormalizeSelectionWithEllipsis(s string) (string, bool) {
	if s == "" {
		return s, false
	}
	out := s
	// Curly single quotes → ASCII apostrophe.
	out = strings.ReplaceAll(out, "\u2018", "'")
	out = strings.ReplaceAll(out, "\u2019", "'")
	// Curly double quotes → ASCII double quote.
	out = strings.ReplaceAll(out, "\u201C", "\"")
	out = strings.ReplaceAll(out, "\u201D", "\"")
	// CRLF / standalone CR → LF. Lark stores LF internally; sending CRLF in
	// a selection would require the document to contain literal CR bytes,
	// which it never does.
	out = strings.ReplaceAll(out, "\r\n", "\n")
	out = strings.ReplaceAll(out, "\r", "\n")
	return out, out != s
}
