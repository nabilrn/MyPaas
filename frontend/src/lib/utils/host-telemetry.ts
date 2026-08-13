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

export function boundedPercent(used: number, total: number) {
	if (!Number.isFinite(used) || !Number.isFinite(total) || total <= 0) return 0;
	return Math.max(0, Math.min(100, (used / total) * 100));
}

export function appendRollingSample(series: number[], value: number, maxSamples = 24) {
	if (!Number.isFinite(value) || maxSamples <= 0) return series.slice(-Math.max(0, maxSamples));
	return [...series, value].slice(-maxSamples);
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
