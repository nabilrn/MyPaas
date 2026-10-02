import { describe, expect, it } from 'vitest';
import hostResourceOverview from './HostResourceOverview.svelte?raw';
import projectsPage from '../../routes/projects/+page.svelte?raw';

describe('host resource overview', () => {
	it('merges memory, cpu, and network into one interactive chart', () => {
		expect(projectsPage).toContain('HostResourceOverview');
		expect(projectsPage).toContain('samples={hostTelemetrySeries}');
		expect(hostResourceOverview).toContain("type SeriesKey = 'memory' | 'cpu' | 'network'");
		expect(hostResourceOverview).toContain("buildSeriesPaths('memory', chartSamples, networkScale)");
		expect(hostResourceOverview).toContain("buildSeriesPaths('cpu', chartSamples, networkScale)");
		expect(hostResourceOverview).toContain("buildSeriesPaths('network', chartSamples, networkScale)");
	});

	it('uses a chart-first desktop split with the metric summary in the right column', () => {
		expect(hostResourceOverview).toContain('data-host-resource-layout');
		expect(hostResourceOverview).toContain('xl:grid-cols-[minmax(0,3fr)_minmax(18rem,1fr)]');
		expect(hostResourceOverview).toContain('data-host-resource-summary');
		expect(hostResourceOverview).toContain('xl:border-l xl:border-t-0');
	});

	it('keeps compact visibility filters for all three time-series metrics', () => {
		expect(hostResourceOverview).toContain('aria-pressed={visibleSeries.memory}');
		expect(hostResourceOverview).toContain('aria-pressed={visibleSeries.cpu}');
		expect(hostResourceOverview).toContain('aria-pressed={visibleSeries.network}');
		expect(hostResourceOverview).toContain("toggleSeries('memory')");
		expect(hostResourceOverview).toContain("toggleSeries('cpu')");
		expect(hostResourceOverview).toContain("toggleSeries('network')");
	});

	it('warms up telemetry history before rendering the chart', () => {
		expect(hostResourceOverview).toContain('const minimumChartSamples = 8');
		expect(hostResourceOverview).toContain('chartReady = telemetryAvailable && chartSamples.length >= minimumChartSamples');
		expect(hostResourceOverview).toContain('data-host-history-loading');
		expect(hostResourceOverview).toContain('Collecting history');
		expect(hostResourceOverview).toContain('aria-valuemax={minimumChartSamples}');
		expect(hostResourceOverview).toContain('disabled={!chartReady}');
	});

	it('shows a stable unavailable state instead of an impossible warmup', () => {
		expect(projectsPage).toContain('telemetryAvailable={Boolean(hostStats.memory || hostStats.cpu || hostStats.network)}');
		expect(hostResourceOverview).toContain('data-host-history-unavailable');
		expect(hostResourceOverview).toContain('History unavailable');
		expect(hostResourceOverview).not.toContain('role="status"');
		expect(hostResourceOverview).not.toContain('aria-live="polite"');
	});

	it('uses subtle semantic area fills beneath the three telemetry lines', () => {
		expect(hostResourceOverview).toContain('id="host-memory-fill"');
		expect(hostResourceOverview).toContain('id="host-cpu-fill"');
		expect(hostResourceOverview).toContain('id="host-network-fill"');
		expect(hostResourceOverview).toContain('fill="url(#host-memory-fill)"');
		expect(hostResourceOverview).toContain('fill="url(#host-cpu-fill)"');
		expect(hostResourceOverview).toContain('fill="url(#host-network-fill)"');
	});

	it('keeps storage as a persistent capacity strip with explicit context', () => {
		expect(hostResourceOverview).toContain('data-storage-capacity');
		expect(hostResourceOverview).toContain("role={storageAvailable ? 'progressbar' : undefined}");
		expect(hostResourceOverview).toContain('Storage capacity');
		expect(hostResourceOverview).toContain('usedStoragePercent.toFixed(0)');
		expect(hostResourceOverview).toContain('h-2.5 overflow-hidden');
	});

	it('gives the merged chart explicit scale and time context', () => {
		expect(hostResourceOverview).toContain('Resource history');
		expect(hostResourceOverview).toContain('RAM/CPU 0–100% · Network adaptive scale');
		expect(hostResourceOverview).toContain('formatRate(networkScale.max)');
		expect(hostResourceOverview).toContain('formatRate(networkScale.midpoint)');
		expect(hostResourceOverview).toContain('formatRate(networkScale.min)');
		expect(hostResourceOverview).toContain('<span>Earlier</span>');
		expect(hostResourceOverview).toContain('<span>Now</span>');
		expect(hostResourceOverview).toContain('formatSampleTime(hoveredSample.sampledAtMs)');
		expect(hostResourceOverview).toContain('role="group" aria-label="Chart series visibility"');
		expect(hostResourceOverview).toContain('chartInsetX}px + (100% - ${chartInsetX * 2}px) * ${tooltipRatio}');
	});

	it('preserves hover and keyboard inspection without fabricating missing samples', () => {
		expect(hostResourceOverview).toContain('on:pointermove={handleChartPointer}');
		expect(hostResourceOverview).toContain('on:keydown={handleChartKeydown}');
		expect(hostResourceOverview).toContain("hoveredSample.memoryPercent === null ? '—'");
		expect(hostResourceOverview).toContain("hoveredSample.cpuPercent === null ? '—'");
		expect(hostResourceOverview).toContain('formatRate(hoveredSample.networkBytesPerSecond)');
	});

	it('renders isolated valid samples instead of dropping move-only SVG paths', () => {
		expect(hostResourceOverview).toContain("isolatedPoints('memory', chartSamples, networkScale)");
		expect(hostResourceOverview).toContain("isolatedPoints('cpu', chartSamples, networkScale)");
		expect(hostResourceOverview).toContain("isolatedPoints('network', chartSamples, networkScale)");
		expect(hostResourceOverview).toContain('memoryIsolatedPoints as point');
	});

	it('keeps rolling samples evenly spaced and breaks only on explicit missing values', () => {
		expect(hostResourceOverview).toContain('alignedChartSamples(samples)');
		expect(hostResourceOverview).toContain('activeSeries.every((series) => hasFiniteSeriesValue(sample, series))');
		expect(hostResourceOverview).toContain('(index / (sourceSamples.length - 1)) * chartWidth');
		expect(hostResourceOverview).toContain('if (value === null || !Number.isFinite(value))');
		expect(hostResourceOverview).not.toContain('deriveTelemetryCadenceMs');
		expect(hostResourceOverview).not.toContain('isTelemetryDiscontinuity');
		expect(hostResourceOverview).toContain('nearestSampleIndex(ratio, chartSamples)');
	});

	it('keeps chart visual weight thin and minimal', () => {
		expect(hostResourceOverview).toContain('stroke-width="1.35"');
		expect(hostResourceOverview).toContain('stroke-width="0.5"');
		expect(hostResourceOverview).toContain('stroke-width="0.6"');
		expect(hostResourceOverview).toContain('r="1.5" class={seriesClasses.');
		expect(hostResourceOverview).toContain('r="1.9" class={seriesClasses.');
		expect(hostResourceOverview).not.toContain('stroke-width="1.85"');
	});

	it('keeps 0% and 100% strokes inside the SVG viewport', () => {
		expect(hostResourceOverview).toContain('const chartPaddingY = 6');
		expect(hostResourceOverview).toContain('chartHeight - chartPaddingY');
		expect(hostResourceOverview).toContain('return percentageY(value)');
		expect(hostResourceOverview).toContain('y1={percentageY(75)}');
		expect(hostResourceOverview).toContain('y1={percentageY(50)}');
		expect(hostResourceOverview).toContain('y1={percentageY(25)}');
	});

	it('uses a bounded power scale for bursty network history', () => {
		expect(hostResourceOverview).toContain('deriveAdaptiveRateScale(networkValues)');
		expect(hostResourceOverview).toContain('linearRatio ** networkScale.exponent');
	});

	it('does not invent history gaps when the browser resumes or a poll fails', () => {
		expect(projectsPage).toContain('const hostTelemetryPollMs = 3000');
		expect(projectsPage).toContain('if (!document.hidden) void refreshDashboardData(true)');
		expect(projectsPage).toContain('setInterval(refreshHost, hostTelemetryPollMs)');
		expect(projectsPage).not.toContain('telemetryPauseThresholdMs');
		expect(projectsPage).not.toContain('markTelemetryPause');
		expect(projectsPage).not.toContain('recordTelemetryGap');
	});

	it('skips all-null host snapshots while preserving cumulative counter baselines', () => {
		expect(projectsPage).toContain('if (!hasCurrentTelemetry) return');
		expect(projectsPage).toContain('cpuPercent = deriveCPUUsage(cpuBaseline, current)');
		expect(projectsPage).toContain('networkRate = deriveNetworkRate(networkBaseline, current)');
		expect(projectsPage).toContain('currentCPUUsage = null');
		expect(projectsPage).toContain('currentNetworkRate = null');
		expect(projectsPage).not.toContain('hasCurrentTelemetry || hostTelemetrySeries.length > 0');
		expect(projectsPage).not.toContain('\t\t\tcpuBaseline = null;');
		expect(projectsPage).not.toContain('\t\t\tnetworkBaseline = null;');
	});
});
