import { writable } from 'svelte/store';
import { browser } from '$app/environment';

type Theme = 'light' | 'dark';

function resolveInitial(): Theme {
	if (!browser) return 'light';
	const stored = localStorage.getItem('theme');
	if (stored === 'dark' || stored === 'light') return stored;
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function createThemeStore() {
	const { subscribe, set, update } = writable<Theme>(resolveInitial());

	function apply(theme: Theme) {
		if (!browser) return;
		document.documentElement.classList.toggle('dark', theme === 'dark');
		localStorage.setItem('theme', theme);
	}

	// Apply immediately on init
	apply(resolveInitial());

	return {
		subscribe,
		toggle() {
			update((t) => {
				const next = t === 'light' ? 'dark' : 'light';
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
