import { writable } from 'svelte/store';

export type ToastKind = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id:      string;
	kind:    ToastKind;
	message: string;
}

function createToastStore() {
	const { subscribe, update } = writable<Toast[]>([]);

	function add(kind: ToastKind, message: string, durationMs = 4000) {
		const id = crypto.randomUUID();
		update((list) => [...list, { id, kind, message }]);
		setTimeout(() => remove(id), durationMs);
	}

	function remove(id: string) {
		update((list) => list.filter((t) => t.id !== id));
	}

	return {
		subscribe,
		success: (msg: string) => add('success', msg),
		error:   (msg: string) => add('error',   msg, 6000),
		warning: (msg: string) => add('warning', msg),
		info:    (msg: string) => add('info',    msg),
		remove
	};
}

export const toast = createToastStore();
