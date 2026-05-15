// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package wiki

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
)

// Shared async-task polling for wiki delete operations. The wiki delete
// endpoints (DELETE /spaces/{id}, DELETE /spaces/{id}/nodes/{token}) may
// return either an empty task_id (sync completion) or a task_id that must
// be polled against /wiki/v2/tasks/{task_id}?task_type=<...>. The response
// envelopes share the same {status, status_msg} shape under different
// result keys, so the poll loop and classification methods are factored
// out here.

const (
	wikiAsyncStatusSuccess    = "success"
	wikiAsyncStatusFailure    = "failure"
	wikiAsyncStatusProcessing = "processing"

	wikiAsyncTaskTypeDeleteSpace = "delete_space"
	wikiAsyncTaskTypeDeleteNode  = "delete_node"

	wikiAsyncResultDeleteSpace = "delete_space_result"
	wikiAsyncResultDeleteNode  = "delete_node_result"
)

// wikiAsyncTaskStatus is the unified poll-response shape used by every wiki
// delete task. The taskID is captured so error/resume hints can name it.
type wikiAsyncTaskStatus struct {
	TaskID    string
	Status    string
	StatusMsg string
}

// normalizedStatus collapses whitespace and case so "  SUCCESS  " is
// classified the same as "success". Ready / Failed / StatusCode all derive
// from this so classification and the output `status` field can't disagree.
func (s wikiAsyncTaskStatus) normalizedStatus() string {
	return strings.ToLower(strings.TrimSpace(s.Status))
}

func (s wikiAsyncTaskStatus) Ready() bool {
	return s.normalizedStatus() == wikiAsyncStatusSuccess
}

func (s wikiAsyncTaskStatus) Failed() bool {
	// The sample protocol only documents "success" as a terminal OK. Treat any
	// explicit "failure"/"failed" signal as terminal, and unknown non-success
	// values as still-processing so we don't misreport a novel status as a hard
	// failure.
	lowered := s.normalizedStatus()
	return lowered == wikiAsyncStatusFailure || lowered == "failed"
}

// StatusCode returns a never-empty status value for the output envelope. If
// the backend response omits delete_*_result.status (or sends whitespace),
// fall back to "processing" so the documented timeout-shape stays accurate.
func (s wikiAsyncTaskStatus) StatusCode() string {
	if status := strings.TrimSpace(s.Status); status != "" {
		return status
	}
	return wikiAsyncStatusProcessing
}

func (s wikiAsyncTaskStatus) StatusLabel() string {
	if msg := strings.TrimSpace(s.StatusMsg); msg != "" {
		return msg
	}
	return s.StatusCode()
}

// wikiAsyncTaskFetcher returns the latest status for taskID. Implementations
// translate from runtime.CallAPI responses or test fakes.
type wikiAsyncTaskFetcher func(ctx context.Context, taskID string) (wikiAsyncTaskStatus, error)

// parseWikiAsyncTaskStatus normalizes an /wiki/v2/tasks/{task_id} payload.
// resultKey selects the right shape ("delete_space_result" / "delete_node_result").
func parseWikiAsyncTaskStatus(taskID string, task map[string]interface{}, resultKey string) (wikiAsyncTaskStatus, error) {
	if task == nil {
		return wikiAsyncTaskStatus{}, output.Errorf(output.ExitAPI, "api_error", "wiki task response missing task")
	}

	result := common.GetMap(task, resultKey)
	status := wikiAsyncTaskStatus{
		TaskID: common.GetString(task, "task_id"),
	}
	if status.TaskID == "" {
		status.TaskID = taskID
	}
	if result != nil {
		status.Status = common.GetString(result, "status")
		status.StatusMsg = common.GetString(result, "status_msg")
	}
	return status, nil
}

// pollWikiAsyncTask runs the bounded polling loop shared by every wiki delete
// shortcut. label is the human-readable operation name surfaced in stderr
// progress lines ("delete-space" / "delete-node"). nextCommand is the resume
// hint embedded into the wrapped error when every poll fails.
//
// attempts/interval are taken as parameters (instead of consts) so callers
// can keep their per-operation tunable constants for back-compat with the
// existing test hooks.
func pollWikiAsyncTask(
	ctx context.Context,
	runtime *common.RuntimeContext,
	taskID, label string,
	attempts int,
	interval time.Duration,
	fetcher wikiAsyncTaskFetcher,
	nextCommand string,
) (wikiAsyncTaskStatus, bool, error) {
	lastStatus := wikiAsyncTaskStatus{TaskID: taskID}
	var lastErr error
	hadSuccessfulPoll := false

	// The delete request already succeeded. Treat poll failures as transient
	// until every attempt fails, then return a resume hint instead of
	// discarding the task identifier.
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return lastStatus, false, ctx.Err()
			case <-time.After(interval):
			}
		}

		status, err := fetcher(ctx, taskID)
		if err != nil {
			lastErr = err
			fmt.Fprintf(runtime.IO().ErrOut, "Wiki %s status attempt %d/%d failed: %v\n", label, attempt, attempts, err)
			continue
		}
		lastStatus = status
		hadSuccessfulPoll = true

		if status.Ready() {
			fmt.Fprintf(runtime.IO().ErrOut, "Wiki %s task completed successfully.\n", label)
			return status, true, nil
		}
		if status.Failed() {
			return status, false, output.Errorf(output.ExitAPI, "api_error", "wiki %s task %s failed: %s", label, taskID, status.StatusLabel())
		}

		fmt.Fprintf(runtime.IO().ErrOut, "Wiki %s status %d/%d: %s\n", label, attempt, attempts, status.StatusLabel())
	}

	if !hadSuccessfulPoll && lastErr != nil {
		hint := fmt.Sprintf(
			"the wiki %s task was created but every status poll failed (task_id=%s)\nretry status lookup with: %s",
			label, taskID, nextCommand,
		)
		var exitErr *output.ExitError
		if errors.As(lastErr, &exitErr) && exitErr.Detail != nil {
			if strings.TrimSpace(exitErr.Detail.Hint) != "" {
				hint = exitErr.Detail.Hint + "\n" + hint
			}
			return lastStatus, false, output.ErrWithHint(exitErr.Code, exitErr.Detail.Type, exitErr.Detail.Message, hint)
		}
		return lastStatus, false, output.ErrWithHint(output.ExitAPI, "api_error", lastErr.Error(), hint)
	}

	return lastStatus, false, nil
}
