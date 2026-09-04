import { describe, expect, it } from 'vitest';
import rootLayout from '../../routes/+layout.svelte?raw';
import envPage from '../../routes/projects/[id]/env/+page.svelte?raw';
import databaseLayout from '../../routes/projects/[id]/database/+layout.svelte?raw';
import settingsPage from '../../routes/projects/[id]/settings/+page.svelte?raw';
import dangerPage from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';
import combinedSettings from '../components/ProjectCombinedSettings.svelte?raw';
import brandLogo from '../components/BrandLogo.svelte?raw';

describe('project detail top rhythm', () => {
	it('aligns leaf title optical top with the operational header reference', () => {
		expect(envPage).toContain('px-5 pt-4');
		expect(databaseLayout).toContain('px-5 pb-3 pt-4');
		expect(settingsPage).toContain('SettingsWorkspace');
		expect(combinedSettings).toContain('px-5');
		expect(combinedSettings).toContain('pt-4');
		expect(combinedSettings).toContain('pb-3');
		expect(dangerPage).toContain('SettingsWorkspace');
	});
});

describe('MyPaaS SVG brand contract', () => {
	it('uses the optimized SVG icon, wordmark, and explicit white favicon throughout the dashboard chrome', () => {
		expect(brandLogo).toContain('assets/brand/mypaas-icon.svg');
		expect(brandLogo).toContain('assets/brand/mypaas-logo.svg');
		expect(brandLogo).not.toContain('.png');
		expect(rootLayout).toContain('assets/brand/mypaas-favicon.svg');
		expect(rootLayout).toContain('type="image/svg+xml"');
		expect(rootLayout).not.toContain('.png');
	});
});
