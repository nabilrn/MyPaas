import { browser } from '$app/environment';
import { page } from '$app/stores';
import { derived, get, writable } from 'svelte/store';

const activeLoads = writable(0);
let activeRoute = '';
let activeGeneration = 0;
let settledGeneration = -1;
const pendingByGeneration = new Map<number, number>();

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

/**
 * Gate GET requests that belong to the first resource load for an authenticated route.
 *
 * Every route/query transition receives its own generation. Requests from a previous
 * generation can finish later without changing the pending count or settled state of
 * the current page. Release is deferred one task so page state commits before the
 * main-content loader disappears. Polling, refreshes, inspections, and mutations stay
 * non-blocking and use their own local progress states.
 */
export function beginInitialRouteRequestLoading(method = 'GET') {
	if (!browser || method.toUpperCase() !== 'GET') return null;

	const currentPage = get(page);
	const pathname = currentPage.url.pathname;
	if (pathname === '/' || pathname === '/login' || pathname.startsWith('/docs')) return null;
	const routeKey = `${pathname}${currentPage.url.search}`;

	if (routeKey !== activeRoute) {
		activeRoute = routeKey;
		activeGeneration += 1;
		settledGeneration = -1;
		pendingByGeneration.set(activeGeneration, 0);
	}

	const generation = activeGeneration;
	if (settledGeneration === generation) return null;

	pendingByGeneration.set(generation, (pendingByGeneration.get(generation) ?? 0) + 1);
	const finishMainLoading = beginMainContentLoading();
	let finished = false;

	return () => {
		if (finished) return;
		finished = true;
		pendingByGeneration.set(generation, Math.max(0, (pendingByGeneration.get(generation) ?? 1) - 1));

		window.setTimeout(() => {
			finishMainLoading();
			const pending = pendingByGeneration.get(generation) ?? 0;
			if (generation === activeGeneration && pending === 0) settledGeneration = generation;
			if (generation !== activeGeneration && pending === 0) pendingByGeneration.delete(generation);
		}, 0);
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
