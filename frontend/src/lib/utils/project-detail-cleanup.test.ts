import { describe, expect, it } from 'vitest';
import overview from '../../routes/projects/[id]/+page.svelte?raw';
import projectLayout from '../../routes/projects/[id]/+layout.svelte?raw';
import settings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import environmentRoute from '../../routes/projects/[id]/env/+page.svelte?raw';
import projectDetailSidebar from '../components/ProjectDetailSidebar.svelte?raw';
import observability from '../components/ProjectObservability.svelte?raw';
import runtimeUsageBar from '../components/RuntimeUsageBar.svelte?raw';

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

	it('keeps project summary copy and navigation concise', () => {
		expect(overview).toContain('Your project settings and configuration.');
		expect(overview).toContain('Browse and manage your project database.');
		expect(overview).toContain('ArrowUpRight');
		expect(overview).toContain('variant="secondary" size="xs"');
		expect(overview).toContain('border-[color:var(--workspace-divider)]');
	});

	it('owns environment management on the project env route instead of settings', () => {
		expect(projectLayout).toContain('ProjectDetailSidebar');
		expect(projectDetailSidebar).toContain("label: 'Environment'");
		expect(projectDetailSidebar).toContain('`${base}/env`');
		expect(environmentRoute).toContain('ProjectEnvironmentSettings');
		expect(environmentRoute).toContain('projectId={$page.params.id');
		expect(settings).not.toContain('ProjectEnvironmentSettings');
		expect(projectDetailSidebar).not.toContain('settings/environment');
	});

	it('shows runtime as bounded low-contrast allocation bars instead of line charts', () => {
		expect(observability).toContain('projectResourceAllocation(project, visibleItems)');
		expect(observability).toContain('RuntimeUsageBar');
		expect(observability).toContain('used={cpuUsed}');
		expect(observability).toContain('limit={cpuLimit}');
		expect(observability).toContain('used={memoryUsed}');
		expect(observability).toContain('limit={memoryLimit}');
		expect(observability).not.toContain('MultiServiceMetricChart');
		expect(runtimeUsageBar).toContain('role="progressbar"');
		expect(runtimeUsageBar).toContain('bg-gray-300/70');
		expect(runtimeUsageBar).toContain('bg-transparent');
		expect(runtimeUsageBar).not.toContain('bg-sky');
	});
});
