import { describe, expect, it } from 'vitest';
import overview from '../../routes/projects/[id]/+page.svelte?raw';
import projectLayout from '../../routes/projects/[id]/+layout.svelte?raw';
import settings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import webhookSettings from '../../routes/projects/[id]/settings/webhook/+page.svelte?raw';
import sourceRedirect from '../../routes/projects/[id]/settings/source/+page.ts?raw';
import resourceRedirect from '../../routes/projects/[id]/settings/resources/+page.ts?raw';
import dangerSettings from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';
import environmentRoute from '../../routes/projects/[id]/env/+page.svelte?raw';
import databaseLayout from '../../routes/projects/[id]/database/+layout.svelte?raw';
import deploymentsRoute from '../../routes/projects/[id]/deployments/+page.svelte?raw';
import logsRoute from '../../routes/projects/[id]/logs/+page.svelte?raw';
import deployControlPanel from '../components/DeployControlPanel.svelte?raw';
import effectiveConfiguration from '../components/ProjectEffectiveConfiguration.svelte?raw';
import projectDetailSidebar from '../components/ProjectDetailSidebar.svelte?raw';
import combinedSettings from '../components/ProjectCombinedSettings.svelte?raw';
import observability from '../components/ProjectObservability.svelte?raw';
import runtimeUsageBar from '../components/RuntimeUsageBar.svelte?raw';
import settingsWorkspace from '../components/SettingsWorkspace.svelte?raw';

describe('project detail cleanup contract', () => {
	it('keeps duplicate sibling navigation out of Overview', () => {
		expect(overview).not.toContain('EnvironmentVariablesDialog');
		expect(overview).not.toContain('api.env.list');
		expect(overview).not.toContain('api.dbStudio.status');
		expect(overview).not.toContain('selectPrimaryProjectMetric');
		expect(overview).not.toContain('Project settings');
		expect(overview).not.toContain('Database Studio');
	});

	it('keeps project identity, application URL, and lifecycle actions persistent across project leaves', () => {
		expect(projectLayout).not.toContain('showOperationalHeader');
		expect(projectLayout).toContain('projectURL');
		expect(projectLayout).toContain('<DeployControlPanel');
		expect(projectLayout).toContain('publicUrl={effectivePublicURL}');
		expect(deployControlPanel).toContain("export let publicUrl = ''");
		expect(deployControlPanel).toContain('ExternalLink');
		expect(deployControlPanel).toContain('publicUrl.replace');
		expect(deployControlPanel).toContain('min-h-14');
	});

	it('keeps Environment dedicated while making Webhook a first-class configuration leaf', () => {
		expect(projectLayout).toContain('ProjectDetailSidebar');
		expect(projectDetailSidebar).toContain("label: 'Environment'");
		expect(projectDetailSidebar).toContain('`${base}/env`');
		expect(projectDetailSidebar).toContain("label: 'Settings'");
		expect(projectDetailSidebar).toContain("label: 'Webhook'");
		expect(projectDetailSidebar).toContain('`${base}/settings/webhook`');
		expect(environmentRoute).toContain('ProjectEnvironmentSettings');
		expect(combinedSettings).not.toContain('ProjectEnvironmentSettings');
		expect(combinedSettings).not.toContain('api.projects.regenerateWebhookSecret');
		expect(settings).toContain('ProjectCombinedSettings');
		expect(webhookSettings).toContain('ProjectWebhookSettings');
		expect(sourceRedirect).toContain("redirect(307, `/projects/${params.id}/settings`)");
		expect(resourceRedirect).toContain("redirect(307, `/projects/${params.id}/settings`)");
		expect(effectiveConfiguration).not.toContain('>Source</p>');
	});

	it('uses Overview, Deployments, and Logs as the horizontal gutter reference', () => {
		expect(projectLayout).toContain('min-w-0 px-3.5 py-3');
		expect(deployControlPanel).toContain('px-4 py-2.5');
		expect(deploymentsRoute).toContain('TableShell');
		expect(logsRoute).toContain('SectionPanel');
		expect(environmentRoute).toContain('class="px-5 pt-4"');
		expect(settings).toContain('SettingsWorkspace');
		expect(webhookSettings).toContain('SettingsWorkspace');
		expect(dangerSettings).toContain('SettingsWorkspace');
		expect(settingsWorkspace).toContain('width: 100%');
		expect(settingsWorkspace).not.toContain('max-width: 64rem');
		expect(databaseLayout).toContain('px-5 pb-3 pt-4');
		expect(effectiveConfiguration).toContain('border-y border-[color:var(--workspace-divider)]');
		expect(effectiveConfiguration).not.toContain('rounded-lg');
	});

	it('keeps settings controls aligned inside the full-width two-column structure', () => {
		expect(combinedSettings).toContain('lg:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]');
		expect(combinedSettings).toContain('sm:grid-cols-2');
		expect(combinedSettings).toContain('grid-cols-[7rem_minmax(0,1fr)]');
		expect(combinedSettings).not.toContain('max-w-xl');
		expect(combinedSettings).not.toContain('max-w-md');
	});

	it('uses semantic resource color and readable overview charts', () => {
		expect(observability).toContain('tone="cpu"');
		expect(observability).toContain('tone="memory"');
		expect(observability).toContain('h-24');
		expect(runtimeUsageBar).toContain('var(--chart-cpu)');
		expect(runtimeUsageBar).toContain('var(--chart-memory)');
		expect(runtimeUsageBar).toContain('role="progressbar"');
	});

	it('keeps Database Studio a normal project leaf', () => {
		expect(databaseLayout).toContain('>Database Studio</h1>');
		expect(databaseLayout).toContain('Schema design');
		expect(databaseLayout).toContain('border-[color:var(--workspace-divider)]');
		expect(databaseLayout).not.toContain('min-h-12 items-center justify-between gap-3 border-b');
	});

	it('removes low-value duplicate status chrome from deployments and logs', () => {
		expect(deploymentsRoute).toContain('>In progress</p>');
		expect(deploymentsRoute).not.toContain('Active pipeline');
		expect(deploymentsRoute).toContain('{#if showPagination}');
		expect(deploymentsRoute).toContain('scrollToLatest');
		expect(logsRoute).toContain('{#if logs.length > 0}');
		expect(logsRoute).toContain('No logs yet.');
	});
});