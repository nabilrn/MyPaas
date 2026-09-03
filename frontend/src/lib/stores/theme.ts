import { writable } from 'svelte/store';
import { browser } from '$app/environment';

type Theme = 'light' | 'dark';

function resolveInitial(): Theme {
	if (!browser) return 'light';
	try {
		const stored = localStorage.getItem('theme');
		if (stored === 'dark' || stored === 'light') return stored;
	} catch {
		// Storage may be unavailable by browser policy.
	}
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function createThemeStore() {
	const initial = resolveInitial();
	const { subscribe, set, update } = writable<Theme>(initial);

	function apply(theme: Theme, persist = true) {
		if (!browser) return;
		const dark = theme === 'dark';
		document.documentElement.classList.toggle('dark', dark);
		document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
		if (!persist) return;
		try {
			localStorage.setItem('theme', theme);
		} catch {
			// Keep the in-memory preference even when storage is unavailable.
		}
	}

	// Reconcile with the blocking app.html prepaint without rewriting persistence.
	apply(initial, false);

	return {
		subscribe,
		toggle() {
			update((theme) => {
				const next = theme === 'light' ? 'dark' : 'light';
				apply(next);
				return next;
			});
		},
		set(theme: Theme) {
			apply(theme);
			set(theme);
		}
	};
}

export const theme = createThemeStore();
