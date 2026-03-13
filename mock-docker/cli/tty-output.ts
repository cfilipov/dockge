// ANSI escape sequences matching Docker Compose v2
const ANSI_GREEN = "\x1b[32m";
const ANSI_RESET = "\x1b[0m";
const ANSI_HIDE_CURSOR = "\x1b[?25l";
const ANSI_SHOW_CURSOR = "\x1b[?25h";
const ANSI_CURSOR_UP = "\x1b[A";
const ANSI_ERASE_LINE = "\x1b[2K";

const SPINNER_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];

export interface ProgressTask {
    name: string;
    action: string;
    done: string;
}

function sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

function writeHeader(verb: string, completed: number, total: number): void {
    process.stdout.write(
        `\r${ANSI_ERASE_LINE} ${ANSI_GREEN}[+]${ANSI_RESET} ${verb} ${completed}/${total}\r\n`,
    );
}

function writeTaskPending(task: ProgressTask, spinIdx: number): void {
    const frame = SPINNER_FRAMES[spinIdx % SPINNER_FRAMES.length];
    process.stdout.write(
        `\r${ANSI_ERASE_LINE} ${frame} ${task.name}  ${task.action}\r\n`,
    );
}

function writeTaskDone(task: ProgressTask, elapsedSec: number): void {
    process.stdout.write(
        `\r${ANSI_ERASE_LINE} ${ANSI_GREEN}✔${ANSI_RESET} ${task.name}  ${task.done.padEnd(12)}${elapsedSec.toFixed(1)}s\r\n`,
    );
}

function moveCursorUp(lines: number): void {
    for (let i = 0; i < lines; i++) {
        process.stdout.write(ANSI_CURSOR_UP);
    }
}

/**
 * Render Docker Compose v2-style animated progress.
 * In TTY mode: animated with spinners. In non-TTY mode: sequential output.
 */
export async function renderProgress(
    verb: string,
    tasks: ProgressTask[],
): Promise<void> {
    const n = tasks.length;
    if (n === 0) return;

    if (!process.stdout.isTTY) {
        // Non-TTY: simple sequential output
        for (const task of tasks) {
            process.stdout.write(` ${ANSI_GREEN}✔${ANSI_RESET} ${task.name}  ${task.done}\n`);
        }
        return;
    }

    const FRAMES_PER_TASK = 3;
    const DELAY = 50;

    const taskStart: number[] = new Array(n);
    const taskElapsed: number[] = new Array(n);

    process.stdout.write(ANSI_HIDE_CURSOR);

    // Draw initial frame
    writeHeader(verb, 0, n);
    let spinIdx = 0;
    for (const task of tasks) {
        writeTaskPending(task, spinIdx);
    }

    // Animate: complete tasks one at a time
    for (let completed = 0; completed < n; completed++) {
        taskStart[completed] = performance.now();

        for (let frame = 0; frame < FRAMES_PER_TASK; frame++) {
            await sleep(DELAY);
            spinIdx++;
            moveCursorUp(n + 1);
            writeHeader(verb, completed, n);
            for (let i = 0; i < n; i++) {
                if (i < completed) {
                    writeTaskDone(tasks[i], taskElapsed[i]);
                } else {
                    writeTaskPending(tasks[i], spinIdx);
                }
            }
        }

        taskElapsed[completed] = (performance.now() - taskStart[completed]) / 1000;
        await sleep(DELAY);
        moveCursorUp(n + 1);
        writeHeader(verb, completed + 1, n);
        for (let i = 0; i < n; i++) {
            if (i <= completed) {
                writeTaskDone(tasks[i], taskElapsed[i]);
            } else {
                writeTaskPending(tasks[i], spinIdx);
            }
        }
    }

    process.stdout.write(ANSI_SHOW_CURSOR);
}

/**
 * Build progress tasks for compose up.
 */
export function composeUpTasks(
    project: string,
    services: string[],
    isWholeStack: boolean,
    forceRecreate: boolean,
): ProgressTask[] {
    const tasks: ProgressTask[] = [];
    if (isWholeStack) {
        tasks.push({
            name: `Network ${project}_default`,
            action: "Creating",
            done: "Created",
        });
    }
    for (const svc of services) {
        if (forceRecreate) {
            tasks.push(
                { name: `Container ${project}-${svc}-1`, action: "Recreating", done: "Recreated" },
                { name: `Container ${project}-${svc}-1`, action: "Starting", done: "Started" },
            );
        } else {
            tasks.push(
                { name: `Container ${project}-${svc}-1`, action: "Creating", done: "Created" },
                { name: `Container ${project}-${svc}-1`, action: "Starting", done: "Started" },
            );
        }
    }
    return tasks;
}

/**
 * Build progress tasks for compose stop.
 */
export function composeStopTasks(
    project: string,
    services: string[],
): ProgressTask[] {
    return services.map((svc) => ({
        name: `Container ${project}-${svc}-1`,
        action: "Stopping",
        done: "Stopped",
    }));
}

/**
 * Build progress tasks for compose down.
 */
export function composeDownTasks(
    project: string,
    services: string[],
    removeVolumes: boolean,
): ProgressTask[] {
    const tasks: ProgressTask[] = [];
    for (const svc of services) {
        tasks.push(
            { name: `Container ${project}-${svc}-1`, action: "Stopping", done: "Stopped" },
            { name: `Container ${project}-${svc}-1`, action: "Removing", done: "Removed" },
        );
    }
    if (removeVolumes) {
        for (const svc of services) {
            tasks.push({
                name: `Volume ${project}_${svc}-data`,
                action: "Removing",
                done: "Removed",
            });
        }
    }
    tasks.push({
        name: `Network ${project}_default`,
        action: "Removing",
        done: "Removed",
    });
    return tasks;
}

/**
 * Build progress tasks for compose restart.
 */
export function composeRestartTasks(
    project: string,
    services: string[],
): ProgressTask[] {
    return services.map((svc) => ({
        name: `Container ${project}-${svc}-1`,
        action: "Restarting",
        done: "Started",
    }));
}

/**
 * Build progress tasks for compose pull.
 */
export function composePullTasks(services: string[]): ProgressTask[] {
    return services.map((svc) => ({
        name: svc,
        action: "Pulling",
        done: "Pulled",
    }));
}

/**
 * Build progress tasks for compose pause.
 */
export function composePauseTasks(
    project: string,
    services: string[],
): ProgressTask[] {
    return services.map((svc) => ({
        name: `Container ${project}-${svc}-1`,
        action: "Pausing",
        done: "Paused",
    }));
}

/**
 * Build progress tasks for compose unpause.
 */
export function composeUnpauseTasks(
    project: string,
    services: string[],
): ProgressTask[] {
    return services.map((svc) => ({
        name: `Container ${project}-${svc}-1`,
        action: "Unpausing",
        done: "Unpaused",
    }));
}
