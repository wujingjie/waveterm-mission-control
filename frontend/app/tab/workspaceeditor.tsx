// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { atoms, getApi } from "@/app/store/global";
import { RpcApi } from "@/app/store/wshclientapi";
import { TabRpcClient } from "@/app/store/wshrpcutil";
import { fireAndForget, makeIconClass } from "@/util/util";
import * as WOS from "@/store/wos";
import clsx from "clsx";
import { useAtomValue } from "jotai";
import { memo, useEffect, useRef, useState } from "react";
import { Button } from "../element/button";
import { Input } from "../element/input";
import { WorkspaceService } from "../store/services";
import "./workspaceeditor.scss";

interface ColorSelectorProps {
    colors: string[];
    selectedColor?: string;
    onSelect: (color: string) => void;
    className?: string;
}

const ColorSelector = memo(({ colors, selectedColor, onSelect, className }: ColorSelectorProps) => {
    return (
        <div className={clsx("color-selector", className)}>
            {colors.map((color) => (
                <div
                    key={color}
                    className={clsx("color-circle", { selected: selectedColor === color })}
                    style={{ backgroundColor: color }}
                    onClick={() => onSelect(color)}
                />
            ))}
        </div>
    );
});
ColorSelector.displayName = "ColorSelector";

interface IconSelectorProps {
    icons: string[];
    selectedIcon?: string;
    onSelect: (icon: string) => void;
    className?: string;
}

const IconSelector = memo(({ icons, selectedIcon, onSelect, className }: IconSelectorProps) => {
    return (
        <div className={clsx("icon-selector", className)}>
            {icons.map((icon) => {
                const iconClass = makeIconClass(icon, true);
                return (
                    <i
                        key={icon}
                        className={clsx(iconClass, "icon-item", { selected: selectedIcon === icon })}
                        onClick={() => onSelect(icon)}
                    />
                );
            })}
        </div>
    );
});
IconSelector.displayName = "IconSelector";

type MCProject = {
    id: string;
    name: string;
    repopath: string;
};

const MCProjectSelector = memo(({ workspaceId }: { workspaceId: string }) => {
    const ws = useAtomValue(atoms.workspace);
    const [projects, setProjects] = useState<MCProject[]>([]);
    const [loading, setLoading] = useState(false);
    const selectedId = (ws?.meta?.["mc:projectid"] as string) ?? "";

    useEffect(() => {
        const apiUrl = getApi().getEnv("MC_API_URL") ?? "http://127.0.0.1:3001";
        const authKey = getApi().getEnv("MC_AUTH_KEY") ?? "";
        if (!authKey) return;
        setLoading(true);
        fetch(`${apiUrl}/api/projects`, { headers: { Authorization: `Bearer ${authKey}` } })
            .then((r) => (r.ok ? r.json() : []))
            .then((data: MCProject[]) => setProjects(data ?? []))
            .catch(() => setProjects([]))
            .finally(() => setLoading(false));
    }, []);

    const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
        const projectId = e.target.value;
        const project = projects.find((p) => p.id === projectId);
        const meta: Record<string, string> = {
            "mc:projectid": projectId,
            "mc:projectname": project?.name ?? "",
            "mc:repopath": project?.repopath ?? "",
        };
        if (!projectId) {
            meta["mc:projectid"] = "";
            meta["mc:projectname"] = "";
            meta["mc:repopath"] = "";
        }
        fireAndForget(() =>
            RpcApi.SetMetaCommand(TabRpcClient, {
                oref: WOS.makeORef("workspace", workspaceId),
                meta: meta as any,
            })
        );
    };

    return (
        <div className="mt-2">
            <div className="text-xs text-gray-400 mb-1">MC Project</div>
            {loading ? (
                <div className="text-xs text-gray-500">Loading…</div>
            ) : (
                <select
                    value={selectedId}
                    onChange={handleChange}
                    className="w-full bg-zinc-800 border border-zinc-600 text-white text-sm rounded px-2 py-1 cursor-pointer"
                >
                    <option value="">— None —</option>
                    {projects.map((p) => (
                        <option key={p.id} value={p.id}>
                            {p.name}
                        </option>
                    ))}
                </select>
            )}
            {selectedId && (
                <div className="text-xs text-gray-500 mt-1 truncate">
                    {(ws?.meta?.["mc:repopath"] as string) ?? ""}
                </div>
            )}
        </div>
    );
});
MCProjectSelector.displayName = "MCProjectSelector";

interface WorkspaceEditorProps {
    title: string;
    icon: string;
    color: string;
    focusInput: boolean;
    workspaceId: string;
    onTitleChange: (newTitle: string) => void;
    onColorChange: (newColor: string) => void;
    onIconChange: (newIcon: string) => void;
    onDeleteWorkspace: () => void;
}
const WorkspaceEditorComponent = ({
    title,
    icon,
    color,
    focusInput,
    workspaceId,
    onTitleChange,
    onColorChange,
    onIconChange,
    onDeleteWorkspace,
}: WorkspaceEditorProps) => {
    const inputRef = useRef<HTMLInputElement>(null);
    const [colors, setColors] = useState<string[]>([]);
    const [icons, setIcons] = useState<string[]>([]);

    useEffect(() => {
        fireAndForget(async () => {
            const colors = await WorkspaceService.GetColors();
            const icons = await WorkspaceService.GetIcons();
            setColors(colors);
            setIcons(icons);
        });
    }, []);

    useEffect(() => {
        if (focusInput && inputRef.current) {
            inputRef.current.focus();
            inputRef.current.select();
        }
    }, [focusInput]);

    return (
        <div className="workspace-editor">
            <Input
                ref={inputRef}
                className={clsx("py-[3px]", { error: title === "" })}
                onChange={onTitleChange}
                value={title}
                autoFocus
                autoSelect
            />
            <ColorSelector selectedColor={color} colors={colors} onSelect={onColorChange} />
            <IconSelector selectedIcon={icon} icons={icons} onSelect={onIconChange} />
            <MCProjectSelector workspaceId={workspaceId} />
            <div className="delete-ws-btn-wrapper">
                <Button className="ghost red text-[12px] bold" onClick={onDeleteWorkspace}>
                    Delete workspace
                </Button>
            </div>
        </div>
    );
};

export const WorkspaceEditor = memo(WorkspaceEditorComponent) as typeof WorkspaceEditorComponent;
