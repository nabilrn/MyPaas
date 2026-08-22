import { describe, expect, it } from 'vitest';
import type { CaddyDeliveryStats } from '$lib/api/delivery';
import { deriveDeliveryRate, histogramDeltaQuantile } from './delivery-telemetry';

function sample(overrides: Partial<CaddyDeliveryStats> = {}): CaddyDeliveryStats {
	return {
		sampled_at_unix_ms: 1_000,
		requests_total: 100,
		request_errors_total: 2,
		requests_in_flight: 1,
		response_body_bytes_total: 1_000_000,
		responses_by_status_class: { '2xx': 98, '3xx': 0, '4xx': 1, '5xx': 1 },
		request_duration_buckets: [
			{ le: '0.1', count: 50 },
			{ le: '0.5', count: 95 },
			{ le: '+Inf', count: 100 }
		],
		response_ttfb_buckets: [
			{ le: '0.05', count: 80 },
			{ le: '0.25', count: 98 },
			{ le: '+Inf', count: 100 }
		],
		upstreams_healthy: 1,
		upstreams_total: 1,
		...overrides
	};
}

describe('delivery telemetry helpers', () => {
	it('derives request, byte and 5xx rates from cumulative Caddy counters', () => {
		const previous = sample();
		const current = sample({
			sampled_at_unix_ms: 3_000,
			requests_total: 140,
			request_errors_total: 4,
			response_body_bytes_total: 1_800_000,
			responses_by_status_class: { '2xx': 136, '3xx': 0, '4xx': 1, '5xx': 3 },
			request_duration_buckets: [
				{ le: '0.1', count: 70 },
				{ le: '0.5', count: 133 },
				{ le: '+Inf', count: 140 }
			],
			response_ttfb_buckets: [
				{ le: '0.05', count: 112 },
				{ le: '0.25', count: 138 },
				{ le: '+Inf', count: 140 }
			]
		});

		const rate = deriveDeliveryRate(previous, current);
		expect(rate?.requestsPerSecond).toBe(20);
		expect(rate?.responseBodyBytesPerSecond).toBe(400_000);
		expect(rate?.middlewareErrorsPerSecond).toBe(1);
		expect(rate?.status5xxPercent).toBe(5);
		expect(rate?.requestP95Ms).not.toBeNull();
		expect(rate?.ttfbP95Ms).not.toBeNull();
	});

	it('computes a quantile from histogram deltas instead of process-lifetime totals', () => {
		const previous = [
			{ le: '0.1', count: 50 },
			{ le: '0.5', count: 95 },
			{ le: '+Inf', count: 100 }
		];
		const current = [
			{ le: '0.1', count: 60 },
			{ le: '0.5', count: 114 },
			{ le: '+Inf', count: 120 }
		];
		const p95 = histogramDeltaQuantile(previous, current, 0.95);
		expect(p95).toBeGreaterThan(100);
		expect(p95).toBeLessThanOrEqual(500);
	});

	it('resets the rate baseline when Caddy counters reset', () => {
		const previous = sample({ requests_total: 100, response_body_bytes_total: 1_000_000 });
		const current = sample({ sampled_at_unix_ms: 2_000, requests_total: 5, response_body_bytes_total: 10_000 });
		expect(deriveDeliveryRate(previous, current)).toBeNull();
	});
});
