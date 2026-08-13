import { writable } from 'svelte/store';
import type { LogLine, MetricsSnapshot } from '$types';

export type ProjectStreamConnection = 'idle' | 'connecting' | 'open' | 'reconnecting';

export const projectStreamMetrics = writable<MetricsSnapshot | null>(null);
export const projectStreamConnection = writable<ProjectStreamConnection>('idle');
export const projectStreamLogs = writable<LogLine[]>([]);

let reconnectHandler: (() => void) | null = null;

export function setProjectStreamReconnect(handler: (() => void) | null) {
	reconnectHandler = handler;
}

export function reconnectProjectStream() {
	reconnectHandler?.();
}

export function resetProjectStreamState() {
	projectStreamMetrics.set(null);
	projectStreamConnection.set('idle');
	projectStreamLogs.set([]);
}

export function appendProjectStreamLog(item: LogLine) {
	projectStreamLogs.update((items) => [...items.slice(-499), item]);
}
