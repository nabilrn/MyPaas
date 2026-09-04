import { describe, expect, it } from 'vitest';
import projectsPage from '../../routes/projects/+page.svelte?raw';

describe('project inventory runtime presentation', () => {
	it('keeps Updated as project update metadata while adding runtime-mode iconography', () => {
		expect(projectsPage).toContain('function runtimeIcon(project: Project)');
		expect(projectsPage).toContain("project.deployMode === 'compose'");
		expect(projectsPage).toContain("project.deployMode === 'static'");
		expect(projectsPage.match(/runtimeIcon\(project\)/g)?.length ?? 0).toBeGreaterThanOrEqual(2);
		expect(projectsPage).toContain('{formatDate(project.updatedAt)}');
		expect(projectsPage).toContain('<th>Updated</th>');
	});
});
