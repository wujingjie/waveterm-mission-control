// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atoms, createBlock, getApi } from "@/store/global";
import { globalStore } from "@/store/jotaiStore";
import { cn, fireAndForget } from "@/util/util";
import { useAtomValue } from "jotai";
import { memo, useCallback } from "react";
import { createIntent, createSession, MCTask, patchTask } from "./mc-api";
import { MCPanelModel } from "./mcpanel-model";

const StatusDot = memo(({ status }: { status: string }) => {
    const colors: Record<string, string> = {
        doing: "bg-blue-400",
        review: "bg-yellow-400",
        blocked: "bg-red-400",
        done: "bg-green-400",
        todo: "bg-zinc-500",
        parked: "bg-zinc-600",
    };
    return <span className={cn("inline-block w-2 h-2 rounded-full flex-shrink-0", colors[status] ?? "bg-zinc-500")} />;
});
StatusDot.displayName = "StatusDot";

const ExecutorBadge = memo(({ executor }: { executor: string }) => {
    if (!executor) return null;
    const labels: Record<string, string> = {
        claude: "Claude",
        codex: "Codex",
        gemini: "Gemini",
        opencode: "OC",
        manual: "Manual",
    };
    return (
        <span className="text-[10px] bg-zinc-700 text-zinc-300 px-1 rounded flex-shrink-0">
            {labels[executor] ?? executor}
        </span>
    );
});
ExecutorBadge.displayName = "ExecutorBadge";

const TaskCard = memo(({ task }: { task: MCTask }) => {
    const model = MCPanelModel.getInstance();

    const handleStart = useCallback(() => {
        fireAndForget(async () => {
            const ws = globalStore.get(atoms.workspace);
            const projectId = ws?.meta?.["mc:projectid"] as string;
            const repoPath = (ws?.meta?.["mc:repopath"] as string) ?? "";
            const apiUrl = getApi().getEnv("MC_API_URL") ?? "http://127.0.0.1:3001";
            const authKey = getApi().getEnv("MC_AUTH_KEY") ?? "";

            // Create intent in MC API (tracking record)
            await createIntent({
                type: "start-agent",
                projectid: projectId,
                taskid: task.id,
                payload: JSON.stringify({ executor: task.executor || "claude", cwd: repoPath }),
                status: "pending",
                createdby: "mc-panel",
            });

            // Create terminal block with agent command and inject it into the layout
            const command = getAgentCommand(task.executor || "claude");
            const blockDef: BlockDef = {
                meta: {
                    view: "term",
                    controller: "cmd",
                    cmd: command,
                    "cmd:interactive": true,
                    "cmd:cwd": repoPath || undefined,
                    "cmd:env": {
                        MC_PROJECT_ID: projectId,
                        MC_TASK_ID: task.id,
                        MC_API_URL: apiUrl,
                        MC_AUTH_KEY: authKey,
                    },
                },
            };
            // createBlock handles both CreateBlock RPC and inserting into current tab layout
            const blockId = await createBlock(blockDef);

            // Register session in MC API
            await createSession({
                projectid: projectId,
                taskid: task.id,
                provider: task.executor || "claude",
                terminalblockid: blockId,
                cwd: repoPath,
                command,
                status: "starting",
            });

            // Update task status to doing
            await patchTask(task.id, { status: "doing" });

            // Refresh panel
            model.loadData();
        });
    }, [task]);

    const isStartable = task.status === "todo" || task.status === "blocked";

    return (
        <div className="flex items-start gap-2 px-3 py-2 hover:bg-zinc-800/50 rounded group">
            <StatusDot status={task.status} />
            <div className="flex-1 min-w-0">
                <div className="text-sm text-zinc-100 truncate">{task.title}</div>
                {task.phase && <div className="text-[10px] text-zinc-500">{task.phase}</div>}
            </div>
            <ExecutorBadge executor={task.executor} />
            {isStartable && (
                <button
                    onClick={handleStart}
                    className="text-[10px] bg-accent/80 text-primary rounded px-1.5 py-0.5 hover:bg-accent transition-colors cursor-pointer opacity-0 group-hover:opacity-100 flex-shrink-0"
                    title="开工"
                >
                    开工
                </button>
            )}
        </div>
    );
});
TaskCard.displayName = "TaskCard";

function getAgentCommand(executor: string): string {
    const commands: Record<string, string> = {
        claude: "claude",
        codex: "codex",
        gemini: "gemini",
        opencode: "opencode",
        manual: "",
    };
    return commands[executor] ?? executor;
}

const TaskGroup = memo(({ title, tasks, defaultOpen = true }: { title: string; tasks: MCTask[]; defaultOpen?: boolean }) => {
    if (tasks.length === 0) return null;
    return (
        <div className="mb-2">
            <div className="flex items-center gap-1.5 px-3 py-1 text-[11px] text-zinc-400 font-semibold uppercase tracking-wide">
                <span>{title}</span>
                <span className="text-zinc-600">({tasks.length})</span>
            </div>
            {tasks.map((t) => (
                <TaskCard key={t.id} task={t} />
            ))}
        </div>
    );
});
TaskGroup.displayName = "TaskGroup";

export const MCPanelTasks = memo(() => {
    const model = MCPanelModel.getInstance();
    const tasks = useAtomValue(model.tasksAtom);
    const loading = useAtomValue(model.loadingAtom);
    const error = useAtomValue(model.errorAtom);
    const ws = useAtomValue(atoms.workspace);
    const projectId = ws?.meta?.["mc:projectid"] as string;
    const projectName = ws?.meta?.["mc:projectname"] as string;

    if (!projectId) {
        return (
            <div className="flex flex-col items-center justify-center h-full text-zinc-500 px-4 text-center">
                <i className="fa fa-satellite-dish text-2xl mb-2" />
                <div className="text-sm">No project bound</div>
                <div className="text-xs mt-1">Click the workspace name → edit → set MC Project</div>
            </div>
        );
    }

    const doing = tasks.filter((t) => t.status === "doing");
    const review = tasks.filter((t) => t.status === "review");
    const blocked = tasks.filter((t) => t.status === "blocked");
    const todo = tasks.filter((t) => t.status === "todo");

    return (
        <div className="flex flex-col h-full">
            <div className="px-3 py-2 border-b border-zinc-700 flex items-center justify-between">
                <span className="text-xs font-semibold text-zinc-300 truncate">{projectName || projectId}</span>
                <button
                    onClick={() => model.loadData()}
                    className="text-zinc-500 hover:text-zinc-300 cursor-pointer transition-colors"
                    title="Refresh"
                >
                    <i className={cn("fa fa-refresh text-xs", loading && "animate-spin")} />
                </button>
            </div>
            {error && (
                <div className="px-3 py-2 text-xs text-red-400 bg-red-900/20 border-b border-red-800">
                    {error}
                </div>
            )}
            <div className="flex-1 overflow-y-auto py-1">
                {loading && tasks.length === 0 ? (
                    <div className="flex items-center justify-center h-20 text-zinc-500 text-sm">Loading…</div>
                ) : (
                    <>
                        <TaskGroup title="Doing" tasks={doing} />
                        <TaskGroup title="Review" tasks={review} />
                        <TaskGroup title="Blocked" tasks={blocked} />
                        <TaskGroup title="Todo" tasks={todo} />
                        {tasks.length === 0 && (
                            <div className="text-center text-zinc-600 text-sm py-8">No tasks</div>
                        )}
                    </>
                )}
            </div>
        </div>
    );
});
MCPanelTasks.displayName = "MCPanelTasks";
