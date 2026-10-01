import { describe, expect, it } from 'vitest';
import hostResourceOverview from './HostResourceOverview.svelte?raw';
import projectsPage from '../../routes/projects/+page.svelte?raw';

describe('host resource overview', () => {
	it('merges memory, cpu, and network into one interactive chart', () => {
		expect(projectsPage).toContain('HostResourceOverview');
		expect(projectsPage).toContain('samples={hostTelemetrySeries}');
		expect(hostResourceOverview).toContain("type SeriesKey = 'memory' | 'cpu' | 'network'");
		expect(hostResourceOverview).toContain("buildSeriesPaths('memory', samples, networkDomain)");
		expect(hostResourceOverview).toContain("buildSeriesPaths('cpu', samples, networkDomain)");
		expect(hostResourceOverview).toContain("buildSeriesPaths('network', samples, networkDomain)");
	});

	it('keeps compact visibility filters for all three time-series metrics', () => {
		expect(hostResourceOverview).toContain('aria-pressed={visibleSeries.memory}');
		expect(hostResourceOverview).toContain('aria-pressed={visibleSeries.cpu}');
		expect(hostResourceOverview).toContain('aria-pressed={visibleSeries.network}');
		expect(hostResourceOverview).toContain("toggleSeries('memory')");
		expect(hostResourceOverview).toContain("toggleSeries('cpu')");
		expect(hostResourceOverview).toContain("toggleSeries('network')");
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
		expect(hostResourceOverview).toContain('RAM and CPU use a 0–100% scale');
		expect(hostResourceOverview).toContain('formatRate(networkDomain.max)');
		expect(hostResourceOverview).toContain('formatRate(networkDomain.min)');
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
		expect(hostResourceOverview).toContain("isolatedPoints('memory', samples, networkDomain)");
		expect(hostResourceOverview).toContain("isolatedPoints('cpu', samples, networkDomain)");
		expect(hostResourceOverview).toContain("isolatedPoints('network', samples, networkDomain)");
		expect(hostResourceOverview).toContain('memoryIsolatedPoints as point');
	});

	it('records successful all-null polls as gaps once history exists', () => {
		expect(projectsPage).toContain('hasCurrentTelemetry || hostTelemetrySeries.length > 0');
		expect(projectsPage).toContain('networkBytesPerSecond: currentNetworkRate?.totalBytesPerSecond ?? null');
	});
});
