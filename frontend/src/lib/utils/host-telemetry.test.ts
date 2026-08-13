import { describe, expect, it } from 'vitest';
import { appendRollingSample, boundedPercent, deriveCPUUsage, deriveNetworkRate } from './host-telemetry';

describe('host telemetry helpers', () => {
	it('bounds resource percentages', () => {
		expect(boundedPercent(25, 100)).toBe(25);
		expect(boundedPercent(150, 100)).toBe(100);
		expect(boundedPercent(-10, 100)).toBe(0);
		expect(boundedPercent(1, 0)).toBe(0);
	});

	it('keeps only the newest rolling samples', () => {
		expect(appendRollingSample([1, 2, 3], 4, 3)).toEqual([2, 3, 4]);
	});

	it('derives cpu usage from cumulative total and idle counters', () => {
		expect(deriveCPUUsage(
			{ totalTicks: 1_000, idleTicks: 700 },
			{ totalTicks: 1_200, idleTicks: 760 }
		)).toBe(70);
	});

	it('resets the cpu baseline on counter reset or invalid delta', () => {
		expect(deriveCPUUsage(null, { totalTicks: 100, idleTicks: 50 })).toBeNull();
		expect(deriveCPUUsage(
			{ totalTicks: 1_000, idleTicks: 700 },
			{ totalTicks: 900, idleTicks: 650 }
		)).toBeNull();
		expect(deriveCPUUsage(
			{ totalTicks: 1_000, idleTicks: 700 },
			{ totalTicks: 1_000, idleTicks: 700 }
		)).toBeNull();
	});

	it('derives network rates from cumulative counters and elapsed time', () => {
		const rate = deriveNetworkRate(
			{ interface: 'eth0', rxBytes: 1_000, txBytes: 2_000, sampledAtMs: 1_000 },
			{ interface: 'eth0', rxBytes: 3_000, txBytes: 3_000, sampledAtMs: 3_000 }
		);
		expect(rate).toEqual({ rxBytesPerSecond: 1_000, txBytesPerSecond: 500, totalBytesPerSecond: 1_500 });
	});

	it('resets the network baseline on interface or counter changes', () => {
		expect(deriveNetworkRate(
			{ interface: 'eth0', rxBytes: 100, txBytes: 100, sampledAtMs: 1_000 },
			{ interface: 'eth1', rxBytes: 200, txBytes: 200, sampledAtMs: 2_000 }
		)).toBeNull();
		expect(deriveNetworkRate(
			{ interface: 'eth0', rxBytes: 100, txBytes: 100, sampledAtMs: 1_000 },
			{ interface: 'eth0', rxBytes: 50, txBytes: 120, sampledAtMs: 2_000 }
		)).toBeNull();
	});
});
