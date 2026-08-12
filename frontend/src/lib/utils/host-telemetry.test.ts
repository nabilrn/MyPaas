import { describe, expect, it } from 'vitest';
import { appendRollingSample, boundedPercent, deriveNetworkRate } from './host-telemetry';

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
