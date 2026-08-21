<script lang="ts">
	import { onMount } from 'svelte';
	import CapacityMetricChart from './CapacityMetricChart.svelte';
	import SectionPanel from './SectionPanel.svelte';
	import { deliveryApi, type CaddyDeliveryStats } from '$lib/api/delivery';
	import { deriveDeliveryRate, type DeliveryRate } from '$lib/utils/delivery-telemetry';
	import { appendRollingSample } from '$lib/utils/host-telemetry';

	const telemetrySamples = 40;
	let current: CaddyDeliveryStats | null = null;
	let baseline: CaddyDeliveryStats | null = null;
	let rate: DeliveryRate | null = null;
	let requestSeries: number[] = [];
	let latencySeries: number[] = [];
	let throughputSeries: number[] = [];
	let errorSeries: number[] = [];
	let status: 'loading' | 'available' | 'unavailable' = 'loading';
	let inFlight = false;

	onMount(() => {
		void load();
		const timer = setInterval(() => void load(), 3000);
		return () => clearInterval(timer);
	});

	async function load() {
		if (inFlight) return;
		inFlight = true;
		try {
			const response = await deliveryApi.stats();
			if (response.status !== 'available' || !response.caddy) {
				status = 'unavailable';
				baseline = null;
				rate = null;
				return;
			}

			status = 'available';
			current = response.caddy;
			const nextRate = deriveDeliveryRate(baseline, response.caddy);
			baseline = response.caddy;
			if (!nextRate) return;
			rate = nextRate;
			requestSeries = appendRollingSample(requestSeries, nextRate.requestsPerSecond, telemetrySamples);
			throughputSeries = appendRollingSample(throughputSeries, nextRate.responseBodyBytesPerSecond, telemetrySamples);
			errorSeries = appendRollingSample(errorSeries, nextRate.status5xxPercent, telemetrySamples);
			if (nextRate.requestP95Ms !== null) latencySeries = appendRollingSample(latencySeries, nextRate.requestP95Ms, telemetrySamples);
		} catch {
			status = 'unavailable';
			baseline = null;
			rate = null;
		} finally {
			inFlight = false;
		}
	}

	function formatBytes(value: number) {
		if (!Number.isFinite(value) || value < 0) return '-';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		let amount = value;
		let index = 0;
		while (amount >= 1024 && index < units.length - 1) {
			amount /= 1024;
			index += 1;
		}
		const digits = amount >= 100 || index === 0 ? 0 : amount >= 10 ? 1 : 2;
		return `${amount.toFixed(digits)} ${units[index]}`;
	}

	function formatDuration(value: number | null) {
		if (value === null || !Number.isFinite(value)) return '-';
		if (value >= 1000) return `${(value / 1000).toFixed(value >= 10_000 ? 1 : 2)} s`;
		return `${value.toFixed(value >= 100 ? 0 : 1)} ms`;
	}

	$: upstreamLabel = current?.upstreams_total
		? `${current.upstreams_healthy}/${current.upstreams_total} upstreams healthy`
		: 'Passive proxy path';
</script>

<SectionPanel
	title="Delivery path"
	description="Live terminal-handler telemetry from Caddy. Compare it with host Network to separate origin/proxy pressure from tunnel and NIC traffic."
	contentClass="p-0"
	className="mb-5"
>
	{#if status === 'unavailable'}
		<div class="px-4 py-5 text-sm text-gray-500 dark:text-gray-400">
			Caddy delivery metrics are unavailable. The platform remains healthy; this telemetry requires metrics-enabled Caddy configuration.
		</div>
	{:else}
		<div class="grid gap-px bg-gray-100 dark:bg-neutral-800 sm:grid-cols-2 xl:grid-cols-4">
			<CapacityMetricChart
				label="Delivery rate"
				value={rate ? `${rate.requestsPerSecond.toFixed(1)} req/s` : 'Collecting...'}
				indicator={current ? `${current.requests_in_flight.toFixed(0)} active` : ''}
				detail={upstreamLabel}
				series={requestSeries}
				resource="network"
				maxValue={null}
				rangeLabel="auto scale"
				className="bg-white dark:bg-neutral-900"
			/>
			<CapacityMetricChart
				label="Handler p95"
				value={rate ? formatDuration(rate.requestP95Ms) : 'Collecting...'}
				indicator={rate?.ttfbP95Ms !== null && rate?.ttfbP95Ms !== undefined ? `TTFB ${formatDuration(rate.ttfbP95Ms)}` : ''}
				detail="reverse_proxy + file_server histogram delta"
				series={latencySeries}
				resource="neutral"
				maxValue={null}
				rangeLabel="auto scale"
				className="bg-white dark:bg-neutral-900"
			/>
			<CapacityMetricChart
				label="Handler response body"
				value={rate ? `${formatBytes(rate.responseBodyBytesPerSecond)}/s` : 'Collecting...'}
				detail="Terminal-handler body rate; host TX also includes tunnel/protocol overhead"
				series={throughputSeries}
				resource="network"
				maxValue={null}
				rangeLabel="auto scale"
				className="bg-white dark:bg-neutral-900"
			/>
			<CapacityMetricChart
				label="5xx responses"
				value={rate ? `${rate.status5xxPercent.toFixed(2)}%` : 'Collecting...'}
				detail={rate ? `${rate.middlewareErrorsPerSecond.toFixed(2)}/s terminal-handler errors - latest sample interval` : 'Latest sample interval'}
				series={errorSeries}
				resource="neutral"
				maxValue={100}
				className="bg-white dark:bg-neutral-900"
			/>
		</div>
	{/if}
</SectionPanel>
