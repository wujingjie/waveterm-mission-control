// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { getApi } from "@/store/global";

export type MCProject = {
    id: string;
    name: string;
    description: string;
    repopath: string;
    obsidianpath: string;
    createdat: string;
};

export type MCTask = {
    id: string;
    projectid: string;
    title: string;
    description: string;
    status: string;
    priority: string;
    executor: string;
    dependson: string;
    contextnotes: string;
    phase: string;
    phaseorder: number;
    createdat: string;
    updatedat: string;
};

export type MCSession = {
    id: string;
    projectid: string;
    taskid: string;
    provider: string;
    terminalblockid: string;
    cwd: string;
    command: string;
    status: string;
    startedat: string;
    lastsleenat: string;
};

export type MCIntent = {
    id: string;
    type: string;
    projectid: string;
    taskid: string;
    payload: string;
    status: string;
    createdby: string;
    targetworkspaceid: string;
};

function getMCBase(): { apiUrl: string; authKey: string } {
    const apiUrl = getApi().getEnv("MC_API_URL") ?? "http://127.0.0.1:3001";
    const authKey = getApi().getEnv("MC_AUTH_KEY") ?? "";
    return { apiUrl, authKey };
}

async function mcFetch(path: string, opts: RequestInit = {}): Promise<any> {
    const { apiUrl, authKey } = getMCBase();
    if (!authKey) throw new Error("MC_AUTH_KEY not set");
    const res = await fetch(`${apiUrl}${path}`, {
        ...opts,
        headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${authKey}`,
            ...(opts.headers ?? {}),
        },
    });
    if (!res.ok) throw new Error(`MC API ${path}: ${res.status}`);
    if (res.status === 204) return null;
    return res.json();
}

export async function fetchTasks(projectId: string): Promise<MCTask[]> {
    return mcFetch(`/api/tasks?project_id=${encodeURIComponent(projectId)}`);
}

export async function fetchSessions(projectId: string): Promise<MCSession[]> {
    return mcFetch(`/api/sessions?project_id=${encodeURIComponent(projectId)}`);
}

export async function fetchProject(projectId: string): Promise<MCProject> {
    return mcFetch(`/api/projects/${encodeURIComponent(projectId)}`);
}

export async function patchTask(taskId: string, patch: Partial<MCTask>): Promise<MCTask> {
    return mcFetch(`/api/tasks/${encodeURIComponent(taskId)}`, {
        method: "PATCH",
        body: JSON.stringify(patch),
    });
}

export async function createIntent(intent: Partial<MCIntent>): Promise<MCIntent> {
    return mcFetch("/api/intents", { method: "POST", body: JSON.stringify(intent) });
}

export async function createSession(session: Partial<MCSession>): Promise<MCSession> {
    return mcFetch("/api/sessions", { method: "POST", body: JSON.stringify(session) });
}

export function openSSE(onEvent: (data: string) => void): () => void {
    const { apiUrl, authKey } = getMCBase();
    if (!authKey) return () => {};
    const url = `${apiUrl}/api/events?auth=${encodeURIComponent(authKey)}`;
    const es = new EventSource(url);
    es.onmessage = (e) => onEvent(e.data);
    es.onerror = () => {};
    return () => es.close();
}
