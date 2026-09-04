import { describe, expect, it } from 'vitest';
import settingsPage from '../../routes/projects/[id]/settings/+page.svelte?raw';
import webhookPage from '../../routes/projects/[id]/settings/webhook/+page.svelte?raw';
import settingsWorkspace from '../components/SettingsWorkspace.svelte?raw';
import combinedSettings from '../components/ProjectCombinedSettings.svelte?raw';
import webhookSettings from '../components/ProjectWebhookSettings.svelte?raw';
import sidebar from '../components/ProjectDetailSidebar.svelte?raw';

describe('project settings workspace', () => {
	it('uses the full main canvas with a compact aligned Source and Resources layout', () => {
		expect(settingsPage).toContain('ProjectCombinedSettings');
		expect(settingsPage).toContain('section="settings"');
		expect(settingsWorkspace).toContain('width: 100%');
		expect(settingsWorkspace).not.toContain('max-width: 64rem');
		expect(combinedSettings).toContain('project-settings-workspace w-full');
		expect(combinedSettings).toContain('lg:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]');
		expect(combinedSettings).toContain('sm:grid-cols-2');
		expect(combinedSettings).toContain("grid min-w-0 grid-cols-[7rem_minmax(0,1fr)]");
		expect(combinedSettings).not.toContain('max-w-xl');
		expect(combinedSettings).not.toContain('max-w-md');
	});

	it('keeps source analysis and resource state while removing redundant general and webhook chrome', () => {
		expect(combinedSettings).toContain('api.projects.get');
		expect(combinedSettings).toContain('api.projects.inspectRepository');
		expect(combinedSettings).toContain('api.projects.composeResources');
		expect(combinedSettings).toContain('api.admin.getSettings');
		expect(combinedSettings).toContain('validateRepositoryBeforeSave');
		expect(combinedSettings).toContain('sourceChanged');
		expect(combinedSettings).toContain('resourcesChanged');
		expect(combinedSettings).toContain('>Source<');
		expect(combinedSettings).toContain('>Resources<');
		expect(combinedSettings).not.toContain('ProjectEffectiveConfiguration');
		expect(combinedSettings).not.toContain('api.projects.regenerateWebhookSecret');
		expect(combinedSettings).not.toContain('>General<');
		expect(combinedSettings).not.toContain('>Webhook<');
	});

	it('keeps Webhook as a first-class configuration destination', () => {
		expect(sidebar).toContain("label: 'Settings'");
		expect(sidebar).toContain("label: 'Webhook'");
		for (const oldLabel of ['General', 'Source', 'Resources']) {
			expect(sidebar).not.toContain(`label: '${oldLabel}'`);
		}
		expect(sidebar).toContain("label: 'Danger zone'");
		expect(webhookPage).toContain('ProjectWebhookSettings');
		expect(webhookPage).toContain('section="webhook"');
	});

	it('derives webhook connection state from delivery evidence and links official GitHub setup docs', () => {
		expect(webhookSettings).toContain('api.projects.webhookStatus');
		expect(webhookSettings).toContain('webhookStatus?.lastDelivery');
		expect(webhookSettings).toContain('signed GitHub delivery');
		expect(webhookSettings).toContain('api.projects.regenerateWebhookSecret');
		expect(webhookSettings).toContain('https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks');
		expect(webhookSettings).toContain('ConfirmActionDialog');
		expect(webhookSettings).toContain('No GitHub delivery has been verified');
	});
});