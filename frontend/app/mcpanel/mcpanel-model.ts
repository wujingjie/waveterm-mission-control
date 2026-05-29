// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atoms } from "@/store/global";
import { globalStore } from "@/store/jotaiStore";
import * as jotai from "jotai";
import { fetchSessions, fetchTasks, MCSession, MCTask, openSSE } from "./mc-api";

export class MCPanelModel {
    private static instance: MCPanelModel | null = null;

    tasksAtom = jotai.atom<MCTask[]>([]) as jotai.PrimitiveAtom<MCTask[]>;
    sessionsAtom = jotai.atom<MCSession[]>([]) as jotai.PrimitiveAtom<MCSession[]>;
    loadingAtom = jotai.atom(false) as jotai.PrimitiveAtom<boolean>;
    errorAtom = jotai.atom(null) as jotai.PrimitiveAtom<string>;

    private sseCleanup: (() => void) | null = null;
    private lastProjectId: string | null = null;

    private constructor() {}

    static getInstance(): MCPanelModel {
        if (!MCPanelModel.instance) {
            MCPanelModel.instance = new MCPanelModel();
        }
        return MCPanelModel.instance;
    }

    getProjectId(): string | null {
        const ws = globalStore.get(atoms.workspace);
        return (ws?.meta?.["mc:projectid"] as string) || null;
    }

    async loadData(): Promise<void> {
        const projectId = this.getProjectId();
        if (!projectId) {
            globalStore.set(this.tasksAtom, []);
            globalStore.set(this.sessionsAtom, []);
            return;
        }
        globalStore.set(this.loadingAtom, true);
        globalStore.set(this.errorAtom, null);
        try {
            const [tasks, sessions] = await Promise.all([fetchTasks(projectId), fetchSessions(projectId)]);
            globalStore.set(this.tasksAtom, tasks ?? []);
            globalStore.set(this.sessionsAtom, sessions ?? []);
        } catch (e: any) {
            globalStore.set(this.errorAtom, e?.message ?? "Failed to load MC data");
        } finally {
            globalStore.set(this.loadingAtom, false);
        }
    }

    startSSE(): void {
        this.stopSSE();
        this.sseCleanup = openSSE(() => {
            // Any event triggers a refresh
            this.loadData();
        });
    }

    stopSSE(): void {
        this.sseCleanup?.();
        this.sseCleanup = null;
    }

    syncProjectId(): void {
        const projectId = this.getProjectId();
        if (projectId !== this.lastProjectId) {
            this.lastProjectId = projectId;
            this.loadData();
        }
    }

    getTasksByStatus(status: string): MCTask[] {
        return globalStore.get(this.tasksAtom).filter((t) => t.status === status);
    }
}
