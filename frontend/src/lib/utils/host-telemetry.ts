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

export function boundedPercent(used: number, total: number) {
	if (!Number.isFinite(used) || !Number.isFinite(total) || total <= 0) return 0;
	return Math.max(0, Math.min(100, (used / total) * 100));
}

export function appendRollingSample(series: number[], value: number, maxSamples = 24) {
	if (!Number.isFinite(value) || maxSamples <= 0) return series.slice(-Math.max(0, maxSamples));
	return [...series, value].slice(-maxSamples);
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
