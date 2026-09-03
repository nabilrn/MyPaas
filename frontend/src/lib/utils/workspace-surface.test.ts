import { describe, expect, it } from 'vitest';
import appTemplate from '../../app.html?raw';
import layout from '../../routes/+layout.svelte?raw';
import themeStore from '../stores/theme.ts?raw';

const dashboardRoutes = import.meta.glob('../../routes/**/*.svelte', {
	eager: true,
	query: '?raw',
	import: 'default'
}) as Record<string, string>;

describe('authenticated workspace surface contract', () => {
	it('uses one shared surface for shell chrome and operational sections', () => {
		expect(layout).toContain('class="app-shell min-h-screen lg:pl-14"');
		expect(layout).toContain(':global(.app-workspace .workspace-section)');
		expect(layout).toContain(':global(.app-workspace .surface-muted)');
		expect(layout).toContain('background: var(--app-surface) !important;');
	});

	it('normalizes legacy neutral section fills while preserving structure as strokes', () => {
		expect(layout).toContain("[class~='bg-white']");
		expect(layout).toContain("[class~='dark:bg-neutral-900']");
		expect(layout).toContain("[class~='dark:bg-neutral-950']");
		expect(layout).toContain(':global(.app-workspace .workspace-section .grid.gap-px)');
		expect(layout).toContain('background-color: var(--workspace-divider) !important;');
		expect(layout).toContain('border-bottom: 1px solid var(--app-border) !important;');
	});

	it('keeps tables on the same surface in idle, hover, and selected states', () => {
		expect(layout).toContain(':global(.app-workspace .data-table thead th)');
		expect(layout).toContain(':global(.app-workspace .data-table tbody tr:hover)');
		expect(layout).toContain("tr[aria-selected='true']");
		expect(layout).toContain('box-shadow: inset 2px 0 0 var(--app-border-strong);');
	});

	it('uses Host Shell colors as the canonical dashboard technical-output surface', () => {
		expect(layout).toContain('--technical-surface-bg: #171717;');
		expect(layout).toContain('--technical-surface-bg: #0a0a0a;');
		expect(layout).toContain(':global(.app-workspace .console-surface)');
		expect(layout).toContain(':global(.app-workspace .code-surface)');
		expect(layout).toContain(':global(.app-workspace pre)');
	});

	it('does not introduce an alternate slate/zinc/stone surface palette in dashboard routes', () => {
		for (const [path, source] of Object.entries(dashboardRoutes)) {
			if (path.endsWith('/login/+page.svelte')) continue;
			expect(source, path).not.toMatch(/(?:dark:)?bg-(?:slate|zinc|stone)-/);
		}
	});
});

describe('theme paint contract', () => {
	it('applies the stored/system theme in app.html before Svelte head and hydration', () => {
		const themeScript = appTemplate.indexOf("localStorage.getItem('theme')");
		const svelteHead = appTemplate.indexOf('%sveltekit.head%');
		expect(themeScript).toBeGreaterThan(-1);
		expect(themeScript).toBeLessThan(svelteHead);
		expect(appTemplate).toContain("document.documentElement.classList.toggle('dark', dark)");
		expect(appTemplate).toContain("document.documentElement.style.colorScheme = dark ? 'dark' : 'light'");
	});

	it('keeps the hydrated theme store aligned with the prepaint state', () => {
		expect(themeStore).toContain("document.documentElement.classList.toggle('dark', dark)");
		expect(themeStore).toContain("document.documentElement.style.colorScheme = dark ? 'dark' : 'light'");
		expect(themeStore).toContain('apply(initial, false)');
	});

	it('does not blank the workspace merely because client-side navigation is in progress', () => {
		expect(layout).not.toContain("import { navigating, page } from '$app/stores'");
		expect(layout).toContain('$: showMainLoader = !checked || $mainContentLoading;');
	});
});
