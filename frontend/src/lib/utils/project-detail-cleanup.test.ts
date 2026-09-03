import { describe, expect, it } from 'vitest';
import overview from '../../routes/projects/[id]/+page.svelte?raw';
import projectLayout from '../../routes/projects/[id]/+layout.svelte?raw';
import settings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import environmentRoute from '../../routes/projects/[id]/env/+page.svelte?raw';
import deploymentsRoute from '../../routes/projects/[id]/deployments/+page.svelte?raw';
import logsRoute from '../../routes/projects/[id]/logs/+page.svelte?raw';
import deployControlPanel from '../components/DeployControlPanel.svelte?raw';
import projectDetailSidebar from '../components/ProjectDetailSidebar.svelte?raw';
import observability from '../components/ProjectObservability.svelte?raw';
import runtimeUsageBar from '../components/RuntimeUsageBar.svelte?raw';

describe('project detail cleanup contract', () => {
	it('keeps configuration detail and duplicate settings navigation out of the overview', () => {
		expect(overview).not.toContain('EnvironmentVariablesDialog');
		expect(overview).not.toContain('api.env.list');
		expect(overview).not.toContain('api.dbStudio.status');
		expect(overview).not.toContain('selectPrimaryProjectMetric');
		expect(overview).not.toContain('Project settings');
		expect(overview).not.toContain('Your project settings and configuration.');
		expect(overview).toContain("project.deployMode !== 'static'");
		expect(overview).toContain('Database Studio');
	});

	it('uses one compact project action bar on every project detail route', () => {
		expect(projectLayout).toContain('DeployControlPanel');
		expect(projectLayout).not.toContain('databaseWorkspace');
		expect(deployControlPanel).toContain('min-h-14');
		expect(deployControlPanel).toContain('{project.name}');
		expect(deployControlPanel).not.toContain('publicProjectURL');
		expect(deployControlPanel).not.toContain('projectSummary');
		expect(deployControlPanel).not.toContain('>View logs</a>');
	});

	it('owns environment management on the project env route instead of settings', () => {
		expect(projectLayout).toContain('ProjectDetailSidebar');
		expect(projectDetailSidebar).toContain("label: 'Environment'");
		expect(projectDetailSidebar).toContain('`${base}/env`');
		expect(environmentRoute).toContain('ProjectEnvironmentSettings');
		expect(environmentRoute).toContain('projectId={$page.params.id');
		expect(environmentRoute).toContain('>Environment</h1>');
		expect(environmentRoute).toContain('Variables available to your app at runtime.');
		expect(settings).not.toContain('ProjectEnvironmentSettings');
		expect(projectDetailSidebar).not.toContain('settings/environment');
	});

	it('shows runtime usage once against a clear allocation', () => {
		expect(observability).toContain('projectResourceAllocation(project, visibleItems)');
		expect(observability).toContain('RuntimeUsageBar');
		expect(observability).toContain('used={cpuUsed}');
		expect(observability).toContain('limit={cpuLimit}');
		expect(observability).toContain('used={memoryUsed}');
		expect(observability).toContain('limit={memoryLimit}');
		expect(observability).toContain('formatCPU(cpuUsed)} / ${formatCPU(cpuLimit)');
		expect(observability).toContain('formatMemory(memoryUsed)} / ${formatMemory(memoryLimit)');
		expect(observability).not.toContain('% of ${formatPercent');
		expect(observability).not.toContain('allocationLabel=');
		expect(runtimeUsageBar).toContain('role="progressbar"');
		expect(runtimeUsageBar).toContain('bg-gray-300/70');
		expect(runtimeUsageBar).toContain('bg-transparent');
		expect(runtimeUsageBar).not.toContain('Allocated resource');
		expect(runtimeUsageBar).not.toContain('bg-sky');
	});

	it('removes low-value duplicate status controls from deployments and logs', () => {
		expect(deploymentsRoute).toContain('In-progress pipelines');
		expect(deploymentsRoute).not.toContain('Active pipeline');
		expect(deploymentsRoute).toContain('{#if showPagination}');
		expect(logsRoute).toContain('{filteredLogs.length} visible');
		expect(logsRoute).toContain('Latest {maxLines} lines kept in memory.');
		expect(logsRoute).not.toContain('Showing {filteredLogs.length} of {logs.length} lines');
		expect(logsRoute).not.toContain('text-sky-300');
	});
});
