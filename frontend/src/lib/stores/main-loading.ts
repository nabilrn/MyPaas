import { derived, writable } from 'svelte/store';

const activeLoads = writable(0);

export const mainContentLoading = derived(activeLoads, (count) => count > 0);

export function beginMainContentLoading() {
	let finished = false;
	activeLoads.update((count) => count + 1);

	return () => {
		if (finished) return;
		finished = true;
		activeLoads.update((count) => Math.max(0, count - 1));
	};
}

export async function withMainContentLoading<T>(task: () => Promise<T>): Promise<T> {
	const finish = beginMainContentLoading();
	try {
		return await task();
	} finally {
		finish();
	}
}
