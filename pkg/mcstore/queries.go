// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package mcstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// allowedColumns defines the safe set of column names for each table's PATCH endpoint.
// Keys entered by API callers are validated against this list before being spliced into SQL.
var allowedColumns = map[string]map[string]bool{
	"projects": {
		"name": true, "description": true, "repo_path": true, "obsidian_path": true,
	},
	"tasks": {
		"title": true, "description": true, "status": true, "priority": true,
		"executor": true, "depends_on": true, "context_notes": true, "phase": true,
		"phase_order": true, "scheduled_at": true, "auto_trigger": true,
		"batch_id": true, "worktree_path": true, "base_commit": true, "updated_at": true,
	},
	"agent_sessions": {
		"status": true, "last_seen_at": true, "transcript_path": true,
		"terminal_block_id": true, "run_id": true,
	},
	"intents": {
		"status": true, "executed_at": true, "retry_count": true, "error_message": true,
		"claimed_by": true, "claimed_at": true, "lease_expires_at": true,
	},
}

func buildUpdateQuery(table string, id string, fields map[string]any) (string, []any, error) {
	allowed := allowedColumns[table]
	setClauses := []string{}
	args := []any{}
	for k, v := range fields {
		if allowed != nil && !allowed[k] {
			return "", nil, fmt.Errorf("column %q is not allowed in UPDATE %s", k, table)
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}
	if len(setClauses) == 0 {
		return "", nil, fmt.Errorf("no valid fields to update")
	}
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", table, strings.Join(setClauses, ", "))
	args = append(args, id)
	return query, args, nil
}

var globalDB *sqlx.DB

func SetDB(db *sqlx.DB) {
	globalDB = db
}

func GetDB() *sqlx.DB {
	return globalDB
}

func newId() string {
	return uuid.New().String()
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Projects

func GetAllProjects(ctx context.Context) ([]*Project, error) {
	projects := []*Project{}
	err := globalDB.SelectContext(ctx, &projects, `SELECT * FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func GetProject(ctx context.Context, id string) (*Project, error) {
	var p Project
	err := globalDB.GetContext(ctx, &p, `SELECT * FROM projects WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func InsertProject(ctx context.Context, p *Project) error {
	if p.Id == "" {
		p.Id = newId()
	}
	if p.CreatedAt == "" {
		p.CreatedAt = nowISO()
	}
	_, err := globalDB.NamedExecContext(ctx, `
		INSERT INTO projects (id, name, description, repo_path, obsidian_path, created_at)
		VALUES (:id, :name, :description, :repo_path, :obsidian_path, :created_at)
	`, p)
	return err
}

func UpdateProject(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	query, args, err := buildUpdateQuery("projects", id, fields)
	if err != nil {
		return err
	}
	_, err = globalDB.ExecContext(ctx, query, args...)
	return err
}

// Tasks

func GetTasksByProject(ctx context.Context, projectId string, status string) ([]*Task, error) {
	tasks := []*Task{}
	query := `SELECT * FROM tasks WHERE project_id = ?`
	args := []any{projectId}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY phase_order ASC, created_at ASC`
	err := globalDB.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func GetTask(ctx context.Context, id string) (*Task, error) {
	var t Task
	err := globalDB.GetContext(ctx, &t, `SELECT * FROM tasks WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func InsertTask(ctx context.Context, t *Task) error {
	if t.Id == "" {
		t.Id = newId()
	}
	now := nowISO()
	if t.CreatedAt == "" {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	_, err := globalDB.NamedExecContext(ctx, `
		INSERT INTO tasks (id, project_id, title, description, status, priority, executor,
			depends_on, context_notes, phase, phase_order, source_document_id, split_order,
			scheduled_at, auto_trigger, batch_id, worktree_path, base_commit, created_at, updated_at)
		VALUES (:id, :project_id, :title, :description, :status, :priority, :executor,
			:depends_on, :context_notes, :phase, :phase_order, :source_document_id, :split_order,
			:scheduled_at, :auto_trigger, :batch_id, :worktree_path, :base_commit, :created_at, :updated_at)
	`, t)
	return err
}

func UpdateTask(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = nowISO()
	query, args, err := buildUpdateQuery("tasks", id, fields)
	if err != nil {
		return err
	}
	_, err = globalDB.ExecContext(ctx, query, args...)
	return err
}

// AgentSessions

func GetSessionsByProject(ctx context.Context, projectId string, status string) ([]*AgentSession, error) {
	sessions := []*AgentSession{}
	query := `SELECT * FROM agent_sessions WHERE project_id = ?`
	args := []any{projectId}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY started_at DESC`
	err := globalDB.SelectContext(ctx, &sessions, query, args...)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func InsertSession(ctx context.Context, s *AgentSession) error {
	if s.Id == "" {
		s.Id = newId()
	}
	now := nowISO()
	if s.StartedAt == "" {
		s.StartedAt = now
	}
	s.LastSeenAt = now
	_, err := globalDB.NamedExecContext(ctx, `
		INSERT INTO agent_sessions (id, project_id, task_id, provider, terminal_block_id,
			cwd, command, status, transcript_path, started_at, last_seen_at)
		VALUES (:id, :project_id, :task_id, :provider, :terminal_block_id,
			:cwd, :command, :status, :transcript_path, :started_at, :last_seen_at)
	`, s)
	return err
}

func UpdateSession(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	query, args, err := buildUpdateQuery("agent_sessions", id, fields)
	if err != nil {
		return err
	}
	_, err = globalDB.ExecContext(ctx, query, args...)
	return err
}

func UpdateSessionHeartbeat(ctx context.Context, id string) error {
	_, err := globalDB.ExecContext(ctx,
		`UPDATE agent_sessions SET last_seen_at = ? WHERE id = ?`,
		nowISO(), id,
	)
	return err
}

func MarkStaleSessionsOlderThan(ctx context.Context, seconds int) (int64, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(seconds) * time.Second).Format(time.RFC3339)
	res, err := globalDB.ExecContext(ctx,
		`UPDATE agent_sessions SET status = 'stale'
		 WHERE status IN ('running', 'starting') AND last_seen_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Intents

func GetIntents(ctx context.Context, projectId string, status string) ([]*Intent, error) {
	intents := []*Intent{}
	query := `SELECT * FROM intents WHERE 1=1`
	args := []any{}
	if projectId != "" {
		query += ` AND project_id = ?`
		args = append(args, projectId)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at ASC`
	err := globalDB.SelectContext(ctx, &intents, query, args...)
	if err != nil {
		return nil, err
	}
	return intents, nil
}

func InsertIntent(ctx context.Context, intent *Intent) error {
	if intent.Id == "" {
		intent.Id = newId()
	}
	if intent.CreatedAt == "" {
		intent.CreatedAt = nowISO()
	}
	_, err := globalDB.NamedExecContext(ctx, `
		INSERT INTO intents (id, type, project_id, task_id, payload, status, created_by,
			created_at, idempotency_key, target_workspace_id, retry_count, error_message,
			claimed_by, claimed_at, lease_expires_at)
		VALUES (:id, :type, :project_id, :task_id, :payload, :status, :created_by,
			:created_at, :idempotency_key, :target_workspace_id, :retry_count, :error_message,
			:claimed_by, :claimed_at, :lease_expires_at)
	`, intent)
	return err
}

func UpdateIntent(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	query, args, err := buildUpdateQuery("intents", id, fields)
	if err != nil {
		return err
	}
	_, err = globalDB.ExecContext(ctx, query, args...)
	return err
}

// ClaimIntent atomically claims a specific intent (pending → claimed).
// Returns false if the intent was already claimed or doesn't exist.
func ClaimIntent(ctx context.Context, intentId string, claimedBy string) (bool, error) {
	now := nowISO()
	leaseExpires := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)
	res, err := globalDB.ExecContext(ctx, `
		UPDATE intents
		SET status = 'claimed', claimed_by = ?, claimed_at = ?, lease_expires_at = ?
		WHERE id = ? AND status = 'pending'
	`, claimedBy, now, leaseExpires, intentId)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	return rows == 1, err
}

// ClaimNextPendingIntent atomically claims the oldest pending intent.
// Returns nil if no pending intents exist.
func ClaimNextPendingIntent(ctx context.Context, claimedBy string) (*Intent, error) {
	tx, err := globalDB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var intent Intent
	err = tx.GetContext(ctx, &intent,
		`SELECT * FROM intents WHERE status = 'pending' ORDER BY created_at ASC LIMIT 1`,
	)
	if err != nil {
		return nil, nil // no pending intents
	}

	now := nowISO()
	leaseExpires := time.Now().UTC().Add(5 * time.Minute).Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
		UPDATE intents
		SET status = 'claimed', claimed_by = ?, claimed_at = ?, lease_expires_at = ?
		WHERE id = ? AND status = 'pending'
	`, claimedBy, now, leaseExpires, intent.Id)
	if err != nil {
		return nil, err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, nil
	}

	intent.Status = IntentStatusClaimed
	intent.ClaimedBy = claimedBy
	intent.ClaimedAt = &now
	intent.LeaseExpiresAt = &leaseExpires

	return &intent, tx.Commit()
}

// Activities

func GetActivities(ctx context.Context, projectId string, taskId string, limit int) ([]*Activity, error) {
	activities := []*Activity{}
	query := `SELECT * FROM activities WHERE 1=1`
	args := []any{}
	if projectId != "" {
		query += ` AND project_id = ?`
		args = append(args, projectId)
	}
	if taskId != "" {
		query += ` AND task_id = ?`
		args = append(args, taskId)
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}
	err := globalDB.SelectContext(ctx, &activities, query, args...)
	if err != nil {
		return nil, err
	}
	return activities, nil
}

func InsertActivity(ctx context.Context, a *Activity) error {
	if a.Id == "" {
		a.Id = newId()
	}
	if a.CreatedAt == "" {
		a.CreatedAt = nowISO()
	}
	_, err := globalDB.NamedExecContext(ctx, `
		INSERT INTO activities (id, project_id, task_id, session_id, type, summary, created_at)
		VALUES (:id, :project_id, :task_id, :session_id, :type, :summary, :created_at)
	`, a)
	return err
}

func GetSession(ctx context.Context, id string) (*AgentSession, error) {
	var s AgentSession
	err := globalDB.GetContext(ctx, &s, `SELECT * FROM agent_sessions WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func InsertTaskRun(ctx context.Context, r *TaskRun) error {
	if r.Id == "" {
		r.Id = newId()
	}
	if r.StartedAt == "" {
		r.StartedAt = nowISO()
	}
	_, err := globalDB.NamedExecContext(ctx, `
		INSERT INTO task_runs
			(id, task_id, session_id, run_number, pipeline_mode, handoff_path, handoff_status, verdict, base_commit, head_commit, started_at, finished_at)
		VALUES
			(:id, :task_id, :session_id, :run_number, :pipeline_mode, :handoff_path, :handoff_status, :verdict, :base_commit, :head_commit, :started_at, :finished_at)
	`, r)
	return err
}

func InsertVerifierResult(ctx context.Context, v *VerifierResult) error {
	if v.Id == "" {
		v.Id = newId()
	}
	if v.VerifiedAt == "" {
		v.VerifiedAt = nowISO()
	}
	_, err := globalDB.NamedExecContext(ctx, `
		INSERT INTO verifier_results (id, run_id, l1_status, l1_command, l1_output, retry_count, verified_at)
		VALUES (:id, :run_id, :l1_status, :l1_command, :l1_output, :retry_count, :verified_at)
	`, v)
	return err
}
