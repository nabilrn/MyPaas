import type { CaddyDeliveryStats, CaddyHistogramBucket } from '$lib/api/delivery';

export type DeliveryRate = {
	requestsPerSecond: number;
	responseBodyBytesPerSecond: number;
	middlewareErrorsPerSecond: number;
	status5xxPercent: number;
	requestP95Ms: number | null;
	ttfbP95Ms: number | null;
};

export function deriveDeliveryRate(previous: CaddyDeliveryStats | null, current: CaddyDeliveryStats): DeliveryRate | null {
	if (!previous) return null;
	const elapsedSeconds = (current.sampled_at_unix_ms - previous.sampled_at_unix_ms) / 1000;
	if (!Number.isFinite(elapsedSeconds) || elapsedSeconds <= 0) return null;
	if (
		current.requests_total < previous.requests_total ||
		current.response_body_bytes_total < previous.response_body_bytes_total ||
		current.request_errors_total < previous.request_errors_total
	) {
		return null;
	}

	const requestDelta = current.requests_total - previous.requests_total;
	const errorDelta = current.request_errors_total - previous.request_errors_total;
	const bodyBytesDelta = current.response_body_bytes_total - previous.response_body_bytes_total;
	const previous5xx = previous.responses_by_status_class?.['5xx'] ?? 0;
	const current5xx = current.responses_by_status_class?.['5xx'] ?? 0;
	const status5xxDelta = current5xx >= previous5xx ? current5xx - previous5xx : 0;

	return {
		requestsPerSecond: requestDelta / elapsedSeconds,
		responseBodyBytesPerSecond: bodyBytesDelta / elapsedSeconds,
		middlewareErrorsPerSecond: errorDelta / elapsedSeconds,
		status5xxPercent: requestDelta > 0 ? (status5xxDelta / requestDelta) * 100 : 0,
		requestP95Ms: histogramDeltaQuantile(previous.request_duration_buckets, current.request_duration_buckets, 0.95),
		ttfbP95Ms: histogramDeltaQuantile(previous.response_ttfb_buckets, current.response_ttfb_buckets, 0.95)
	};
}

export function histogramDeltaQuantile(previous: CaddyHistogramBucket[], current: CaddyHistogramBucket[], quantile: number): number | null {
	if (!Number.isFinite(quantile) || quantile <= 0 || quantile >= 1) return null;
	const previousCounts = new Map(previous.map((bucket) => [bucket.le, bucket.count]));
	const deltaBuckets = current
		.map((bucket) => ({
			upperBound: parseUpperBound(bucket.le),
			count: bucket.count - (previousCounts.get(bucket.le) ?? 0)
		}))
		.filter((bucket) => Number.isFinite(bucket.count) && bucket.count >= 0)
		.sort((a, b) => a.upperBound - b.upperBound);

	if (deltaBuckets.length === 0) return null;
	const totalBucket = deltaBuckets.find((bucket) => !Number.isFinite(bucket.upperBound)) ?? deltaBuckets[deltaBuckets.length - 1];
	const total = totalBucket.count;
	if (!Number.isFinite(total) || total <= 0) return null;

	const target = total * quantile;
	let previousCount = 0;
	let previousUpperBound = 0;
	let lastFiniteUpperBound = 0;

	for (const bucket of deltaBuckets) {
		if (!Number.isFinite(bucket.upperBound)) {
			return lastFiniteUpperBound > 0 ? lastFiniteUpperBound * 1000 : null;
		}
		lastFiniteUpperBound = bucket.upperBound;
		if (bucket.count >= target) {
			const observationsInBucket = bucket.count - previousCount;
			if (observationsInBucket <= 0) return bucket.upperBound * 1000;
			const fraction = Math.max(0, Math.min(1, (target - previousCount) / observationsInBucket));
			return (previousUpperBound + (bucket.upperBound - previousUpperBound) * fraction) * 1000;
		}
		previousCount = bucket.count;
		previousUpperBound = bucket.upperBound;
	}

	return lastFiniteUpperBound > 0 ? lastFiniteUpperBound * 1000 : null;
}

function parseUpperBound(value: string) {
	if (value === '+Inf' || value === 'Inf') return Number.POSITIVE_INFINITY;
	const parsed = Number.parseFloat(value);
	return Number.isFinite(parsed) ? parsed : Number.POSITIVE_INFINITY;
}
