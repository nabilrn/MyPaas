import { describe, expect, it } from 'vitest';
import appHeader from '../components/AppHeader.svelte?raw';
import projectDetailSidebar from '../components/ProjectDetailSidebar.svelte?raw';

describe('project detail navigation consistency', () => {
	it('reacts active state to the current project leaf route', () => {
		expect(projectDetailSidebar).toContain('$: pathname = $page.url.pathname');
		expect(projectDetailSidebar).toContain('isActive(item, pathname)');
		expect(projectDetailSidebar).toContain('if (item.exact) return currentPath === item.href');
	});

	it('names settings leaves directly in the breadcrumb', () => {
		expect(appHeader).toContain("'': 'Settings'");
		expect(appHeader).toContain("source: 'Source'");
		expect(appHeader).toContain("resources: 'Resources'");
		expect(appHeader).toContain("webhook: 'Webhook'");
		expect(appHeader).toContain("danger: 'Danger zone'");
		expect(appHeader).not.toContain("settings: 'Settings'");
	});
});