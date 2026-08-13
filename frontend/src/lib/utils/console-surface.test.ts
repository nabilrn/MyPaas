import { describe, expect, it } from 'vitest';
import appCss from '../../app.css?raw';
import logsPage from '../../routes/projects/[id]/logs/+page.svelte?raw';

describe('console surface interaction contract', () => {
	it('leaves overflow behavior to each console consumer', () => {
		const afterSelector = appCss.split('.console-surface {')[1] ?? '';
		const consoleRule = afterSelector.split('}')[0] ?? '';
		expect(consoleRule.length).toBeGreaterThan(0);
		expect(consoleRule).not.toContain('overflow-hidden');
	});

	it('keeps project logs text-selectable with visible selection feedback', () => {
		expect(logsPage).toContain('select-text');
		expect(logsPage).toContain('selection:bg-gray-700');
		expect(logsPage).toContain('selection:text-white');
	});
});
