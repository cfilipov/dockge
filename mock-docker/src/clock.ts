export interface Clock {
    now(): Date;
}

export class RealClock implements Clock {
    now(): Date {
        return new Date();
    }
}

export class FixedClock implements Clock {
    private base: number;
    private tick = 0;
    private static readonly TICK_MS = 100;

    constructor(time: Date) {
        this.base = time.getTime();
    }

    now(): Date {
        return new Date(this.base + (this.tick++) * FixedClock.TICK_MS);
    }

    advance(ms: number): void {
        this.base += ms;
    }

    /** Reset the tick counter (e.g. for mock reset). */
    resetTick(): void {
        this.tick = 0;
    }
}

export function createClock(opts: { fixed?: boolean; base?: string } = {}): Clock {
    if (opts.fixed) {
        const base = opts.base ? new Date(opts.base) : new Date("2025-01-01T00:00:00Z");
        return new FixedClock(base);
    }
    return new RealClock();
}
