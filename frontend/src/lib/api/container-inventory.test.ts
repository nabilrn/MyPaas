import { describe, expect, it } from 'vitest';
import containerInventoryApi from './container-inventory.ts?raw';
import containersPage from '../../routes/containers/+page.svelte?raw';

describe('container inventory product contract', () => {
	it('keeps the host inventory metadata-only', () => {
		expect(containerInventoryApi).toContain('/api/admin/containers?telemetry=false');
		expect(containerInventoryApi).not.toContain('mergeRuntimeContainerTelemetry');
		expect(containerInventoryApi).not.toContain('cpuAvailable');
		expect(containerInventoryApi).not.toContain('memoryAvailable');
	});

	it('shows operational metadata without CPU/RAM telemetry UI', () => {
		expect(containersPage).toContain('Restarts');
		expect(containersPage).toContain('Ports');
		expect(containersPage).toContain('Uptime');
		expect(containersPage).toContain('Network details unavailable.');
		expect(containersPage).not.toContain('CPU/RAM telemetry');
		expect(containersPage).not.toContain('loadTelemetry');
		expect(containersPage).not.toContain('telemetryPoll');
		expect(containersPage).not.toContain('>CPU<');
		expect(containersPage).not.toContain('>Memory<');
	});
});
