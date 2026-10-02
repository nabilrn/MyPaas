<script lang="ts">
	import { deriveAdaptiveRateScale, type AdaptiveRateScale, type HostTelemetrySample } from '$lib/utils/host-telemetry';

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
	export let telemetryAvailable = true;
	export let samples: HostTelemetrySample[] = [];

	const chartWidth = 1000;
	const chartHeight = 220;
	const chartPaddingY = 6;
	const chartInsetX = 40;
	const curveTension = 0.68;
	const minimumChartSamples = 8;
	const seriesKeys: SeriesKey[] = ['memory', 'cpu', 'network'];

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
			const cp1y = clamp(
				p1.y + ((p2.y - p0.y) / 6) * curveTension,
				chartPaddingY,
				chartHeight - chartPaddingY
			);
			const cp2x = clamp(p2.x - ((p3.x - p1.x) / 6) * curveTension, 0, chartWidth);
			const cp2y = clamp(
				p2.y - ((p3.y - p1.y) / 6) * curveTension,
				chartPaddingY,
				chartHeight - chartPaddingY
			);

			path += ` C ${cp1x.toFixed(2)},${cp1y.toFixed(2)} ${cp2x.toFixed(2)},${cp2y.toFixed(2)} ${p2.x.toFixed(2)},${p2.y.toFixed(2)}`;
		}
		return path;
	}

	function buildAreaPath(input: ChartPoint[]) {
		if (input.length < 2) return '';
		const linePath = buildSmoothPath(input);
		const first = input[0];
		const last = input[input.length - 1];
		const baselineY = chartHeight - chartPaddingY;
		return `${linePath} L ${last.x.toFixed(2)},${baselineY} L ${first.x.toFixed(2)},${baselineY} Z`;
	}

	function seriesValue(sample: HostTelemetrySample, series: SeriesKey) {
		if (series === 'memory') return sample.memoryPercent;
		if (series === 'cpu') return sample.cpuPercent;
		return sample.networkBytesPerSecond;
	}

	function hasFiniteSeriesValue(sample: HostTelemetrySample, series: SeriesKey) {
		const value = seriesValue(sample, series);
		return value !== null && Number.isFinite(value);
	}

	function alignedChartSamples(sourceSamples: HostTelemetrySample[]) {
		const activeSeries = seriesKeys.filter((series) =>
			sourceSamples.filter((sample) => hasFiniteSeriesValue(sample, series)).length >= 2
		);
		if (activeSeries.length === 0) return sourceSamples;

		const alignedStart = sourceSamples.findIndex((sample) =>
			activeSeries.every((series) => hasFiniteSeriesValue(sample, series))
		);
		return alignedStart > 0 ? sourceSamples.slice(alignedStart) : sourceSamples;
	}

	function sampleX(index: number, sourceSamples: HostTelemetrySample[]) {
		if (sourceSamples.length <= 1) return chartWidth;
		return (index / (sourceSamples.length - 1)) * chartWidth;
	}

	function nearestSampleIndex(ratio: number, sourceSamples: HostTelemetrySample[]) {
		if (sourceSamples.length <= 1) return 0;
		const targetX = clamp(ratio, 0, 1) * chartWidth;
		let nearestIndex = 0;
		let nearestDistance = Number.POSITIVE_INFINITY;
		sourceSamples.forEach((_, index) => {
			const distance = Math.abs(sampleX(index, sourceSamples) - targetX);
			if (distance < nearestDistance) {
				nearestDistance = distance;
				nearestIndex = index;
			}
		});
		return nearestIndex;
	}

	function percentageY(value: number) {
		const usableHeight = chartHeight - chartPaddingY * 2;
		return chartPaddingY + (1 - clamp(value, 0, 100) / 100) * usableHeight;
	}

	function seriesY(value: number, series: SeriesKey, networkScale: AdaptiveRateScale) {
		const usableHeight = chartHeight - chartPaddingY * 2;
		if (series === 'network') {
			const span = Math.max(0.0001, networkScale.max - networkScale.min);
			const linearRatio = clamp((value - networkScale.min) / span, 0, 1);
			const normalized = linearRatio === 0 ? 0 : linearRatio ** networkScale.exponent;
			return chartPaddingY + (1 - normalized) * usableHeight;
		}
		return percentageY(value);
	}

	function buildSeriesPaths(
		series: SeriesKey,
		sourceSamples: HostTelemetrySample[],
		networkScale: AdaptiveRateScale
	) {
		const paths: { line: string; area: string }[] = [];
		let segment: ChartPoint[] = [];
		const flush = () => {
			if (segment.length > 0) {
				paths.push({
					line: buildSmoothPath(segment),
					area: buildAreaPath(segment)
				});
			}
			segment = [];
		};

		sourceSamples.forEach((sample, index) => {
			const value = seriesValue(sample, series);
			if (value === null || !Number.isFinite(value)) {
				flush();
				return;
			}
			const x = sampleX(index, sourceSamples);
			segment.push({ x, y: seriesY(value, series, networkScale) });
		});
		flush();
		return paths;
	}

	function pointFor(series: SeriesKey, index: number, sourceSamples: HostTelemetrySample[], networkScale: AdaptiveRateScale) {
		const sample = sourceSamples[index];
		if (!sample) return null;
		const value = seriesValue(sample, series);
		if (value === null || !Number.isFinite(value)) return null;
		const x = sampleX(index, sourceSamples);
		return { x, y: seriesY(value, series, networkScale) };
	}

	function isolatedPoints(
		series: SeriesKey,
		sourceSamples: HostTelemetrySample[],
		networkScale: AdaptiveRateScale
	) {
		return sourceSamples.flatMap((sample, index) => {
			const value = seriesValue(sample, series);
			if (value === null || !Number.isFinite(value)) return [];
			const previousSample = index > 0 ? sourceSamples[index - 1] : undefined;
			const nextSample = index < sourceSamples.length - 1 ? sourceSamples[index + 1] : undefined;
			const previous = previousSample ? seriesValue(previousSample, series) : null;
			const next = nextSample ? seriesValue(nextSample, series) : null;
			const hasPrevious = previous !== null && Number.isFinite(previous);
			const hasNext = next !== null && Number.isFinite(next);
			if (hasPrevious || hasNext) return [];
			const x = sampleX(index, sourceSamples);
			return [{ x, y: seriesY(value, series, networkScale) }];
		});
	}

	function handleChartPointer(event: PointerEvent) {
		if (!chartReady) return;
		const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect();
		const plotWidth = Math.max(1, bounds.width - chartInsetX * 2);
		const ratio = clamp((event.clientX - bounds.left - chartInsetX) / plotWidth, 0, 1);
		hoverIndex = nearestSampleIndex(ratio, chartSamples);
	}

	function handleChartKeydown(event: KeyboardEvent) {
		if (!chartReady || (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight')) return;
		event.preventDefault();
		const start = hoverIndex < 0 ? chartSamples.length - 1 : hoverIndex;
		hoverIndex = clamp(start + (event.key === 'ArrowRight' ? 1 : -1), 0, chartSamples.length - 1);
	}

	$: chartSamples = alignedChartSamples(samples);
	$: chartSampleCount = Math.min(chartSamples.length, minimumChartSamples);
	$: chartReady = telemetryAvailable && chartSamples.length >= minimumChartSamples;
	$: chartSampleProgress = (chartSampleCount / minimumChartSamples) * 100;
	$: networkValues = chartSamples
		.map((sample) => sample.networkBytesPerSecond)
		.filter((value): value is number => value !== null && Number.isFinite(value));
	$: networkScale = deriveAdaptiveRateScale(networkValues);
	$: memoryPaths = buildSeriesPaths('memory', chartSamples, networkScale);
	$: cpuPaths = buildSeriesPaths('cpu', chartSamples, networkScale);
	$: networkPaths = buildSeriesPaths('network', chartSamples, networkScale);
	$: memoryIsolatedPoints = isolatedPoints('memory', chartSamples, networkScale);
	$: cpuIsolatedPoints = isolatedPoints('cpu', chartSamples, networkScale);
	$: networkIsolatedPoints = isolatedPoints('network', chartSamples, networkScale);
	$: hoveredSample = hoverIndex >= 0 && hoverIndex < chartSamples.length ? chartSamples[hoverIndex] : null;
	$: hoveredX = hoverIndex >= 0 && chartSamples.length > 0 ? sampleX(hoverIndex, chartSamples) : null;
	$: tooltipRatio = hoveredX === null ? 0.5 : hoveredX / chartWidth;
	$: memoryHoverPoint = hoverIndex >= 0 ? pointFor('memory', hoverIndex, chartSamples, networkScale) : null;
	$: cpuHoverPoint = hoverIndex >= 0 ? pointFor('cpu', hoverIndex, chartSamples, networkScale) : null;
	$: networkHoverPoint = hoverIndex >= 0 ? pointFor('network', hoverIndex, chartSamples, networkScale) : null;
	$: storageAvailable = !/unavailable/i.test(`${storageValue} ${storageDetail}`);
	$: usedStoragePercent = clamp(Number.isFinite(storagePercent) ? storagePercent : 0, 0, 100);
	$: storageFillClass = usedStoragePercent >= 90
		? 'bg-red-500 dark:bg-red-400'
		: usedStoragePercent >= 80
			? 'bg-orange-500 dark:bg-orange-400'
			: 'bg-amber-400 dark:bg-amber-300';
</script>

<div class="bg-white dark:bg-neutral-900" data-host-resource-overview>
	<div class="grid grid-cols-1 xl:grid-cols-[minmax(0,3fr)_minmax(18rem,1fr)]" data-host-resource-layout>
		<section class="min-w-0">
			<div class="flex min-h-10 flex-wrap items-center justify-between gap-x-4 gap-y-1.5 px-4 py-2 text-[11px] text-gray-500 dark:text-gray-400">
				<div class="min-w-0">
					<p class="text-[12px] font-semibold leading-4 text-gray-800 dark:text-gray-200">Resource history</p>
					<p class="text-[10px] leading-4 text-gray-400 dark:text-gray-500">RAM/CPU 0–100% · Network adaptive scale</p>
				</div>
				<div class="flex items-center gap-2.5" role="group" aria-label="Chart series visibility">
					<button
						type="button"
						class="app-focus inline-flex items-center gap-1.5 rounded-sm px-1 py-0.5 transition-opacity disabled:cursor-default disabled:opacity-40"
						class:opacity-40={!visibleSeries.memory}
						disabled={!chartReady}
						aria-pressed={visibleSeries.memory}
						on:click={() => toggleSeries('memory')}
					>
						<span class={`h-1.5 w-1.5 rounded-full ${seriesClasses.memory.dot}`}></span>
						<span>Memory</span>
					</button>
					<button
						type="button"
						class="app-focus inline-flex items-center gap-1.5 rounded-sm px-1 py-0.5 transition-opacity disabled:cursor-default disabled:opacity-40"
						class:opacity-40={!visibleSeries.cpu}
						disabled={!chartReady}
						aria-pressed={visibleSeries.cpu}
						on:click={() => toggleSeries('cpu')}
					>
						<span class={`h-1.5 w-1.5 rounded-full ${seriesClasses.cpu.dot}`}></span>
						<span>CPU</span>
					</button>
					<button
						type="button"
						class="app-focus inline-flex items-center gap-1.5 rounded-sm px-1 py-0.5 transition-opacity disabled:cursor-default disabled:opacity-40"
						class:opacity-40={!visibleSeries.network}
						disabled={!chartReady}
						aria-pressed={visibleSeries.network}
						on:click={() => toggleSeries('network')}
						title="Network history uses an adaptive rate scale"
					>
						<span class={`h-1.5 w-1.5 rounded-full ${seriesClasses.network.dot}`}></span>
						<span>Network</span>
					</button>
				</div>
			</div>

			{#if chartReady}
				<div
					class="app-focus relative h-64 overflow-hidden outline-none"
					role="img"
					aria-label="Host resource history. Memory and CPU use percentage scale; network uses an adaptive rate scale. Use the series controls to hide or show lines."
					tabindex="0"
					on:pointermove={handleChartPointer}
					on:pointerleave={() => (hoverIndex = -1)}
					on:focus={() => {
						if (hoverIndex < 0) hoverIndex = chartSamples.length - 1;
					}}
					on:blur={() => (hoverIndex = -1)}
					on:keydown={handleChartKeydown}
				>
					<div class="pointer-events-none absolute bottom-5 left-2 top-2 z-[1] flex flex-col justify-between text-[9px] tabular-nums text-gray-400 dark:text-gray-500" aria-hidden="true">
						<span>100%</span>
						<span>50%</span>
						<span>0%</span>
					</div>
					<div class="pointer-events-none absolute bottom-5 right-2 top-2 z-[1] flex flex-col items-end justify-between text-[9px] tabular-nums text-violet-500/70 dark:text-violet-300/60" aria-hidden="true">
						<span>{formatRate(networkScale.max)}</span>
						<span>{formatRate(networkScale.midpoint)}</span>
						<span>{formatRate(networkScale.min)}</span>
					</div>
					<div class="pointer-events-none absolute bottom-1.5 left-10 right-10 z-[1] flex justify-between text-[9px] text-gray-400 dark:text-gray-500" aria-hidden="true">
						<span>Earlier</span>
						<span>Now</span>
					</div>

					<div class="pointer-events-none absolute bottom-5 left-10 right-10 top-2">
						<svg class="h-full w-full" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none" aria-hidden="true">
							<defs>
								<linearGradient id="host-memory-fill" x1="0" x2="0" y1="0" y2="1">
									<stop offset="0%" stop-color="#34d399" stop-opacity="0.22" />
									<stop offset="100%" stop-color="#34d399" stop-opacity="0.015" />
								</linearGradient>
								<linearGradient id="host-cpu-fill" x1="0" x2="0" y1="0" y2="1">
									<stop offset="0%" stop-color="#38bdf8" stop-opacity="0.20" />
									<stop offset="100%" stop-color="#38bdf8" stop-opacity="0.015" />
								</linearGradient>
								<linearGradient id="host-network-fill" x1="0" x2="0" y1="0" y2="1">
									<stop offset="0%" stop-color="#a78bfa" stop-opacity="0.20" />
									<stop offset="100%" stop-color="#a78bfa" stop-opacity="0.015" />
								</linearGradient>
							</defs>

							<g class="stroke-gray-200/45 dark:stroke-neutral-700/40" stroke-width="0.5">
								<line x1={chartWidth * 0.2} x2={chartWidth * 0.2} y1="0" y2={chartHeight} />
								<line x1={chartWidth * 0.4} x2={chartWidth * 0.4} y1="0" y2={chartHeight} />
								<line x1={chartWidth * 0.6} x2={chartWidth * 0.6} y1="0" y2={chartHeight} />
								<line x1={chartWidth * 0.8} x2={chartWidth * 0.8} y1="0" y2={chartHeight} />
								<line x1="0" x2={chartWidth} y1={percentageY(75)} y2={percentageY(75)} />
								<line x1="0" x2={chartWidth} y1={percentageY(50)} y2={percentageY(50)} />
								<line x1="0" x2={chartWidth} y1={percentageY(25)} y2={percentageY(25)} />
							</g>

							{#if visibleSeries.memory}
								{#each memoryPaths as path}
									{#if path.area}<path d={path.area} fill="url(#host-memory-fill)" stroke="none" />{/if}
									<path d={path.line} fill="none" class={seriesClasses.memory.stroke} stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
								{/each}
								{#each memoryIsolatedPoints as point}<circle cx={point.x} cy={point.y} r="1.5" class={seriesClasses.memory.point} />{/each}
							{/if}
							{#if visibleSeries.cpu}
								{#each cpuPaths as path}
									{#if path.area}<path d={path.area} fill="url(#host-cpu-fill)" stroke="none" />{/if}
									<path d={path.line} fill="none" class={seriesClasses.cpu.stroke} stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
								{/each}
								{#each cpuIsolatedPoints as point}<circle cx={point.x} cy={point.y} r="1.5" class={seriesClasses.cpu.point} />{/each}
							{/if}
							{#if visibleSeries.network}
								{#each networkPaths as path}
									{#if path.area}<path d={path.area} fill="url(#host-network-fill)" stroke="none" />{/if}
									<path d={path.line} fill="none" class={seriesClasses.network.stroke} stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
								{/each}
								{#each networkIsolatedPoints as point}<circle cx={point.x} cy={point.y} r="1.5" class={seriesClasses.network.point} />{/each}
							{/if}

							{#if hoveredX !== null}
								<line x1={hoveredX} x2={hoveredX} y1="0" y2={chartHeight} class="stroke-gray-500/30 dark:stroke-gray-400/30" stroke-width="0.6" vector-effect="non-scaling-stroke" />
								{#if visibleSeries.memory && memoryHoverPoint}<circle cx={memoryHoverPoint.x} cy={memoryHoverPoint.y} r="1.9" class={seriesClasses.memory.point} />{/if}
								{#if visibleSeries.cpu && cpuHoverPoint}<circle cx={cpuHoverPoint.x} cy={cpuHoverPoint.y} r="1.9" class={seriesClasses.cpu.point} />{/if}
								{#if visibleSeries.network && networkHoverPoint}<circle cx={networkHoverPoint.x} cy={networkHoverPoint.y} r="1.9" class={seriesClasses.network.point} />{/if}
							{/if}
						</svg>
					</div>

					{#if hoveredSample}
						<div
							class="pointer-events-none absolute top-3 z-10 -translate-x-1/2 rounded-md border border-gray-200 bg-white/95 px-2.5 py-2 shadow-sm backdrop-blur dark:border-neutral-700 dark:bg-neutral-950/95"
							style={`left: clamp(64px, calc(${chartInsetX}px + (100% - ${chartInsetX * 2}px) * ${tooltipRatio}), calc(100% - 64px))`}
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
			{:else if telemetryAvailable}
				<div
					class="relative h-64 overflow-hidden"
					data-host-history-loading
				>
					<div class="pointer-events-none absolute bottom-5 left-10 right-10 top-2" aria-hidden="true">
						<svg class="h-full w-full" viewBox={`0 0 ${chartWidth} ${chartHeight}`} preserveAspectRatio="none">
							<g class="stroke-gray-200/45 dark:stroke-neutral-700/40" stroke-width="0.5">
								<line x1={chartWidth * 0.2} x2={chartWidth * 0.2} y1="0" y2={chartHeight} />
								<line x1={chartWidth * 0.4} x2={chartWidth * 0.4} y1="0" y2={chartHeight} />
								<line x1={chartWidth * 0.6} x2={chartWidth * 0.6} y1="0" y2={chartHeight} />
								<line x1={chartWidth * 0.8} x2={chartWidth * 0.8} y1="0" y2={chartHeight} />
								<line x1="0" x2={chartWidth} y1={chartHeight * 0.25} y2={chartHeight * 0.25} />
								<line x1="0" x2={chartWidth} y1={chartHeight * 0.5} y2={chartHeight * 0.5} />
								<line x1="0" x2={chartWidth} y1={chartHeight * 0.75} y2={chartHeight * 0.75} />
							</g>
						</svg>
					</div>
					<div class="absolute inset-0 flex items-center justify-center px-6">
						<div class="w-full max-w-52 rounded-md border border-gray-200/80 bg-white/90 px-3 py-2.5 shadow-sm backdrop-blur dark:border-neutral-700 dark:bg-neutral-900/90">
							<div class="flex items-center justify-between gap-3 text-[10px] font-medium text-gray-600 dark:text-gray-300">
								<span>Collecting history</span>
								<span class="tabular-nums text-gray-400 dark:text-gray-500">{chartSampleCount}/{minimumChartSamples}</span>
							</div>
							<div
								class="mt-2 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800"
								role="progressbar"
								aria-label="Telemetry history samples collected"
								aria-valuemin="0"
								aria-valuemax={minimumChartSamples}
								aria-valuenow={chartSampleCount}
							>
								<div class="h-full bg-gray-400 transition-[width] duration-300 dark:bg-gray-500" style={`width: ${chartSampleProgress}%`}></div>
							</div>
							<p class="mt-1.5 text-[9px] leading-3.5 text-gray-400 dark:text-gray-500">Chart appears after enough samples are collected.</p>
						</div>
					</div>
				</div>
			{:else}
				<div class="relative flex h-64 items-center justify-center px-6" data-host-history-unavailable>
					<div class="max-w-xs text-center">
						<p class="text-[11px] font-medium text-gray-600 dark:text-gray-300">History unavailable</p>
						<p class="mt-1 text-[10px] leading-4 text-gray-400 dark:text-gray-500">Live CPU, memory, and network telemetry is not available from mypaas-statd.</p>
					</div>
				</div>
			{/if}

			<div class="border-t border-gray-100 px-4 py-3 dark:border-neutral-800">
				<div class="mb-2 flex flex-wrap items-end justify-between gap-x-4 gap-y-1">
					<div>
						<p class="text-[11px] font-medium leading-4 text-gray-700 dark:text-gray-300">Storage capacity</p>
						<p class="text-[10px] leading-4 tabular-nums text-gray-500 dark:text-gray-400">{storageValue}</p>
					</div>
					<div class="text-right">
						<p class="text-[11px] font-medium leading-4 tabular-nums text-gray-700 dark:text-gray-300">{storageAvailable ? `${usedStoragePercent.toFixed(0)}% used` : 'Unavailable'}</p>
						{#if storageDetail}<p class="text-[10px] leading-4 text-gray-500 dark:text-gray-400">{storageDetail}</p>{/if}
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
		</section>

		<aside class="border-t border-gray-100 dark:border-neutral-800 xl:border-l xl:border-t-0" data-host-resource-summary>
			<div class="px-4 py-4">
				<div class="flex items-center justify-between gap-3">
					<div class="flex min-w-0 items-center gap-2">
						<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-400/80 dark:bg-emerald-300/75"></span>
						<p class="metric-label truncate">{memoryLabel}</p>
					</div>
					{#if memoryIndicator}<span class="metric-value shrink-0 text-[10px] font-medium text-gray-500 dark:text-gray-400">{memoryIndicator}</span>{/if}
				</div>
				<p class="metric-value mt-1.5 truncate text-[17px] font-semibold tracking-tight text-gray-950 dark:text-white">{memoryValue}</p>
				{#if memoryDetail}<p class="mt-1 truncate text-[10px] text-gray-500 dark:text-gray-400" title={memoryDetail}>{memoryDetail}</p>{/if}
			</div>

			<div class="border-t border-gray-100 px-4 py-4 dark:border-neutral-800">
				<div class="flex items-center justify-between gap-3">
					<div class="flex min-w-0 items-center gap-2">
						<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-sky-400/80 dark:bg-sky-300/75"></span>
						<p class="metric-label truncate">CPU usage</p>
					</div>
					{#if cpuIndicator}<span class="metric-value shrink-0 text-[10px] font-medium text-gray-500 dark:text-gray-400">{cpuIndicator}</span>{/if}
				</div>
				<p class="metric-value mt-1.5 truncate text-[17px] font-semibold tracking-tight text-gray-950 dark:text-white">{cpuValue}</p>
				{#if cpuDetail}<p class="mt-1 text-[10px] leading-4 text-gray-500 dark:text-gray-400" title={cpuDetail}>{cpuDetail}</p>{/if}
			</div>

			<div class="border-t border-gray-100 px-4 py-4 dark:border-neutral-800">
				<div class="flex items-center justify-between gap-3">
					<div class="flex min-w-0 items-center gap-2">
						<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-violet-400/80 dark:bg-violet-300/75"></span>
						<p class="metric-label truncate">Network</p>
					</div>
					{#if networkIndicator}<span class="metric-value shrink-0 text-[10px] font-medium text-gray-500 dark:text-gray-400">{networkIndicator}</span>{/if}
				</div>
				<p class="metric-value mt-1.5 truncate text-[17px] font-semibold tracking-tight text-gray-950 dark:text-white">{networkValue}</p>
				{#if networkDetail}<p class="mt-1 truncate text-[10px] text-gray-500 dark:text-gray-400" title={networkDetail}>{networkDetail}</p>{/if}
			</div>

			<div class="border-t border-gray-100 px-4 py-4 dark:border-neutral-800">
				<div class="flex items-center justify-between gap-3">
					<div class="flex min-w-0 items-center gap-2">
						<span class="h-1.5 w-1.5 shrink-0 rounded-full bg-amber-400/80 dark:bg-amber-300/75"></span>
						<p class="metric-label truncate">Storage</p>
					</div>
					{#if storageIndicator}<span class="metric-value shrink-0 text-[10px] font-medium text-gray-500 dark:text-gray-400">{storageIndicator}</span>{/if}
				</div>
				<p class="metric-value mt-1.5 truncate text-[17px] font-semibold tracking-tight text-gray-950 dark:text-white">{storageValue}</p>
				{#if storageDetail}<p class="mt-1 truncate text-[10px] text-gray-500 dark:text-gray-400" title={storageDetail}>{storageDetail}</p>{/if}
			</div>
		</aside>
	</div>
</div>

