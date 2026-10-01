<script lang="ts">
	import { deriveAdaptiveMetricDomain, type HostTelemetrySample } from '$lib/utils/host-telemetry';

	type SeriesKey = 'memory' | 'cpu' | 'network';
	type ChartPoint = { x: number; y: number };

	export let memoryLabel = 'RAM usage';
	export let memoryValue = 'Unavailable';
	export let memoryIndicator = '';
	export let memoryDetail = '';
	export let cpuValue = 'Unavailable';
	export let cpuIndicator = '';
	export let cpuDetail = '';
	export let networkValue = 'Unavailable';
	export let networkIndicator = '';
	export let networkDetail = '';
	export let storageValue = 'Unavailable';
	export let storageIndicator = '';
	export let storageDetail = '';
	export let storagePercent = 0;
	export let samples: HostTelemetrySample[] = [];

	const chartWidth = 1000;
	const chartHeight = 112;
	const chartPaddingY = 8;
	const curveTension = 0.68;

	let hoverIndex = -1;
	let visibleSeries: Record<SeriesKey, boolean> = {
		memory: true,
		cpu: true,
		network: true
	};

	const seriesClasses = {
		memory: {
			dot: 'bg-emerald-400/80 dark:bg-emerald-300/75',
			stroke: 'stroke-emerald-500/75 dark:stroke-emerald-300/70',
			point: 'fill-emerald-500 dark:fill-emerald-300'
		},
		cpu: {
			dot: 'bg-sky-400/80 dark:bg-sky-300/75',
			stroke: 'stroke-sky-500/75 dark:stroke-sky-300/70',
			point: 'fill-sky-500 dark:fill-sky-300'
		},
		network: {
			dot: 'bg-violet-400/80 dark:bg-violet-300/75',
			stroke: 'stroke-violet-500/75 dark:stroke-violet-300/70',
			point: 'fill-violet-500 dark:fill-violet-300'
		}
	} as const;

	function clamp(value: number, min: number, max: number) {
		return Math.max(min, Math.min(max, value));
	}

	function toggleSeries(series: SeriesKey) {
		visibleSeries = { ...visibleSeries, [series]: !visibleSeries[series] };
	}

	function formatRate(value: number | null) {
		if (value === null || !Number.isFinite(value)) return '—';
		const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
		let amount = value;
		let unitIndex = 0;
		while (amount >= 1024 && unitIndex < units.length - 1) {
			amount /= 1024;
			unitIndex += 1;
		}
		const digits = amount >= 100 || unitIndex === 0 ? 0 : amount >= 10 ? 1 : 2;
		return `${amount.toFixed(digits)} ${units[unitIndex]}`;
	}

	function formatSampleTime(sampledAtMs: number) {
		return new Date(sampledAtMs).toLocaleTimeString([], {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	function buildSmoothPath(input: ChartPoint[]) {
		if (input.length === 0) return '';
		if (input.length === 1) return `M ${input[0].x.toFixed(2)},${input[0].y.toFixed(2)}`;

		let path = `M ${input[0].x.toFixed(2)},${input[0].y.toFixed(2)}`;
		for (let index = 0; index < input.length - 1; index += 1) {
			const p0 = input[index - 1] ?? input[index];
			const p1 = input[index];
			const p2 = input[index + 1];
			const p3 = input[index + 2] ?? p2;

			const cp1x = clamp(p1.x + ((p2.x - p0.x) / 6) * curveTension, 0, chartWidth);
			const cp1y = clamp(p1.y + ((p2.y - p0.y) / 6) * curveTension, 0, chartHeight);
			const cp2x = clamp(p2.x - ((p3.x - p1.x) / 6) * curveTension, 0, chartWidth);
			const cp2y = clamp(p2.y - ((p3.y - p1.y) / 6) * curveTension, 0, chartHeight);

			path += ` C ${cp1x.toFixed(2)},${cp1y.toFixed(2)} ${cp2x.toFixed(2)},${cp2y.toFixed(2)} ${p2.x.toFixed(2)},${p2.y.toFixed(2)}`;
		}
		return path;
	}

	function seriesValue(sample: HostTelemetrySample, series: SeriesKey) {
		if (series === 'memory') return sample.memoryPercent;
		if (series === 'cpu') return sample.cpuPercent;
		return sample.networkBytesPerSecond;
	}

	function seriesY(value: number, series: SeriesKey, networkScale: { min: number; max: number }) {
		const usableHeight = chartHeight - chartPaddingY * 2;
		if (series === 'network') {
			const span = Math.max(0.0001, networkScale.max - networkScale.min);
			const normalized = clamp((value - networkScale.min) / span, 0, 1);
			return chartPaddingY + (1 - normalized) * usableHeight;
		}
		return chartPaddingY + (1 - clamp(value, 0, 100) / 100) * usableHeight;
	}

	function buildSeriesPaths(series: SeriesKey, sourceSamples: HostTelemetrySample[], networkScale: { min: number; max: number }) {
		const paths: string[] = [];
		let segment: ChartPoint[] = [];
		const flush = () => {
			if (segment.length > 0) paths.push(buildSmoothPath(segment));
			segment = [];
		};

		sourceSamples.forEach((sample, index) => {
			const value = seriesValue(sample, series);
			if (value === null || !Number.isFinite(value)) {
				flush();
				return;
			}
			const x = sourceSamples.length <= 1 ? chartWidth : (index / (sourceSamples.length - 1)) * chartWidth;
			segment.push({ x, y: seriesY(value, series, networkScale) });
		});
		flush();
		return paths;
	}

	function pointFor(series: SeriesKey, index: number, sourceSamples: HostTelemetrySample[], networkScale: { min: number; max: number }) {
		const sample = sourceSamples[index];
		if (!sample) return null;
		const value = seriesValue(sample, series);
		if (value === null || !Number.isFinite(value)) return null;
		const x = sourceSamples.length <= 1 ? chartWidth : (index / (sourceSamples.length - 1)) * chartWidth;
		return { x, y: seriesY(value, series, networkScale) };
	}

	function isolatedPoints(series: SeriesKey, sourceSamples: HostTelemetrySample[], networkScale: { min: number; max: number }) {
		return sourceSamples.flatMap((sample, index) => {
			const value = seriesValue(sample, series);
			if (value === null || !Number.isFinite(value)) return [];
			const previous = index > 0 ? seriesValue(sourceSamples[index - 1], series) : null;
			const next = index < sourceSamples.length - 1 ? seriesValue(sourceSamples[index + 1], series) : null;
			const hasPrevious = previous !== null && Number.isFinite(previous);
			const hasNext = next !== null && Number.isFinite(next);
			if (hasPrevious || hasNext) return [];
			const x = sourceSamples.length <= 1 ? chartWidth : (index / (sourceSamples.length - 1)) * chartWidth;
			return [{ x, y: seriesY(value, series, networkScale) }];
		});
	}

	function handleChartPointer(event: PointerEvent) {
		if (samples.length === 0) return;
		const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect();
		const ratio = clamp((event.clientX - bounds.left) / Math.max(1, bounds.width), 0, 1);
		hoverIndex = samples.length === 1 ? 0 : Math.round(ratio * (samples.length - 1));
	}

	function handleChartKeydown(event: KeyboardEvent) {
		if (samples.length === 0 || (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight')) return;
		event.preventDefault();
		const start = hoverIndex < 0 ? samples.length - 1 : hoverIndex;
		hoverIndex = clamp(start + (event.key === 'ArrowRight' ? 1 : -1), 0, samples.length - 1);
	}

	$: networkValues = samples
		.map((sample) => sample.networkBytesPerSecond)
		.filter((value): value is number => value !== null && Number.isFinite(value));
	$: networkDomain = deriveAdaptiveMetricDomain(networkValues, null);
	$: memoryPaths = buildSeriesPaths('memory', samples, networkDomain);
	$: cpuPaths = buildSeriesPaths('cpu', samples, networkDomain);
	$: networkPaths = buildSeriesPaths('network', samples, networkDomain);
	$: memoryIsolatedPoints = isolatedPoints('memory', samples, networkDomain);
	$: cpuIsolatedPoints = isolatedPoints('cpu', samples, networkDomain);
	$: networkIsolatedPoints = isolatedPoints('network', samples, networkDomain);
	$: hoveredSample = hoverIndex >= 0 && hoverIndex < samples.length ? samples[hoverIndex] : null;
	$: hoveredX = hoverIndex >= 0 && samples.length > 0
		? (samples.length === 1 ? chartWidth : (hoverIndex / Math.max(1, samples.length - 1)) * chartWidth)
		: null;
	$: tooltipLeft = hoveredX === null ? 50 : clamp((hoveredX / chartWidth) * 100, 12, 88);
	$: memoryHoverPoint = hoverIndex >= 0 ? pointFor('memory', hoverIndex, samples, networkDomain) : null;
	$: cpuHoverPoint = hoverIndex >= 0 ? pointFor('cpu', hoverIndex, samples, networkDomain) : null;
	$: networkHoverPoint = hoverIndex >= 0 ? pointFor('network', hoverIndex, samples, networkDomain) : null;
	$: storageAvailable = !/unavailable/i.test(`${storageValue} ${storageDetail}`);
	$: usedStoragePercent = clamp(Number.isFinite(storagePercent) ? storagePercent : 0, 0, 100);
	$: storageFillClass = usedStoragePercent >= 90
		? 'bg-red-500 dark:bg-red-400'
		: usedStoragePercent >= 80
			? 'bg-orange-500 dark:bg-orange-400'
			: 'bg-amber-400 dark:bg-amber-300';
</script>

<div class="bg-white dark:bg-neutral-900" data-host-resource-overview>
	<div class="grid grid-cols-2 gap-px bg-gray-100 dark:bg-neutral-800 xl:grid-cols-4">
		<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-900">
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2">
					<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-400/80 dark:bg-emerald-300/75"></span>
					<p class="metric-label truncate">{memoryLabel}</p>
				</div>
				{#if memoryIndicator}<span class="metric-value shrink-0 text-[11px] font-medium text-gray-500 dark:text-gray-400">{memoryIndicator}</span>{/if}
			</div>
			<p class="metric-value mt-1 truncate text-[15px] font-semibold tracking-tight text-gray-950 dark:text-white">{memoryValue}</p>
			{#if memoryDetail}<p class="mt-1 truncate text-[11px] text-gray-500 dark:text-gray-400" title={memoryDetail}>{memoryDetail}</p>{/if}
		</div>

		<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-900">
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2">
					<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-sky-400/80 dark:bg-sky-300/75"></span>
					<p class="metric-label truncate">CPU usage</p>
				</div>
				{#if cpuIndicator}<span class="metric-value shrink-0 text-[11px] font-medium text-gray-500 dark:text-gray-400">{cpuIndicator}</span>{/if}
			</div>
			<p class="metric-value mt-1 truncate text-[15px] font-semibold tracking-tight text-gray-950 dark:text-white">{cpuValue}</p>
			{#if cpuDetail}<p class="mt-1 truncate text-[11px] text-gray-500 dark:text-gray-400" title={cpuDetail}>{cpuDetail}</p>{/if}
		</div>

		<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-900">
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2">
					<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-violet-400/80 dark:bg-violet-300/75"></span>
					<p class="metric-label truncate">Network</p>
				</div>
				{#if networkIndicator}<span class="metric-value shrink-0 text-[11px] font-medium text-gray-500 dark:text-gray-400">{networkIndicator}</span>{/if}
			</div>
			<p class="metric-value mt-1 truncate text-[15px] font-semibold tracking-tight text-gray-950 dark:text-white">{networkValue}</p>
			{#if networkDetail}<p class="mt-1 truncate text-[11px] text-gray-500 dark:text-gray-400" title={networkDetail}>{networkDetail}</p>{/if}
		</div>

		<div class="min-w-0 bg-white px-4 py-3 dark:bg-neutral-900">
			<div class="flex items-center justify-between gap-3">
				<div class="flex min-w-0 items-center gap-2">
					<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-amber-400/80 dark:bg-amber-300/75"></span>
					<p class="metric-label truncate">Storage</p>
				</div>
				{#if storageIndicator}<span class="metric-value shrink-0 text-[11px] font-medium text-gray-500 dark:text-gray-400">{storageIndicator}</span>{/if}
			</div>
			<p class="metric-value mt-1 truncate text-[15px] font-semibold tracking-tight text-gray-950 dark:text-white">{storageValue}</p>
			{#if storageDetail}<p class="mt-1 truncate text-[11px] text-gray-500 dark:text-gray-400" title={storageDetail}>{storageDetail}</p>{/if}
		</div>
	</div>

	<div class="border-t border-gray-100 dark:border-neutral-800">
		<div class="flex min-h-10 flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-2 text-xs text-gray-500 dark:text-gray-400">
			<div class="min-w-0">
				<p class="font-medium text-gray-700 dark:text-gray-300">Resource history</p>
				<p class="mt-0.5 text-[11px] text-gray-400 dark:text-gray-500">RAM and CPU use a 0–100% scale. Network uses its own adaptive rate scale.</p>
			</div>
			<div class="flex items-center gap-3" aria-label="Chart series visibility">
			<button
				type="button"
				class="app-focus inline-flex items-center gap-1.5 rounded-sm px-1 py-0.5 transition-opacity"
				class:opacity-40={!visibleSeries.memory}
				aria-pressed={visibleSeries.memory}
				on:click={() => toggleSeries('memory')}
			>
				<span class={`h-1.5 w-1.5 rounded-full ${seriesClasses.memory.dot}`}></span>
				<span>Memory</span>
			</button>
			<button
				type="button"
				class="app-focus inline-flex items-center gap-1.5 rounded-sm px-1 py-0.5 transition-opacity"
				class:opacity-40={!visibleSeries.cpu}
				aria-pressed={visibleSeries.cpu}
				on:click={() => toggleSeries('cpu')}
			>
				<span class={`h-1.5 w-1.5 rounded-full ${seriesClasses.cpu.dot}`}></span>
				<span>CPU</span>
			</button>
			<button
				type="button"
				class="app-focus inline-flex items-center gap-1.5 rounded-sm px-1 py-0.5 transition-opacity"
				class:opacity-40={!visibleSeries.network}
				aria-pressed={visibleSeries.network}
				on:click={() => toggleSeries('network')}
				title="Network history uses an adaptive rate scale"
			>
				<span class={`h-1.5 w-1.5 rounded-full ${seriesClasses.network.dot}`}></span>
				<span>Network</span>
			</button>
			</div>
		</div>

		<div
			class="app-focus relative h-28 overflow-hidden outline-none"
			role="img"
			aria-label="Host resource history. Memory and CPU use percentage scale; network uses an adaptive rate scale. Use the series controls to hide or show lines."
			tabindex={samples.length > 0 ? 0 : undefined}
			on:pointermove={handleChartPointer}
			on:pointerleave={() => (hoverIndex = -1)}
			on:focus={() => {
				if (samples.length > 0 && hoverIndex < 0) hoverIndex = samples.length - 1;
			}}
			on:blur={() => (hoverIndex = -1)}
			on:keydown={handleChartKeydown}
		>
			<div class="pointer-events-none absolute inset-y-1 left-2 z-[1] flex flex-col justify-between text-[10px] tabular-nums text-gray-400 dark:text-gray-500" aria-hidden="true">
				<span>100%</span>
				<span>50%</span>
				<span>0%</span>
			</div>
			<div class="pointer-events-none absolute inset-y-1 right-2 z-[1] flex flex-col items-end justify-between text-[10px] tabular-nums text-violet-500/70 dark:text-violet-300/60" aria-hidden="true">
				<span>{formatRate(networkDomain.max)}</span>
				<span>Network</span>
				<span>{formatRate(networkDomain.min)}</span>
			</div>
			<div class="pointer-events-none absolute bottom-1 left-10 right-10 z-[1] flex justify-between text-[10px] text-gray-400 dark:text-gray-500" aria-hidden="true">
				<span>Earlier</span>
				<span>Now</span>
			</div>

			<svg class="h-full w-full px-10 pb-4 pt-1" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none" aria-hidden="true">
				<g class="stroke-gray-200/45 dark:stroke-neutral-700/40" stroke-width="0.7">
					<line x1={chartWidth * 0.2} x2={chartWidth * 0.2} y1="0" y2={chartHeight} />
					<line x1={chartWidth * 0.4} x2={chartWidth * 0.4} y1="0" y2={chartHeight} />
					<line x1={chartWidth * 0.6} x2={chartWidth * 0.6} y1="0" y2={chartHeight} />
					<line x1={chartWidth * 0.8} x2={chartWidth * 0.8} y1="0" y2={chartHeight} />
					<line x1="0" x2={chartWidth} y1={chartHeight * 0.25} y2={chartHeight * 0.25} />
					<line x1="0" x2={chartWidth} y1={chartHeight * 0.5} y2={chartHeight * 0.5} />
					<line x1="0" x2={chartWidth} y1={chartHeight * 0.75} y2={chartHeight * 0.75} />
				</g>

				{#if visibleSeries.memory}
					{#each memoryPaths as path}
						<path d={path} fill="none" class={seriesClasses.memory.stroke} stroke-width="1.85" stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
					{/each}
					{#each memoryIsolatedPoints as point}<circle cx={point.x} cy={point.y} r="1.9" class={seriesClasses.memory.point} />{/each}
				{/if}
				{#if visibleSeries.cpu}
					{#each cpuPaths as path}
						<path d={path} fill="none" class={seriesClasses.cpu.stroke} stroke-width="1.85" stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
					{/each}
					{#each cpuIsolatedPoints as point}<circle cx={point.x} cy={point.y} r="1.9" class={seriesClasses.cpu.point} />{/each}
				{/if}
				{#if visibleSeries.network}
					{#each networkPaths as path}
						<path d={path} fill="none" class={seriesClasses.network.stroke} stroke-width="1.85" stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
					{/each}
					{#each networkIsolatedPoints as point}<circle cx={point.x} cy={point.y} r="1.9" class={seriesClasses.network.point} />{/each}
				{/if}

				{#if hoveredX !== null}
					<line x1={hoveredX} x2={hoveredX} y1="0" y2={chartHeight} class="stroke-gray-500/30 dark:stroke-gray-400/30" stroke-width="0.8" vector-effect="non-scaling-stroke" />
					{#if visibleSeries.memory && memoryHoverPoint}<circle cx={memoryHoverPoint.x} cy={memoryHoverPoint.y} r="2.3" class={seriesClasses.memory.point} />{/if}
					{#if visibleSeries.cpu && cpuHoverPoint}<circle cx={cpuHoverPoint.x} cy={cpuHoverPoint.y} r="2.3" class={seriesClasses.cpu.point} />{/if}
					{#if visibleSeries.network && networkHoverPoint}<circle cx={networkHoverPoint.x} cy={networkHoverPoint.y} r="2.3" class={seriesClasses.network.point} />{/if}
				{/if}
			</svg>

			{#if hoveredSample}
				<div
					class="pointer-events-none absolute top-2 z-10 -translate-x-1/2 rounded-md border border-gray-200 bg-white/95 px-2.5 py-2 shadow-sm backdrop-blur dark:border-neutral-700 dark:bg-neutral-950/95"
					style={`left: ${tooltipLeft}%`}
				>
					<div class="space-y-1 text-[11px] tabular-nums">
						<div class="border-b border-gray-100 pb-1 text-[10px] text-gray-400 dark:border-neutral-800 dark:text-gray-500">{formatSampleTime(hoveredSample.sampledAtMs)}</div>
						{#if visibleSeries.memory}
							<div class="flex items-center justify-between gap-4"><span class="text-gray-500 dark:text-gray-400">Memory</span><span class="font-medium text-gray-950 dark:text-white">{hoveredSample.memoryPercent === null ? '—' : `${hoveredSample.memoryPercent.toFixed(1)}%`}</span></div>
						{/if}
						{#if visibleSeries.cpu}
							<div class="flex items-center justify-between gap-4"><span class="text-gray-500 dark:text-gray-400">CPU</span><span class="font-medium text-gray-950 dark:text-white">{hoveredSample.cpuPercent === null ? '—' : `${hoveredSample.cpuPercent.toFixed(1)}%`}</span></div>
						{/if}
						{#if visibleSeries.network}
							<div class="flex items-center justify-between gap-4"><span class="text-gray-500 dark:text-gray-400">Network</span><span class="font-medium text-gray-950 dark:text-white">{formatRate(hoveredSample.networkBytesPerSecond)}</span></div>
						{/if}
					</div>
				</div>
			{/if}
		</div>

		<div class="border-t border-gray-100 px-4 py-3 dark:border-neutral-800">
			<div class="mb-2 flex flex-wrap items-end justify-between gap-x-4 gap-y-1">
				<div>
					<p class="text-xs font-medium text-gray-700 dark:text-gray-300">Storage capacity</p>
					<p class="mt-0.5 text-[11px] tabular-nums text-gray-500 dark:text-gray-400">{storageValue}</p>
				</div>
				<div class="text-right">
					<p class="text-xs font-medium tabular-nums text-gray-700 dark:text-gray-300">{storageAvailable ? `${usedStoragePercent.toFixed(0)}% used` : 'Unavailable'}</p>
					{#if storageDetail}<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">{storageDetail}</p>{/if}
				</div>
			</div>
			<div
				class="h-2.5 overflow-hidden rounded-sm border border-gray-300 bg-gray-100 dark:border-neutral-700 dark:bg-neutral-800"
				role={storageAvailable ? 'progressbar' : undefined}
				aria-label={storageAvailable ? 'Storage used' : undefined}
				aria-valuemin={storageAvailable ? 0 : undefined}
				aria-valuemax={storageAvailable ? 100 : undefined}
				aria-valuenow={storageAvailable ? Math.round(usedStoragePercent) : undefined}
				data-storage-capacity
			>
				{#if storageAvailable}<div class={`h-full transition-[width] duration-300 motion-reduce:transition-none ${storageFillClass}`} style={`width: ${usedStoragePercent}%`}></div>{/if}
			</div>
		</div>
	</div>
</div>
