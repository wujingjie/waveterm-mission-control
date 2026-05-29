// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package mcstore

const (
	TaskStatusTodo    = "todo"
	TaskStatusDoing   = "doing"
	TaskStatusReview  = "review"
	TaskStatusDone    = "done"
	TaskStatusBlocked = "blocked"
	TaskStatusParked  = "parked"

	TaskPriorityHigh   = "high"
	TaskPriorityMedium = "medium"
	TaskPriorityLow    = "low"

	IntentStatusPending   = "pending"
	IntentStatusClaimed   = "claimed"
	IntentStatusExecuting = "executing"
	IntentStatusDone      = "done"
	IntentStatusFailed    = "failed"

	SessionStatusStarting       = "starting"
	SessionStatusRunning        = "running"
	SessionStatusWaitingInput   = "waiting_input"
	SessionStatusNeedsApproval  = "needs_approval"
	SessionStatusStale          = "stale"
	SessionStatusDone           = "done"
	SessionStatusFailed         = "failed"
	SessionStatusCancelled      = "cancelled"

	DocTypeCommunication = "communication"
	DocTypeExecution     = "execution"
	DocTypeProduct       = "product"
	DocTypeReview        = "review"

	DocStatusDraft    = "draft"
	DocStatusFrozen   = "frozen"
	DocStatusArchived = "archived"
)

type Project struct {
	Id           string `db:"id"            json:"id"`
	Name         string `db:"name"          json:"name"`
	Description  string `db:"description"   json:"description"`
	RepoPath     string `db:"repo_path"     json:"repopath"`
	ObsidianPath string `db:"obsidian_path" json:"obsidianpath"`
	CreatedAt    string `db:"created_at"    json:"createdat"`
}

type Task struct {
	Id               string  `db:"id"                 json:"id"`
	ProjectId        string  `db:"project_id"         json:"projectid"`
	Title            string  `db:"title"              json:"title"`
	Description      string  `db:"description"        json:"description"`
	Status           string  `db:"status"             json:"status"`
	Priority         string  `db:"priority"           json:"priority"`
	Executor         string  `db:"executor"           json:"executor"`
	DependsOn        string  `db:"depends_on"         json:"dependson"`
	ContextNotes     string  `db:"context_notes"      json:"contextnotes"`
	Phase            string  `db:"phase"              json:"phase"`
	PhaseOrder       int     `db:"phase_order"        json:"phaseorder"`
	SourceDocumentId string  `db:"source_document_id" json:"sourcedocumentid"`
	SplitOrder       int     `db:"split_order"        json:"splitorder"`
	ScheduledAt      *string `db:"scheduled_at"       json:"scheduledat,omitempty"`
	AutoTrigger      bool    `db:"auto_trigger"       json:"autotrigger"`
	BatchId          string  `db:"batch_id"           json:"batchid"`
	WorktreePath     string  `db:"worktree_path"      json:"worktreepath"`
	BaseCommit       string  `db:"base_commit"        json:"basecommit"`
	CreatedAt        string  `db:"created_at"         json:"createdat"`
	UpdatedAt        string  `db:"updated_at"         json:"updatedat"`
}

type AgentSession struct {
	Id              string `db:"id"                 json:"id"`
	ProjectId       string `db:"project_id"         json:"projectid"`
	TaskId          string `db:"task_id"            json:"taskid"`
	RunId           string `db:"run_id"             json:"runid"`
	Provider        string `db:"provider"           json:"provider"`
	TerminalBlockId string `db:"terminal_block_id"  json:"terminalblockid"`
	Cwd             string `db:"cwd"                json:"cwd"`
	Command         string `db:"command"            json:"command"`
	Status          string `db:"status"             json:"status"`
	TranscriptPath  string `db:"transcript_path"    json:"transcriptpath"`
	StartedAt       string `db:"started_at"         json:"startedat"`
	LastSeenAt      string `db:"last_seen_at"       json:"lastsleenat"`
}

const (
	HandoffStatusMissing  = "missing"
	HandoffStatusPass     = "pass"
	HandoffStatusFail     = "fail"
	HandoffStatusIncomplete = "incomplete"

	VerifierStatusPass = "pass"
	VerifierStatusFail = "fail"
	VerifierStatusSkip = "skip"
)

type TaskRun struct {
	Id           string  `db:"id"            json:"id"`
	TaskId       string  `db:"task_id"       json:"taskid"`
	SessionId    string  `db:"session_id"    json:"sessionid"`
	RunNumber    int     `db:"run_number"    json:"runnumber"`
	PipelineMode string  `db:"pipeline_mode" json:"pipelinemode"`
	HandoffPath  string  `db:"handoff_path"  json:"handoffpath"`
	HandoffStatus string `db:"handoff_status" json:"handoffstatus"`
	Verdict      string  `db:"verdict"       json:"verdict"`
	BaseCommit   string  `db:"base_commit"   json:"basecommit"`
	HeadCommit   string  `db:"head_commit"   json:"headcommit"`
	StartedAt    string  `db:"started_at"    json:"startedat"`
	FinishedAt   *string `db:"finished_at"   json:"finishedat,omitempty"`
}

type VerifierResult struct {
	Id         string `db:"id"          json:"id"`
	RunId      string `db:"run_id"      json:"runid"`
	L1Status   string `db:"l1_status"   json:"l1status"`
	L1Command  string `db:"l1_command"  json:"l1command"`
	L1Output   string `db:"l1_output"   json:"l1output"`
	RetryCount int    `db:"retry_count" json:"retrycount"`
	VerifiedAt string `db:"verified_at" json:"verifiedat"`
}

type Document struct {
	Id              string `db:"id"                 json:"id"`
	ProjectId       string `db:"project_id"         json:"projectid"`
	DocType         string `db:"doc_type"           json:"doctype"`
	Path            string `db:"path"               json:"path"`
	Title           string `db:"title"              json:"title"`
	SourceSessionId string `db:"source_session_id"  json:"sourcesessionid"`
	Status          string `db:"status"             json:"status"`
	CreatedAt       string `db:"created_at"         json:"createdat"`
}

type Intent struct {
	Id                string  `db:"id"                   json:"id"`
	Type              string  `db:"type"                 json:"type"`
	ProjectId         string  `db:"project_id"           json:"projectid"`
	TaskId            string  `db:"task_id"              json:"taskid"`
	Payload           string  `db:"payload"              json:"payload"`
	Status            string  `db:"status"               json:"status"`
	CreatedBy         string  `db:"created_by"           json:"createdby"`
	CreatedAt         string  `db:"created_at"           json:"createdat"`
	ExecutedAt        *string `db:"executed_at"          json:"executedat,omitempty"`
	IdempotencyKey    string  `db:"idempotency_key"      json:"idempotencykey"`
	TargetWorkspaceId string  `db:"target_workspace_id"  json:"targetworkspaceid"`
	RetryCount        int     `db:"retry_count"          json:"retrycount"`
	ErrorMessage      string  `db:"error_message"        json:"errormessage"`
	ClaimedBy         string  `db:"claimed_by"           json:"claimedby"`
	ClaimedAt         *string `db:"claimed_at"           json:"claimedat,omitempty"`
	LeaseExpiresAt    *string `db:"lease_expires_at"     json:"leaseexpiresat,omitempty"`
}

type Activity struct {
	Id        string `db:"id"         json:"id"`
	ProjectId string `db:"project_id" json:"projectid"`
	TaskId    string `db:"task_id"    json:"taskid"`
	SessionId string `db:"session_id" json:"sessionid"`
	Type      string `db:"type"       json:"type"`
	Summary   string `db:"summary"    json:"summary"`
	CreatedAt string `db:"created_at" json:"createdat"`
}
