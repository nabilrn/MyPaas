import { browser } from '$app/environment';
import { page } from '$app/stores';
import { derived, get, writable } from 'svelte/store';

const activeLoads = writable(0);
let activeRoute = '';
let settledRoute = '';
let pendingInitialRequests = 0;

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
 * Gate API requests that belong to the first resource load for an authenticated route.
 *
 * A route stays unsettled while its initial parallel/sequential request chain is active.
 * Release is deferred one task so page state can commit before the main-content loader
 * disappears. Later polling, refreshes, and mutations on the settled route stay non-blocking.
 */
export function beginInitialRouteRequestLoading() {
	if (!browser) return null;

	const pathname = get(page).url.pathname;
	if (pathname === '/' || pathname === '/login' || pathname.startsWith('/docs')) return null;

	if (pathname !== activeRoute) {
		activeRoute = pathname;
		settledRoute = '';
		pendingInitialRequests = 0;
	}
	if (settledRoute === pathname) return null;

	pendingInitialRequests += 1;
	const finishMainLoading = beginMainContentLoading();
	let finished = false;

	return () => {
		if (finished) return;
		finished = true;
		pendingInitialRequests = Math.max(0, pendingInitialRequests - 1);

		window.setTimeout(() => {
			finishMainLoading();
			if (activeRoute === pathname && pendingInitialRequests === 0) settledRoute = pathname;
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
