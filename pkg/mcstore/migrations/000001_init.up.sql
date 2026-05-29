-- Copyright 2026, Command Line Inc.
-- SPDX-License-Identifier: Apache-2.0

CREATE TABLE IF NOT EXISTS projects (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    repo_path   TEXT NOT NULL DEFAULT '',
    obsidian_path TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS tasks (
    id                 TEXT PRIMARY KEY,
    project_id         TEXT NOT NULL REFERENCES projects(id),
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'todo',
    priority           TEXT NOT NULL DEFAULT 'medium',
    executor           TEXT NOT NULL DEFAULT '',
    depends_on         TEXT NOT NULL DEFAULT '[]',
    context_notes      TEXT NOT NULL DEFAULT '',
    phase              TEXT NOT NULL DEFAULT '',
    phase_order        INTEGER NOT NULL DEFAULT 0,
    source_document_id TEXT NOT NULL DEFAULT '',
    split_order        INTEGER NOT NULL DEFAULT 0,
    scheduled_at       TEXT,
    auto_trigger       INTEGER NOT NULL DEFAULT 0,
    batch_id           TEXT NOT NULL DEFAULT '',
    worktree_path      TEXT NOT NULL DEFAULT '',
    base_commit        TEXT NOT NULL DEFAULT '',
    created_at         TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at         TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_phase ON tasks(phase);

CREATE TABLE IF NOT EXISTS agent_sessions (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects(id),
    task_id           TEXT NOT NULL DEFAULT '',
    provider          TEXT NOT NULL DEFAULT 'manual',
    terminal_block_id TEXT NOT NULL DEFAULT '',
    cwd               TEXT NOT NULL DEFAULT '',
    command           TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'starting',
    transcript_path   TEXT NOT NULL DEFAULT '',
    started_at        TEXT NOT NULL DEFAULT (datetime('now')),
    last_seen_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON agent_sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_sessions_task_id ON agent_sessions(task_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON agent_sessions(status);

CREATE TABLE IF NOT EXISTS documents (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects(id),
    doc_type          TEXT NOT NULL DEFAULT 'execution',
    path              TEXT NOT NULL,
    title             TEXT NOT NULL DEFAULT '',
    source_session_id TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'draft',
    created_at        TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_documents_project_id ON documents(project_id);

CREATE TABLE IF NOT EXISTS task_documents (
    task_id     TEXT NOT NULL REFERENCES tasks(id),
    document_id TEXT NOT NULL REFERENCES documents(id),
    PRIMARY KEY (task_id, document_id)
);

CREATE TABLE IF NOT EXISTS intents (
    id                  TEXT PRIMARY KEY,
    type                TEXT NOT NULL,
    project_id          TEXT NOT NULL,
    task_id             TEXT NOT NULL DEFAULT '',
    payload             TEXT NOT NULL DEFAULT '{}',
    status              TEXT NOT NULL DEFAULT 'pending',
    created_by          TEXT NOT NULL DEFAULT '',
    created_at          TEXT NOT NULL DEFAULT (datetime('now')),
    executed_at         TEXT,
    idempotency_key     TEXT NOT NULL DEFAULT '',
    target_workspace_id TEXT NOT NULL DEFAULT '',
    retry_count         INTEGER NOT NULL DEFAULT 0,
    error_message       TEXT NOT NULL DEFAULT '',
    claimed_by          TEXT NOT NULL DEFAULT '',
    claimed_at          TEXT,
    lease_expires_at    TEXT
);

CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
CREATE INDEX IF NOT EXISTS idx_intents_project_id ON intents(project_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_intents_idempotency ON intents(idempotency_key)
    WHERE idempotency_key != '';

CREATE TABLE IF NOT EXISTS activities (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    task_id    TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL,
    summary    TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_activities_project_id ON activities(project_id);
CREATE INDEX IF NOT EXISTS idx_activities_task_id ON activities(task_id);
