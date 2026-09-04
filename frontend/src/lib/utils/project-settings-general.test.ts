import { describe, expect, it } from 'vitest';
import combinedSettings from '../components/ProjectCombinedSettings.svelte?raw';
import settingsWorkspace from '../components/SettingsWorkspace.svelte?raw';
import settingsRoute from '../../routes/projects/[id]/settings/+page.svelte?raw';
import sourceRedirect from '../../routes/projects/[id]/settings/source/+page.ts?raw';
import resourcesRedirect from '../../routes/projects/[id]/settings/resources/+page.ts?raw';
import webhookRedirect from '../../routes/projects/[id]/settings/webhook/+page.ts?raw';
import dangerRoute from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';
import environmentRoute from '../../routes/projects/[id]/env/+page.svelte?raw';
import generalInformation from '../components/ProjectEffectiveConfiguration.svelte?raw';
import environmentSettings from '../components/ProjectEnvironmentSettings.svelte?raw';
import secretField from '../components/SecretField.svelte?raw';
import selectMenu from '../components/SelectMenu.svelte?raw';

describe('project settings product contract', () => {
	it('consolidates short configuration domains into one Settings workspace', () => {
		expect(settingsRoute).toContain('ProjectCombinedSettings');
		expect(settingsRoute).toContain('section="settings"');
		expect(combinedSettings).toContain('>Settings</h1>');
		for (const section of ['General', 'Source', 'Resources', 'Webhook']) {
			expect(combinedSettings).toContain(`>${section}<`);
		}
		expect(settingsWorkspace).not.toContain('max-width: 64rem');
	});

	it('renders immutable project identity as information instead of readonly fields', () => {
		expect(generalInformation).toContain('Project name');
		expect(generalInformation).toContain('Fixed after creation.');
		expect(generalInformation).toContain('Public URL');
		expect(generalInformation).toContain('Deployment type');
		expect(generalInformation).not.toContain('<input');
	});

	it('keeps source analysis automatic and preserves validation before save', () => {
		expect(combinedSettings).toContain('Advanced source settings');
		expect(combinedSettings).toContain('ariaLabel="Deployment branch"');
		expect(combinedSettings).toContain('ariaLabel="Base directory"');
		expect(combinedSettings).toContain('api.projects.inspectRepository');
		expect(combinedSettings).toContain('validateRepositoryBeforeSave');
		expect(combinedSettings).not.toContain('Validate source');
	});

	it('keeps resource profiles compact and runtime information factual', () => {
		expect(combinedSettings).toContain('ariaLabel="Resource profile"');
		expect(combinedSettings).toContain('Advanced resource limits');
		expect(combinedSettings).toContain('Runtime resources');
		expect(combinedSettings).toContain('api.projects.composeResources');
		expect(combinedSettings).not.toContain('Reset resources');
		expect(selectMenu).toContain('role="listbox"');
	});

	it('keeps environment management on the dedicated project env route', () => {
		expect(environmentRoute).toContain('ProjectEnvironmentSettings');
		expect(environmentRoute).toContain('projectId={$page.params.id');
		expect(combinedSettings).not.toContain('ProjectEnvironmentSettings');
		expect(environmentSettings).toContain('Add variable');
		expect(environmentSettings).toContain('Import .env');
		expect(secretField).toContain('••••••••');
	});

	it('keeps webhook actions in Settings and destructive actions isolated', () => {
		expect(combinedSettings).toContain('Deploy on GitHub push events.');
		expect(combinedSettings).toContain('Setup guide');
		expect(combinedSettings).toContain('ConfirmActionDialog');
		expect(dangerRoute).toContain('section="danger"');
	});

	it('redirects the old thin settings leaves to the consolidated route', () => {
		for (const route of [sourceRedirect, resourcesRedirect, webhookRedirect]) {
			expect(route).toContain("redirect(307, `/projects/${params.id}/settings`)");
		}
	});
});
