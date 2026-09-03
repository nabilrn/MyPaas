import { describe, expect, it } from 'vitest';
import overview from '../../routes/projects/[id]/+page.svelte?raw';
import settings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import observability from '../components/ProjectObservability.svelte?raw';
import metricChart from '../components/MultiServiceMetricChart.svelte?raw';

describe('project detail cleanup contract', () => {
	it('keeps configuration detail out of the project overview', () => {
		expect(overview).not.toContain('EnvironmentVariablesDialog');
		expect(overview).not.toContain('api.env.list');
		expect(overview).not.toContain('api.dbStudio.status');
		expect(overview).not.toContain('selectPrimaryProjectMetric');
		expect(overview).toContain('Settings');
		expect(overview).toContain("project.deployMode !== 'static'");
		expect(overview).toContain('Database Studio');
	});

	it('owns environment management inside the settings sub-navigation', () => {
		expect(settings).toContain("'environment'");
		expect(settings).toContain('KeyRound');
		expect(settings).toContain('ProjectEnvironmentSettings');
		expect(settings).toContain("activeSection === 'environment'");
		expect(settings).toContain('projectId={project.id}');
	});

	it('bounds runtime charts by assigned resources instead of observed samples', () => {
		expect(observability).toContain('projectResourceScale(project, visibleItems)');
		expect(observability).toContain('maxValue={resourceScale.cpuPercent}');
		expect(observability).toContain('maxValue={resourceScale.memoryMb}');
		expect(metricChart).toContain("rangeCaption = maxValue !== null ? 'allocated scale'");
		expect(metricChart).toContain('`0–${formatRangeValue(domain.max)}${suffix}`');
	});
});
