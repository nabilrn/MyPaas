import { describe, expect, it } from 'vitest';
import settingsPage from '../../routes/projects/[id]/settings/+page.svelte?raw';
import settingsWorkspace from '../components/SettingsWorkspace.svelte?raw';
import combinedSettings from '../components/ProjectCombinedSettings.svelte?raw';
import sidebar from '../components/ProjectDetailSidebar.svelte?raw';

describe('project settings workspace', () => {
	it('uses the full main canvas with compact controls', () => {
		expect(settingsPage).toContain('ProjectCombinedSettings');
		expect(settingsPage).toContain('section="settings"');
		expect(settingsWorkspace).toContain('width: 100%');
		expect(settingsWorkspace).not.toContain('max-width: 64rem');
		expect(combinedSettings).toContain('project-settings-workspace w-full');
		expect(combinedSettings).toContain('max-w-xl');
		expect(combinedSettings).toContain('max-w-md');
	});

	it('keeps source analysis, resource state, and webhook actions in one loaded component', () => {
		expect(combinedSettings).toContain('api.projects.get');
		expect(combinedSettings).toContain('api.projects.inspectRepository');
		expect(combinedSettings).toContain('api.projects.composeResources');
		expect(combinedSettings).toContain('api.admin.getSettings');
		expect(combinedSettings).toContain('api.projects.regenerateWebhookSecret');
		expect(combinedSettings).toContain('validateRepositoryBeforeSave');
		expect(combinedSettings).toContain('sourceChanged');
		expect(combinedSettings).toContain('resourcesChanged');
	});

	it('consolidates sidebar destinations instead of forcing thin settings pages', () => {
		expect(sidebar).toContain("label: 'Settings'");
		for (const oldLabel of ['General', 'Source', 'Resources', 'Webhook']) {
			expect(sidebar).not.toContain(`label: '${oldLabel}'`);
		}
		expect(sidebar).toContain("label: 'Danger zone'");
	});
});
