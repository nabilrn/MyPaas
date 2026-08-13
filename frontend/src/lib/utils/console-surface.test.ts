import { describe, expect, it } from 'vitest';
import logsPage from '../../routes/projects/[id]/logs/+page.svelte?raw';

const logViewportMarkup =
	logsPage.match(/<div\b[^>]*bind:this=\{logViewport\}[^>]*>/)?.[0] ?? '';

describe('project log console interaction contract', () => {
	it('forces the log viewport to remain scrollable inside the shared console surface', () => {
		expect(logViewportMarkup).toContain('!overflow-auto');
	});

	it('keeps project logs text-selectable with visible selection feedback', () => {
		expect(logViewportMarkup).toContain('select-text');
		expect(logViewportMarkup).toContain('selection:bg-gray-700');
		expect(logViewportMarkup).toContain('selection:text-white');
	});
});
