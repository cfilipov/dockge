import type { DockerEvent } from "./list-types.js";
import type { MockState } from "./state.js";
import type { Clock } from "./clock.js";

export type EventListener = (event: DockerEvent) => void;

export class EventEmitter {
    private listeners = new Set<EventListener>();
    private state: MockState | null = null;

    /** Bind to a MockState so emit() also records to eventHistory. */
    bind(state: MockState): void {
        this.state = state;
    }

    subscribe(listener: EventListener): void {
        this.listeners.add(listener);
    }

    unsubscribe(listener: EventListener): void {
        this.listeners.delete(listener);
    }

    emit(event: DockerEvent): void {
        // Record to event history for /events?since= queries
        if (this.state) {
            this.state.eventHistory.push(event);
        }
        for (const listener of this.listeners) {
            listener(event);
        }
    }
}

/**
 * Build a DockerEvent with time/timeNano from the clock.
 */
export function makeEvent(
    clock: Clock,
    type: string,
    action: string,
    id: string,
    attributes: Record<string, string> = {},
): DockerEvent {
    return makeEventAt(clock.now(), type, action, id, attributes);
}

/**
 * Build a DockerEvent with an explicit timestamp (no clock tick).
 * Use this when the event time must align with a known timestamp
 * (e.g. container StartedAt) rather than the clock's tick counter.
 */
export function makeEventAt(
    time: Date,
    type: string,
    action: string,
    id: string,
    attributes: Record<string, string> = {},
): DockerEvent {
    const epochMs = time.getTime();
    const epochSec = Math.floor(epochMs / 1000);
    const epochNano = epochMs * 1_000_000; // ms → ns

    return {
        Type: type,
        Action: action,
        Actor: { ID: id, Attributes: attributes },
        time: epochSec,
        timeNano: epochNano,
    };
}
