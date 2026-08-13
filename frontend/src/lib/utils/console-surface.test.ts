import { describe, expect, it } from 'vitest';
import logsPage from '../../routes/projects/[id]/logs/+page.svelte?raw';

describe('project log console interaction contract', () => {
	it('forces the log viewport to remain scrollable inside the shared console surface', () => {
		expect(logsPage).toContain('!overflow-auto');
	});

	it('keeps project logs text-selectable with visible selection feedback', () => {
		expect(logsPage).toContain('select-text');
		expect(logsPage).toContain('selection:bg-gray-700');
		expect(logsPage).toContain('selection:text-white');
	});
});
