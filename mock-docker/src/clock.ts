export interface Clock {
    now(): Date;
}

export class RealClock implements Clock {
    now(): Date {
        return new Date();
    }
}

export class FixedClock implements Clock {
    private time: Date;

    constructor(time: Date) {
        this.time = new Date(time.getTime());
    }

    now(): Date {
        return new Date(this.time.getTime());
    }

    advance(ms: number): void {
        this.time = new Date(this.time.getTime() + ms);
    }
}

export function createClock(opts: { fixed?: boolean; base?: string } = {}): Clock {
    if (opts.fixed) {
        const base = opts.base ? new Date(opts.base) : new Date("2025-01-01T00:00:00Z");
        return new FixedClock(base);
    }
    return new RealClock();
}
