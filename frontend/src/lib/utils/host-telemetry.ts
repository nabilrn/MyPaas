export type CPUCounterSample = {
	totalTicks: number;
	idleTicks: number;
};

export type NetworkCounterSample = {
	interface: string;
	rxBytes: number;
	txBytes: number;
	sampledAtMs: number;
};

export type NetworkRate = {
	rxBytesPerSecond: number;
	txBytesPerSecond: number;
	totalBytesPerSecond: number;
};

export type MetricDomain = {
	min: number;
	max: number;
};

export type AdaptiveRateScale = MetricDomain & {
	exponent: number;
	midpoint: number;
};

export type HostTelemetrySample = {
	sampledAtMs: number;
	memoryPercent: number | null;
	cpuPercent: number | null;
	networkBytesPerSecond: number | null;
};

export function boundedPercent(used: number, total: number) {
	if (!Number.isFinite(used) || !Number.isFinite(total) || total <= 0) return 0;
	return Math.max(0, Math.min(100, (used / total) * 100));
}

export function appendRollingSample(series: number[], value: number, maxSamples = 24) {
	if (!Number.isFinite(value) || maxSamples <= 0) return series.slice(-Math.max(0, maxSamples));
	return [...series, value].slice(-maxSamples);
}

export function appendHostTelemetrySample(series: HostTelemetrySample[], sample: HostTelemetrySample, maxSamples = 24) {
	if (!Number.isFinite(sample.sampledAtMs) || maxSamples <= 0) return series.slice(-Math.max(0, maxSamples));
	return [...series, sample].slice(-maxSamples);
}

export function deriveTelemetryCadenceMs(series: HostTelemetrySample[], fallbackMs = 3000) {
	const deltas: number[] = [];
	for (let index = 1; index < series.length; index += 1) {
		const delta = series[index].sampledAtMs - series[index - 1].sampledAtMs;
		if (Number.isFinite(delta) && delta > 0) deltas.push(delta);
	}
	if (deltas.length === 0) return fallbackMs;

	// Ignore the slowest quartile so browser-background pauses and transient
	// request stalls do not redefine the expected polling cadence.
	const sorted = [...deltas].sort((a, b) => a - b);
	const stableCount = Math.max(1, Math.ceil(sorted.length * 0.75));
	const stable = sorted.slice(0, stableCount);
	const middle = Math.floor(stable.length / 2);
	const median = stable.length % 2 === 1
		? stable[middle]
		: (stable[middle - 1] + stable[middle]) / 2;

	return Number.isFinite(median) && median > 0 ? median : fallbackMs;
}

export function isTelemetryDiscontinuity(
	previous: HostTelemetrySample | undefined,
	current: HostTelemetrySample | undefined,
	cadenceMs: number,
	multiplier = 2.5
) {
	if (!previous || !current) return false;
	const delta = current.sampledAtMs - previous.sampledAtMs;
	if (!Number.isFinite(delta) || delta <= 0) return false;
	const safeCadence = Number.isFinite(cadenceMs) && cadenceMs > 0 ? cadenceMs : 3000;
	return delta > safeCadence * multiplier;
}

export function deriveAdaptiveMetricDomain(series: number[], maxValue: number | null = 100): MetricDomain {
	const clean = series.filter((sample) => Number.isFinite(sample) && sample >= 0);
	const rawMin = clean.length > 0 ? Math.min(...clean) : 0;
	const rawMax = clean.length > 0 ? Math.max(...clean) : 0;
	const hardMax = maxValue !== null && Number.isFinite(maxValue) && maxValue > 0 ? maxValue : null;

	if (clean.length < 2) {
		return {
			min: 0,
			max: hardMax ?? Math.max(1, rawMax * 1.15)
		};
	}

	const observedSpan = Math.max(0, rawMax - rawMin);
	const minimumSpan = hardMax !== null
		? rawMax <= 1
			? 0.25
			: rawMax <= 5
				? 1
				: Math.max(2, hardMax * 0.02)
		: Math.max(1, rawMax * 0.05);
	const desiredSpan = Math.max(minimumSpan, observedSpan * (hardMax !== null ? 1.6 : 1.5));
	const center = (rawMin + rawMax) / 2;
	let min = center - desiredSpan / 2;
	let max = center + desiredSpan / 2;

	if (rawMin <= desiredSpan * 0.2 || min < 0) {
		min = 0;
		max = Math.max(desiredSpan, rawMax * 1.15);
	}
	if (hardMax !== null && hardMax - rawMax <= desiredSpan * 0.5) {
		max = hardMax;
		min = Math.max(0, hardMax - desiredSpan);
	} else if (hardMax !== null && max > hardMax) {
		min -= max - hardMax;
		max = hardMax;
		if (min < 0) min = 0;
	}
	if (max <= min) {
		max = hardMax !== null ? Math.min(hardMax, min + 1) : min + 1;
	}

	return { min, max };
}

export function deriveAdaptiveRateScale(series: number[]): AdaptiveRateScale {
	const clean = series.filter((sample) => Number.isFinite(sample) && sample >= 0);
	const peak = clean.length > 0 ? Math.max(...clean) : 0;
	if (peak <= 0) {
		return { min: 0, max: 1, exponent: 1, midpoint: 0.5 };
	}

	// Keep the highest observed rate below the top edge while rounding the
	// ceiling to a readable engineering-style value.
	const paddedPeak = peak / 0.88;
	const magnitude = 10 ** Math.floor(Math.log10(paddedPeak));
	const normalizedPeak = paddedPeak / magnitude;
	const niceSteps = [1, 1.25, 1.5, 2, 2.5, 3, 4, 5, 7.5, 10];
	const niceStep = niceSteps.find((step) => step >= normalizedPeak) ?? 10;
	const max = niceStep * magnitude;

	// Network traffic is often bursty. Choose a bounded power exponent so a
	// representative non-zero rate remains readable without clipping peaks.
	const positives = clean.filter((sample) => sample > 0).sort((a, b) => a - b);
	const median = positives.length === 0
		? 0
		: positives.length % 2 === 1
			? positives[(positives.length - 1) / 2]
			: (positives[positives.length / 2 - 1] + positives[positives.length / 2]) / 2;
	const medianRatio = median > 0 ? Math.min(1, median / max) : 0;
	const targetMedianPosition = 0.35;
	const rawExponent = medianRatio > 0 && medianRatio < targetMedianPosition
		? Math.log(targetMedianPosition) / Math.log(medianRatio)
		: 1;
	const exponent = Math.max(0.5, Math.min(1, rawExponent));
	const midpoint = max * 0.5 ** (1 / exponent);

	return { min: 0, max, exponent, midpoint };
}

export function deriveCPUUsage(previous: CPUCounterSample | null, current: CPUCounterSample): number | null {
	if (!previous) return null;
	if (!Number.isFinite(current.totalTicks) || !Number.isFinite(current.idleTicks)) return null;
	if (current.totalTicks < previous.totalTicks || current.idleTicks < previous.idleTicks) return null;

	const deltaTotal = current.totalTicks - previous.totalTicks;
	const deltaIdle = current.idleTicks - previous.idleTicks;
	if (deltaTotal <= 0 || deltaIdle < 0 || deltaIdle > deltaTotal) return null;

	return Math.max(0, Math.min(100, ((deltaTotal - deltaIdle) / deltaTotal) * 100));
}

export function deriveNetworkRate(previous: NetworkCounterSample | null, current: NetworkCounterSample): NetworkRate | null {
	if (!previous || !current.interface || current.interface !== previous.interface) return null;
	const elapsedSeconds = (current.sampledAtMs - previous.sampledAtMs) / 1000;
	if (!Number.isFinite(elapsedSeconds) || elapsedSeconds <= 0) return null;
	if (current.rxBytes < previous.rxBytes || current.txBytes < previous.txBytes) return null;

	const rxBytesPerSecond = (current.rxBytes - previous.rxBytes) / elapsedSeconds;
	const txBytesPerSecond = (current.txBytes - previous.txBytes) / elapsedSeconds;
	if (!Number.isFinite(rxBytesPerSecond) || !Number.isFinite(txBytesPerSecond)) return null;
	return {
		rxBytesPerSecond,
		txBytesPerSecond,
		totalBytesPerSecond: rxBytesPerSecond + txBytesPerSecond
	};
}
