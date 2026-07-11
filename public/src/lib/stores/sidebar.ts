import { browser } from '$app/environment';
import { writable } from 'svelte/store';

const storageKey = 'mypaas-sidebar-collapsed';

function initialValue() {
	if (!browser) return false;
	return localStorage.getItem(storageKey) === 'true';
}

function createSidebarStore() {
	const { subscribe, set, update } = writable(initialValue());

	function persist(value: boolean) {
		if (!browser) return;
		localStorage.setItem(storageKey, String(value));
	}

	return {
		subscribe,
		set(value: boolean) {
			persist(value);
			set(value);
		},
		toggle() {
			update((value) => {
				const next = !value;
				persist(next);
				return next;
			});
		}
	};
}

export const sidebarCollapsed = createSidebarStore();
