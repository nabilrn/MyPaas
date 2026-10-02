import { describe, expect, it } from 'vitest';
import { appendHostTelemetrySample, appendRollingSample, boundedPercent, deriveAdaptiveMetricDomain, deriveAdaptiveRateScale, deriveCPUUsage, deriveNetworkRate } from './host-telemetry';

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

	it('keeps host metrics aligned in one bounded telemetry history', () => {
		const first = {
			sampledAtMs: 1_000,
			memoryPercent: 36,
			cpuPercent: null,
			networkBytesPerSecond: null
		};
		const second = {
			sampledAtMs: 4_000,
			memoryPercent: 37,
			cpuPercent: 24.9,
			networkBytesPerSecond: 34_900
		};
		const third = {
			sampledAtMs: 7_000,
			memoryPercent: null,
			cpuPercent: 25.8,
			networkBytesPerSecond: 3_640
		};

		expect(appendHostTelemetrySample([first, second], third, 2)).toEqual([second, third]);
	});

	it('zooms percentage domains enough to show small utilization movement', () => {
		const domain = deriveAdaptiveMetricDomain([38.7, 39.1, 39.4], 100);
		expect(domain.min).toBeLessThanOrEqual(38.7);
		expect(domain.max).toBeGreaterThanOrEqual(39.4);
		expect(domain.max - domain.min).toBeCloseTo(2, 5);
	});

	it('magnifies very small non-zero utilization without inventing movement', () => {
		const domain = deriveAdaptiveMetricDomain([0.82, 0.86, 0.9], 100);
		expect(domain.min).toBeGreaterThan(0);
		expect(domain.max).toBeGreaterThanOrEqual(0.9);
		expect(domain.max - domain.min).toBeCloseTo(0.25, 5);
	});

	it('keeps adaptive percentage domains inside physical bounds', () => {
		const low = deriveAdaptiveMetricDomain([0.1, 0.5], 100);
		expect(low.min).toBe(0);
		expect(low.max).toBeCloseTo(0.64, 5);

		const high = deriveAdaptiveMetricDomain([98.5, 99.2], 100);
		expect(high.max).toBe(100);
		expect(high.min).toBeLessThanOrEqual(98.5);
	});

	it('uses a relative rolling scale for unbounded rate metrics', () => {
		const domain = deriveAdaptiveMetricDomain([100, 105, 110], null);
		expect(domain.min).toBeLessThanOrEqual(100);
		expect(domain.max).toBeGreaterThanOrEqual(110);
		expect(domain.max - domain.min).toBeCloseTo(15, 5);
	});

	it('anchors adaptive network scales at zero with readable headroom', () => {
		const scale = deriveAdaptiveRateScale([0, 512, 1024, 3840]);
		expect(scale.min).toBe(0);
		expect(scale.max).toBeGreaterThan(3840);
		expect(3840 / scale.max).toBeLessThanOrEqual(0.88);
		expect(scale.midpoint).toBeGreaterThan(0);
		expect(scale.midpoint).toBeLessThan(scale.max);
	});

	it('compresses bursty network peaks without turning balanced traffic nonlinear', () => {
		const bursty = deriveAdaptiveRateScale([0, 80, 100, 120, 5000]);
		const balanced = deriveAdaptiveRateScale([800, 900, 1000, 1100]);
		expect(bursty.exponent).toBeGreaterThanOrEqual(0.5);
		expect(bursty.exponent).toBeLessThan(1);
		expect(balanced.exponent).toBe(1);
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

	it('derives network rate across a long collection pause from real elapsed time', () => {
		const rate = deriveNetworkRate(
			{ interface: 'eth0', rxBytes: 10_000, txBytes: 20_000, sampledAtMs: 1_000 },
			{ interface: 'eth0', rxBytes: 70_000, txBytes: 50_000, sampledAtMs: 61_000 }
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
