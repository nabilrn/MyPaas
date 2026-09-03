import { describe, expect, it } from 'vitest';
import { mergeRuntimeContainerTelemetry, type RuntimeContainer } from './container-inventory';

function container(overrides: Partial<RuntimeContainer> = {}): RuntimeContainer {
	return {
		id: 'container-a',
		name: 'app-a',
		image: 'example/app:latest',
		state: 'running',
		status: 'Up 1 minute',
		composeProject: '',
		service: '',
		cpu: 0,
		memoryMb: 0,
		memoryLimitMb: 0,
		metricsAvailable: false,
		...overrides
	};
}

describe('mergeRuntimeContainerTelemetry', () => {
	it('updates metrics by stable container ID without replacing metadata', () => {
		const rows = [container({ image: 'example/app:v1' })];
		const telemetry = [container({ cpu: 2.5, memoryMb: 64, memoryLimitMb: 512, metricsAvailable: true })];

		expect(mergeRuntimeContainerTelemetry(rows, telemetry)).toEqual([
			container({ image: 'example/app:v1', cpu: 2.5, memoryMb: 64, memoryLimitMb: 512, metricsAvailable: true })
		]);
	});

	it('keeps metrics unavailable when telemetry has no matching row', () => {
		const rows = [container({ cpu: 4, memoryMb: 128, memoryLimitMb: 512, metricsAvailable: true })];

		expect(mergeRuntimeContainerTelemetry(rows, [container({ id: 'other' })])).toEqual([
		container({ cpu: 0, memoryMb: 0, memoryLimitMb: 0, metricsAvailable: false })
		]);
	});
});
