-- Copyright 2026, Command Line Inc.
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE agent_sessions ADD COLUMN run_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS task_runs (
    id            TEXT PRIMARY KEY,
    task_id       TEXT NOT NULL REFERENCES tasks(id),
    session_id    TEXT NOT NULL DEFAULT '',
    run_number    INTEGER NOT NULL DEFAULT 1,
    pipeline_mode TEXT NOT NULL DEFAULT 'standard',
    handoff_path  TEXT NOT NULL DEFAULT '',
    handoff_status TEXT NOT NULL DEFAULT 'missing',
    verdict       TEXT NOT NULL DEFAULT '',
    base_commit   TEXT NOT NULL DEFAULT '',
    head_commit   TEXT NOT NULL DEFAULT '',
    started_at    TEXT NOT NULL DEFAULT (datetime('now')),
    finished_at   TEXT
);

CREATE INDEX IF NOT EXISTS idx_task_runs_task_id ON task_runs(task_id);
CREATE INDEX IF NOT EXISTS idx_task_runs_session_id ON task_runs(session_id);

CREATE TABLE IF NOT EXISTS verifier_results (
    id          TEXT PRIMARY KEY,
    run_id      TEXT NOT NULL REFERENCES task_runs(id),
    l1_status   TEXT NOT NULL DEFAULT 'skip',
    l1_command  TEXT NOT NULL DEFAULT '',
    l1_output   TEXT NOT NULL DEFAULT '',
    retry_count INTEGER NOT NULL DEFAULT 0,
    verified_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_verifier_results_run_id ON verifier_results(run_id);
