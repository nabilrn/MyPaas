import { describe, expect, it } from 'vitest';

import { canRollbackDeployment, deploymentHistoryLabel, isCurrentDeployment, isPipelineActive } from './deploymentHistory';

describe('deployment history state', () => {
	it('marks only pipeline statuses as active work', () => {
		for (const status of ['queued', 'cloning', 'building', 'starting'] as const) {
			expect(isPipelineActive(status)).toBe(true);
		}
		expect(isPipelineActive('running')).toBe(false);
		expect(isPipelineActive('failed')).toBe(false);
	});

	it('identifies the project active deployment by id', () => {
		expect(isCurrentDeployment('dep-1', 'dep-1')).toBe(true);
		expect(isCurrentDeployment('dep-1', 'dep-2')).toBe(false);
		expect(isCurrentDeployment('dep-1', null)).toBe(false);
	});

	it('shows the serving release as Active and older successful rows as Succeeded', () => {
		expect(deploymentHistoryLabel('running', 'dep-1', 'dep-1', 'running')).toBe('Active');
		expect(deploymentHistoryLabel('running', 'dep-2', 'dep-1', 'running')).toBe('Succeeded');
	});

	it('shows Current when the project runtime is stopped but the release remains selected', () => {
		expect(deploymentHistoryLabel('running', 'dep-1', 'dep-1', 'stopped')).toBe('Current');
	});

	it('does not override non-success terminal or pipeline labels', () => {
		expect(deploymentHistoryLabel('failed', 'dep-1', 'dep-1', 'running')).toBeUndefined();
		expect(deploymentHistoryLabel('building', 'dep-1', 'dep-1', 'building')).toBeUndefined();
	});

	it('allows rollback only to a successful deployment that is not already current', () => {
		expect(canRollbackDeployment('running', 'dep-old', 'dep-current')).toBe(true);
		expect(canRollbackDeployment('running', 'dep-current', 'dep-current')).toBe(false);
		expect(canRollbackDeployment('stopped', 'dep-old', 'dep-current')).toBe(false);
		expect(canRollbackDeployment('failed', 'dep-old', 'dep-current')).toBe(false);
	});
});
