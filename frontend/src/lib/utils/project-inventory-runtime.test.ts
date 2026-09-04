import { describe, expect, it } from 'vitest';
import projectsPage from '../../routes/projects/+page.svelte?raw';
import runtimeModeIcon from '../components/RuntimeModeIcon.svelte?raw';

describe('project inventory runtime presentation', () => {
	it('keeps Projects as the compact inventory baseline with explicit runtime semantics', () => {
		expect(projectsPage).toContain("import RuntimeModeIcon from '$components/RuntimeModeIcon.svelte'");
		expect(projectsPage.match(/<RuntimeModeIcon mode=\{project\.deployMode\}/g)?.length ?? 0).toBeGreaterThanOrEqual(2);
		expect(projectsPage).toContain('<th>Limits</th>');
		expect(projectsPage).not.toContain('<th>Uptime / release</th>');
		expect(projectsPage).toContain('{formatDate(project.updatedAt)}');
		expect(projectsPage).toContain('<th>Updated</th>');
		expect(projectsPage).toContain("% CPU</span>");
		expect(projectsPage).toContain('{project.memoryLimitMb} MB');
		expect(projectsPage).toContain('{project.cpuLimit} CPU');
	});

	it('uses web, Compose, and Docker-shaped runtime icons instead of generic file/package glyphs', () => {
		expect(runtimeModeIcon).toContain("mode === 'static'");
		expect(runtimeModeIcon).toContain("mode === 'compose'");
		expect(runtimeModeIcon).toContain('<circle cx="12" cy="12" r="8.5" />');
		expect(runtimeModeIcon).toContain('M4 10.5h13.2');
		expect(projectsPage).not.toContain('function runtimeIcon(project: Project)');
	});
});
