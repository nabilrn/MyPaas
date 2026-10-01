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

	it('keeps storage as a persistent horizontal capacity bar below the chart', () => {
		expect(hostResourceOverview).toContain('data-storage-capacity');
		expect(hostResourceOverview).toContain("role={storageAvailable ? 'progressbar' : undefined}");
		expect(hostResourceOverview).toContain('h-3 overflow-hidden');
	});

	it('preserves hover and keyboard inspection without fabricating missing samples', () => {
		expect(hostResourceOverview).toContain('on:pointermove={handleChartPointer}');
		expect(hostResourceOverview).toContain('on:keydown={handleChartKeydown}');
		expect(hostResourceOverview).toContain("hoveredSample.memoryPercent === null ? '—'");
		expect(hostResourceOverview).toContain("hoveredSample.cpuPercent === null ? '—'");
		expect(hostResourceOverview).toContain('formatRate(hoveredSample.networkBytesPerSecond)');
	});
});
