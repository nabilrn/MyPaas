import { describe, expect, it } from 'vitest';
import layout from '../../routes/+layout.svelte?raw';

describe('authenticated workspace surface contract', () => {
	it('uses one shared surface for shell chrome and main workspace', () => {
		expect(layout).toContain('class="app-shell min-h-screen lg:pl-14"');
		expect(layout).toContain(':global(.app-shell > aside)');
		expect(layout).toContain(':global(.app-shell > header)');
		expect(layout).toContain(':global(.app-workspace)');
		expect(layout).toContain('background: var(--app-surface) !important;');
	});

	it('separates workspace sections with borders instead of alternate section fills', () => {
		expect(layout).toContain('border-bottom: 1px solid var(--app-border) !important;');
		expect(layout).toContain(':global(.app-workspace .panel-header)');
		expect(layout).toContain('border-color: var(--app-border) !important;');
	});
});
