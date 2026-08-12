import { describe, expect, it } from 'vitest';

import { runtimeServiceSummary, selectPrimaryProjectMetric } from './project-dashboard';

const metrics = {
	collectedAt: '2026-08-12T00:00:00Z',
	items: [
		{ service: 'worker', cpu: 8, memoryMb: 64, memoryLimitMb: 128, uptime: '1h' },
		{ service: 'api', cpu: 12, memoryMb: 96, memoryLimitMb: 256, uptime: '2h' }
	]
};

describe('project dashboard metrics', () => {
	it('prefers the configured main service for Compose metrics', () => {
		expect(selectPrimaryProjectMetric(metrics, 'api')?.service).toBe('api');
	});

	it('falls back to the first runtime metric when no main service matches', () => {
		expect(selectPrimaryProjectMetric(metrics, 'web')?.service).toBe('worker');
	});

	it('reports other services without treating them as the primary project metric', () => {
		const primary = selectPrimaryProjectMetric(metrics, 'api');
		expect(runtimeServiceSummary(metrics, primary)).toEqual({
			label: 'api',
			otherServices: 1
		});
	});

	it('handles an empty metrics snapshot', () => {
		expect(selectPrimaryProjectMetric({ collectedAt: '2026-08-12T00:00:00Z', items: [] }, 'api')).toBeNull();
		expect(runtimeServiceSummary(null, null)).toEqual({ label: 'No live service', otherServices: 0 });
	});
});
