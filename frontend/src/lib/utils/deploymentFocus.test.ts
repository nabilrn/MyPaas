import { describe, expect, it } from 'vitest';
import { expandFocusedDeployment, normalizeDeploymentFocus, pinFocusedDeployment } from './deploymentFocus';

describe('deployment focus state', () => {
	it('normalizes a focus query value', () => {
		expect(normalizeDeploymentFocus('  deployment-42  ')).toBe('deployment-42');
		expect(normalizeDeploymentFocus(null)).toBe('');
	});

	it('auto-expands the focused deployment without dropping manual expansion', () => {
		const expanded = expandFocusedDeployment(new Set(['deployment-1']), 'deployment-42');

		expect([...expanded]).toEqual(['deployment-1', 'deployment-42']);
	});

	it('pins an off-page focused deployment exactly once', () => {
		const rows = [{ id: 'deployment-1' }, { id: 'deployment-2' }];
		const pinned = pinFocusedDeployment(rows, { id: 'deployment-42' });
		const alreadyVisible = pinFocusedDeployment(rows, rows[1]);

		expect(pinned.map((row) => row.id)).toEqual(['deployment-42', 'deployment-1', 'deployment-2']);
		expect(alreadyVisible.map((row) => row.id)).toEqual(['deployment-2', 'deployment-1']);
	});
});
