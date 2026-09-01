import { describe, expect, it } from 'vitest';
import createProjectPage from '../../routes/projects/new/+page.svelte?raw';

const chooseRepositoryFunction =
	createProjectPage.match(/function chooseGithubRepository\(repository: GitHubRepository\) \{[\s\S]*?\n\t\}/)?.[0] ?? '';

describe('create project repository picker', () => {
	it('clears stale inspection state before applying the selected repository', () => {
		const resetIndex = chooseRepositoryFunction.indexOf('resetRepositoryInspection();');
		const repoIndex = chooseRepositoryFunction.indexOf('form.repoUrl = repository.cloneUrl;');

		expect(resetIndex).toBeGreaterThanOrEqual(0);
		expect(repoIndex).toBeGreaterThan(resetIndex);
	});

	it('seeds the selected default branch while repository inspection refreshes the full branch list', () => {
		expect(chooseRepositoryFunction).toContain('const selectedBranch = repository.defaultBranch.trim();');
		expect(chooseRepositoryFunction).toContain('form.branch = selectedBranch;');
		expect(chooseRepositoryFunction).toContain('defaultBranch = selectedBranch;');
		expect(chooseRepositoryFunction).toContain('branchOptions = normalizeBranches([], selectedBranch);');
		expect(chooseRepositoryFunction).toContain('void inspectRepository(false, true)');
	});
});
