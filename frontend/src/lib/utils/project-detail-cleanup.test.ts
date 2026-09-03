import { describe, expect, it } from 'vitest';
import overview from '../../routes/projects/[id]/+page.svelte?raw';
import projectLayout from '../../routes/projects/[id]/+layout.svelte?raw';
import settings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import sourceSettings from '../../routes/projects/[id]/settings/source/+page.svelte?raw';
import resourceSettings from '../../routes/projects/[id]/settings/resources/+page.svelte?raw';
import dangerSettings from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';
import environmentRoute from '../../routes/projects/[id]/env/+page.svelte?raw';
import databaseLayout from '../../routes/projects/[id]/database/+layout.svelte?raw';
import deploymentsRoute from '../../routes/projects/[id]/deployments/+page.svelte?raw';
import logsRoute from '../../routes/projects/[id]/logs/+page.svelte?raw';
import deployControlPanel from '../components/DeployControlPanel.svelte?raw';
import effectiveConfiguration from '../components/ProjectEffectiveConfiguration.svelte?raw';
import projectDetailSidebar from '../components/ProjectDetailSidebar.svelte?raw';
import observability from '../components/ProjectObservability.svelte?raw';
import runtimeUsageBar from '../components/RuntimeUsageBar.svelte?raw';

describe('project detail cleanup contract', () => {
	it('keeps duplicate sibling navigation out of Overview', () => {
		expect(overview).not.toContain('EnvironmentVariablesDialog');
		expect(overview).not.toContain('api.env.list');
		expect(overview).not.toContain('api.dbStudio.status');
		expect(overview).not.toContain('selectPrimaryProjectMetric');
		expect(overview).not.toContain('Project settings');
		expect(overview).not.toContain('Database Studio');
	});

	it('shows lifecycle chrome only on Overview and Deployments', () => {
		expect(projectLayout).toContain('showOperationalHeader');
		expect(projectLayout).toContain('$page.url.pathname === projectBase');
		expect(projectLayout).toContain('$page.url.pathname === `${projectBase}/deployments`');
		expect(projectLayout).toContain('{#if showOperationalHeader}');
		expect(deployControlPanel).toContain('min-h-14');
		expect(deployControlPanel).not.toContain('publicProjectURL');
		expect(deployControlPanel).not.toContain('projectSummary');
		expect(deployControlPanel).not.toContain('>View logs</a>');
	});

	it('owns each configuration domain on one leaf route', () => {
		expect(projectLayout).toContain('ProjectDetailSidebar');
		expect(projectDetailSidebar).toContain("label: 'Environment'");
		expect(projectDetailSidebar).toContain('`${base}/env`');
		expect(environmentRoute).toContain('ProjectEnvironmentSettings');
		expect(environmentRoute).toContain('project-environment-leaf');
		expect(settings).not.toContain('ProjectEnvironmentSettings');
		expect(projectDetailSidebar).not.toContain('settings/environment');
		expect(effectiveConfiguration).not.toContain('>Source</p>');
		expect(effectiveConfiguration).not.toContain('sourceSummary');
	});

	it('uses Overview, Deployments, and Logs as the horizontal gutter reference', () => {
		expect(projectLayout).toContain('min-w-0 px-3.5 py-3');
		expect(deployControlPanel).toContain('px-4 py-2.5');
		expect(deploymentsRoute).toContain('TableShell');
		expect(logsRoute).toContain('SectionPanel');
		expect(environmentRoute).toContain('class="px-5 pt-4"');
		expect(environmentRoute).toContain('padding-inline: 1rem');
		expect(settings).toContain('padding-inline: 1.25rem');
		expect(settings).toContain('padding-inline: 1rem');
		expect(sourceSettings).toContain('padding-inline: 1.25rem');
		expect(sourceSettings).toContain('padding-inline: 1rem');
		expect(resourceSettings).toContain('padding-inline: 1.25rem');
		expect(resourceSettings).toContain('padding-inline: 1rem');
		expect(dangerSettings).toContain('padding-inline: 1.25rem');
		expect(dangerSettings).toContain('padding-inline: 1rem');
		expect(databaseLayout).toContain('px-5 pb-3 pt-4');
		expect(effectiveConfiguration).toContain('border-y border-[color:var(--workspace-divider)]');
		expect(effectiveConfiguration).not.toContain('rounded-lg');
	});

	it('keeps settings controls at sensible widths', () => {
		expect(sourceSettings).toContain('max-width: 36rem');
		expect(resourceSettings).toContain('max-width: 32rem');
		expect(dangerSettings).toContain('max-width: 36rem');
	});

	it('uses semantic resource color and readable overview charts', () => {
		expect(observability).toContain('tone="cpu"');
		expect(observability).toContain('tone="memory"');
		expect(observability).toContain('h-24');
		expect(runtimeUsageBar).toContain('var(--chart-cpu)');
		expect(runtimeUsageBar).toContain('var(--chart-memory)');
		expect(runtimeUsageBar).toContain('role="progressbar"');
		expect(runtimeUsageBar).not.toContain('Allocated resource');
	});

	it('keeps Database Studio a normal project leaf', () => {
		expect(databaseLayout).toContain('>Database Studio</h1>');
		expect(databaseLayout).toContain('Schema design');
		expect(databaseLayout).toContain('border-[color:var(--workspace-divider)]');
		expect(databaseLayout).not.toContain('min-h-12 items-center justify-between gap-3 border-b');
	});

	it('removes low-value duplicate status chrome from deployments and logs', () => {
		expect(deploymentsRoute).toContain('In-progress pipelines');
		expect(deploymentsRoute).not.toContain('Active pipeline');
		expect(deploymentsRoute).toContain('{#if showPagination}');
		expect(logsRoute).toContain('{#if logs.length > 0}');
		expect(logsRoute).toContain('No logs yet.');
		expect(logsRoute).not.toContain('Showing {filteredLogs.length} of {logs.length} lines');
	});
});
