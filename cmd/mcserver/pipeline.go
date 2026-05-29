// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/wavetermdev/waveterm/pkg/mcstore"
)

// HandoffResult holds parsed values from handoff.md
type HandoffResult struct {
	Verdict     string // PASS | FAIL | MISSING
	BuildStatus string
	TokensUsed  int
	Notes       string
}

func getHandoffPath(runId string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mc", "runs", runId, "handoff.md")
}

// parseHandoff reads and parses handoff.md with loose matching.
// Missing file → verdict=MISSING. Present but no verdict → verdict=INCOMPLETE.
func parseHandoff(path string) HandoffResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return HandoffResult{Verdict: mcstore.HandoffStatusMissing}
	}
	content := string(data)

	result := HandoffResult{Verdict: mcstore.HandoffStatusIncomplete}

	verdictRe := regexp.MustCompile(`(?im)^verdict:\s*(PASS|FAIL|MISSING|INCOMPLETE)`)
	if m := verdictRe.FindStringSubmatch(content); m != nil {
		result.Verdict = strings.ToLower(m[1])
	}

	buildRe := regexp.MustCompile(`(?im)^build_status:\s*(\S+)`)
	if m := buildRe.FindStringSubmatch(content); m != nil {
		result.BuildStatus = m[1]
	}

	return result
}

// runL1Verify executes a build/lint command and returns (pass, output).
func runL1Verify(verifyCmd string, cwd string) (bool, string) {
	if verifyCmd == "" {
		return true, "skipped (no verify command)"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	parts := strings.Fields(verifyCmd)
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	output := string(out)
	if len(output) > 4000 {
		output = output[:4000] + "\n...(truncated)"
	}
	return err == nil, output
}

// extractVerifyCmd parses `@verify: <cmd>` from task's context_notes.
func extractVerifyCmd(contextNotes string) string {
	re := regexp.MustCompile(`(?m)^@verify:\s*(.+)$`)
	if m := re.FindStringSubmatch(contextNotes); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// handleSessionComplete is called by the agent (or stop hook) after finishing work.
// Body: { "run_id": "...", "cwd": "..." }
func handleSessionComplete(w http.ResponseWriter, r *http.Request) {
	sessionId := mux.Vars(r)["id"]

	var body struct {
		RunId string `json:"runid"`
		Cwd   string `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	ctx := r.Context()

	session, err := mcstore.GetSession(ctx, sessionId)
	if err != nil || session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	task, err := mcstore.GetTask(ctx, session.TaskId)
	if err != nil || task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	runId := body.RunId
	if runId == "" {
		runId = uuid.NewString()
	}
	cwd := body.Cwd
	if cwd == "" {
		cwd = session.Cwd
	}

	// 1. Parse handoff.md
	handoffPath := getHandoffPath(runId)
	handoff := parseHandoff(handoffPath)

	// 2. Run L1 verification
	verifyCmd := extractVerifyCmd(task.ContextNotes)
	l1Pass, l1Output := runL1Verify(verifyCmd, cwd)
	l1Status := mcstore.VerifierStatusPass
	if !l1Pass {
		l1Status = mcstore.VerifierStatusFail
	}
	if verifyCmd == "" {
		l1Status = mcstore.VerifierStatusSkip
	}

	// 3. Write VerifierResult
	now := time.Now().UTC().Format(time.RFC3339)
	taskRunId := uuid.NewString()
	runNumber := 1

	taskRun := &mcstore.TaskRun{
		Id:            taskRunId,
		TaskId:        session.TaskId,
		SessionId:     sessionId,
		RunNumber:     runNumber,
		PipelineMode:  "standard",
		HandoffPath:   handoffPath,
		HandoffStatus: handoff.Verdict,
		Verdict:       handoff.Verdict,
		StartedAt:     session.StartedAt,
	}
	finishedAt := now
	taskRun.FinishedAt = &finishedAt

	verifierResult := &mcstore.VerifierResult{
		Id:         uuid.NewString(),
		RunId:      taskRunId,
		L1Status:   l1Status,
		L1Command:  verifyCmd,
		L1Output:   l1Output,
		RetryCount: 0,
		VerifiedAt: now,
	}

	if err := mcstore.InsertTaskRun(ctx, taskRun); err != nil {
		log.Printf("InsertTaskRun error: %v\n", err)
	}
	if err := mcstore.InsertVerifierResult(ctx, verifierResult); err != nil {
		log.Printf("InsertVerifierResult error: %v\n", err)
	}

	// 4. Determine new task status
	newStatus := mcstore.TaskStatusReview // default: hand off to human review
	reason := ""

	if l1Status == mcstore.VerifierStatusFail {
		newStatus = mcstore.TaskStatusBlocked
		reason = fmt.Sprintf("L1 verification failed: %s", verifyCmd)
	} else if handoff.Verdict == mcstore.HandoffStatusFail {
		newStatus = mcstore.TaskStatusBlocked
		reason = "agent self-reported FAIL in handoff.md"
	} else if handoff.Verdict == mcstore.HandoffStatusMissing {
		newStatus = mcstore.TaskStatusReview
		reason = "handoff.md not found — entering review without agent sign-off"
	}

	contextNotes := task.ContextNotes
	if reason != "" {
		contextNotes = task.ContextNotes + "\n\n[verifier] " + reason
	}

	if err := mcstore.UpdateTask(ctx, session.TaskId, map[string]any{
		"status":        newStatus,
		"context_notes": contextNotes,
	}); err != nil {
		log.Printf("UpdateTask error: %v\n", err)
	}
	if err := mcstore.UpdateSession(ctx, sessionId, map[string]any{
		"status": mcstore.SessionStatusDone,
	}); err != nil {
		log.Printf("UpdateSession error: %v\n", err)
	}

	// 5. Publish SSE event
	publishEvent("session.complete", map[string]string{
		"sessionid":  sessionId,
		"taskstatus": newStatus,
	})

	log.Printf("session %s complete: task=%s → %s (l1=%s, handoff=%s)\n",
		sessionId, session.TaskId, newStatus, l1Status, handoff.Verdict)

	writeJSON(w, http.StatusOK, map[string]string{
		"taskstatus":    newStatus,
		"l1status":      l1Status,
		"handoffstatus": handoff.Verdict,
		"runid":         taskRunId,
	})
}
