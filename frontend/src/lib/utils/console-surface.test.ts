import { describe, expect, it } from 'vitest';
import appCss from '../../app.css?raw';
import logsPage from '../../routes/projects/[id]/logs/+page.svelte?raw';

function ruleBody(css: string, selector: string): string {
	const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	const match = css.match(new RegExp(`${escaped}\\s*\\{([^}]*)\\}`));
	return match?.[1] ?? '';
}

describe('console surface interaction contract', () => {
	it('leaves overflow behavior to each console consumer', () => {
		expect(ruleBody(appCss, '.console-surface')).not.toContain('overflow-hidden');
	});

	it('keeps project logs text-selectable with visible selection feedback', () => {
		expect(logsPage).toContain('select-text');
		expect(logsPage).toContain('selection:bg-gray-700');
		expect(logsPage).toContain('selection:text-white');
	});
});
