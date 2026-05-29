// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import * as electron from "electron";
import * as child_process from "node:child_process";
import * as crypto from "node:crypto";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import * as readline from "readline";
import { getElectronAppUnpackedBasePath, unameArch } from "./emain-platform";

export const McApiUrlEnvName = "MC_API_URL";
export const McAuthKeyEnvName = "MC_AUTH_KEY";

// Generated once at startup, injected into mcsrv and wavesrv envs.
export const McAuthKey: string = crypto.randomBytes(32).toString("hex");

let isMcSrvDead = false;
let mcSrvProc: child_process.ChildProcessWithoutNullStreams | null = null;

let mcSrvReadyResolve = (value: boolean) => {};
const mcSrvReady: Promise<boolean> = new Promise((resolve) => {
    mcSrvReadyResolve = resolve;
});

export function getMcSrvReady(): Promise<boolean> {
    return mcSrvReady;
}

export function getMcSrvProc(): child_process.ChildProcessWithoutNullStreams | null {
    return mcSrvProc;
}

export function getIsMcSrvDead(): boolean {
    return isMcSrvDead;
}

function getMcSrvPath(): string {
    const binName = process.platform === "win32" ? `mcsrv.${unameArch}.exe` : `mcsrv.${unameArch}`;
    return path.join(getElectronAppUnpackedBasePath(), "bin", binName);
}

function getMcDataHome(): string {
    const override = process.env["MC_DATA_HOME"];
    if (override) return override;
    return path.join(os.homedir(), ".mc");
}

export function runMcSrv(): Promise<boolean> {
    let pResolve: (value: boolean) => void;
    let pReject: (reason?: any) => void;
    const rtnPromise = new Promise<boolean>((argResolve, argReject) => {
        pResolve = argResolve;
        pReject = argReject;
    });

    const dataHome = getMcDataHome();
    try {
        fs.mkdirSync(dataHome, { recursive: true });
    } catch (_) {}

    const mcSrvCmd = getMcSrvPath();
    if (!fs.existsSync(mcSrvCmd)) {
        const err = new Error(`mcsrv binary not found: ${mcSrvCmd}`);
        console.log(err.message);
        return Promise.reject(err);
    }

    const envCopy = { ...process.env };
    envCopy[McAuthKeyEnvName] = McAuthKey;
    envCopy["MC_DATA_HOME"] = dataHome;

    console.log("trying to run mcsrv", mcSrvCmd);

    const proc = child_process.spawn(mcSrvCmd, {
        cwd: dataHome,
        env: envCopy,
    });

    proc.on("exit", (code) => {
        console.log("mcsrv exited with code", code, "— MC features unavailable");
        isMcSrvDead = true;
        mcSrvProc = null;
        // mcsrv dying is not fatal for Wave — don't call electronApp.quit()
        // Broadcast to any open windows that MC backend is down
        for (const win of electron.BrowserWindow.getAllWindows()) {
            if (!win.isDestroyed()) {
                win.webContents.send("mc-backend-status", { alive: false });
            }
        }
    });

    proc.on("spawn", () => {
        console.log("spawned mcsrv");
        mcSrvProc = proc;
        pResolve(true);
    });

    proc.on("error", (e) => {
        console.log("error running mcsrv", e);
        isMcSrvDead = true;
        pReject(e);
    });

    const rlStdout = readline.createInterface({ input: proc.stdout, terminal: false });
    rlStdout.on("line", (line) => {
        console.log("[mcsrv]", line);
    });

    const rlStderr = readline.createInterface({ input: proc.stderr, terminal: false });
    rlStderr.on("line", (line) => {
        if (line.includes("MCSRV-ESTART")) {
            // Format: "MCSRV-ESTART api:127.0.0.1:3001"
            const match = /MCSRV-ESTART api:([a-z0-9.:]+)/m.exec(line);
            if (match == null) {
                console.log("error parsing MCSRV-ESTART line", line);
                return;
            }
            const apiAddr = match[1];
            process.env[McApiUrlEnvName] = "http://" + apiAddr;
            console.log("mcsrv ready, MC_API_URL =", process.env[McApiUrlEnvName]);
            mcSrvReadyResolve(true);
            return;
        }
        console.log("[mcsrv]", line);
    });

    return rtnPromise;
}
