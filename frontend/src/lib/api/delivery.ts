export interface CaddyHistogramBucket {
	le: string;
	count: number;
}

export interface CaddyDeliveryStats {
	sampled_at_unix_ms: number;
	requests_total: number;
	request_errors_total: number;
	requests_in_flight: number;
	response_body_bytes_total: number;
	responses_by_status_class: Record<string, number>;
	request_duration_buckets: CaddyHistogramBucket[];
	response_ttfb_buckets: CaddyHistogramBucket[];
	upstreams_healthy: number;
	upstreams_total: number;
}

export interface DeliveryStatsResponse {
	status: 'available' | 'unavailable';
	error_code?: string;
	caddy: CaddyDeliveryStats | null;
}

async function requestDeliveryStats(retryOnUnauthorized = true): Promise<DeliveryStatsResponse> {
	const res = await fetch('/api/admin/delivery-stats', {
		credentials: 'include',
		headers: { Accept: 'application/json' }
	});

	if (res.status === 401 && retryOnUnauthorized) {
		const refreshed = await fetch('/api/auth/refresh', {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json' }
		});
		if (refreshed.ok) return requestDeliveryStats(false);
	}

	const body = await res.json().catch(() => ({}));
	if (!res.ok) {
		throw new Error(body.error?.message ?? 'Failed to load delivery telemetry');
	}
	return (body as { data: DeliveryStatsResponse }).data;
}

export const deliveryApi = {
	stats: (): Promise<DeliveryStatsResponse> => requestDeliveryStats()
};
