import { describe, expect, it } from 'vitest';
import settingsWorkspace from './SettingsWorkspace.svelte?raw';
import storageCapacityMetric from './StorageCapacityMetric.svelte?raw';
import projectGeneral from '../../routes/projects/[id]/settings/+page.svelte?raw';
import projectSource from '../../routes/projects/[id]/settings/source/+page.svelte?raw';
import projectResources from '../../routes/projects/[id]/settings/resources/+page.svelte?raw';
import projectWebhook from '../../routes/projects/[id]/settings/webhook/+page.svelte?raw';
import projectDanger from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';

describe('settings workspace layout contract', () => {
	it('centralizes project settings geometry instead of route-specific style overrides', () => {
		for (const page of [projectGeneral, projectSource, projectResources, projectWebhook, projectDanger]) {
			expect(page).toContain('SettingsWorkspace');
			expect(page).not.toContain('<style>');
		}

		expect(settingsWorkspace).toContain('max-width: 64rem');
		expect(settingsWorkspace).toContain('var(--workspace-divider)');
		expect(settingsWorkspace).toContain('padding-inline: 1.25rem');
		expect(settingsWorkspace).toContain('padding-inline: 1rem');
		expect(settingsWorkspace).toContain('padding-top: 1rem');
		expect(settingsWorkspace).not.toContain('max-width: 56rem');
	});

	it('keeps decorative icons out of ordinary settings-row geometry', () => {
		expect(settingsWorkspace).toContain('decorative icons are not part of ordinary field rows');
		expect(settingsWorkspace).not.toContain('@lucide/svelte');
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
