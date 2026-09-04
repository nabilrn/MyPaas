import { describe, expect, it } from 'vitest';
import settingsWorkspace from './SettingsWorkspace.svelte?raw';
import storageCapacityMetric from './StorageCapacityMetric.svelte?raw';
import projectSettings from '../../routes/projects/[id]/settings/+page.svelte?raw';
import projectDanger from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';
import projectSourceRedirect from '../../routes/projects/[id]/settings/source/+page.ts?raw';
import projectResourcesRedirect from '../../routes/projects/[id]/settings/resources/+page.ts?raw';
import projectWebhookRedirect from '../../routes/projects/[id]/settings/webhook/+page.ts?raw';

describe('settings workspace layout contract', () => {
	it('uses the full project main canvas for structural settings strokes', () => {
		for (const page of [projectSettings, projectDanger]) {
			expect(page).toContain('SettingsWorkspace');
			expect(page).not.toContain('<style>');
		}
		expect(projectSettings).toContain('section="settings"');
		expect(settingsWorkspace).toContain('width: 100%');
		expect(settingsWorkspace).not.toContain('max-width: 64rem');
		expect(settingsWorkspace).toContain('var(--workspace-divider)');
		expect(settingsWorkspace).toContain('padding-inline: 1.25rem');
		expect(settingsWorkspace).toContain('padding-inline: 1rem');
	});

	it('preserves legacy settings URLs through redirects', () => {
		for (const route of [projectSourceRedirect, projectResourcesRedirect, projectWebhookRedirect]) {
			expect(route).toContain("redirect(307, `/projects/${params.id}/settings`)");
		}
	});

	it('keeps workspace geometry free of decorative route-local dependencies', () => {
		expect(settingsWorkspace).not.toContain('@lucide/svelte');
		expect(settingsWorkspace).toContain('Controls keep route-specific');
	});
});

describe('storage capacity treatment', () => {
	it('uses the larger desktop-disk icon and a thicker capacity bar', () => {
		expect(storageCapacityMetric).toContain('h-9 w-9');
		expect(storageCapacityMetric).toContain('h-4 overflow-hidden');
		expect(storageCapacityMetric).toContain("grid-cols-[2.75rem_minmax(0,1fr)]");
		expect(storageCapacityMetric).toContain('repeat(4, minmax(0, 1fr))');
	});
});
