import { describe, expect, it } from 'vitest';
import { describeProjectSource, describeRepoUrl } from './repository';

describe('describeRepoUrl', () => {
	it('extracts owner/repo from GitHub HTTPS URLs', () => {
		expect(describeRepoUrl('https://github.com/nabilrn/MyPaas')).toEqual({
			host: 'github',
			label: 'nabilrn/MyPaas',
			href: 'https://github.com/nabilrn/MyPaas'
		});
	});

	it('strips .git suffix and trailing slashes', () => {
		expect(describeRepoUrl('https://github.com/nabilrn/MyPaas.git/')).toEqual({
			host: 'github',
			label: 'nabilrn/MyPaas',
			href: 'https://github.com/nabilrn/MyPaas'
		});
	});

	it('parses SCP-style SSH remotes', () => {
		const result = describeRepoUrl('git@github.com:nabilrn/MyPaas.git');
		expect(result.host).toBe('github');
		expect(result.label).toBe('nabilrn/MyPaas');
		expect(result.href).toBe('https://github.com/nabilrn/MyPaas');
	});

	it('recognizes GitLab', () => {
		const result = describeRepoUrl('https://gitlab.com/group/sub/project');
		expect(result.host).toBe('gitlab');
		expect(result.label).toBe('group/sub/project');
	});

	it('falls back to a generic git label for unknown hosts', () => {
		const result = describeRepoUrl('https://git.example.com/team/service.git');
		expect(result.host).toBe('git');
		expect(result.label).toBe('git.example.com/team/service');
	});

	it('keeps the raw value when the URL cannot be parsed', () => {
		const result = describeRepoUrl('not a url');
		expect(result.host).toBe('git');
		expect(result.label).toBe('not a url');
		expect(result.href).toBeNull();
	});
});

describe('describeProjectSource', () => {
	it('uses the image reference for registry projects', () => {
		expect(
			describeProjectSource({ sourceType: 'registry', repoUrl: '', imageRef: 'grafana/grafana:10.4.2' })
		).toEqual({ host: 'registry', label: 'grafana/grafana:10.4.2', href: null });
	});

	it('parses the repository URL for git projects', () => {
		const result = describeProjectSource({
			sourceType: 'git',
			repoUrl: 'https://github.com/nabilrn/MyPaas',
			imageRef: null
		});
		expect(result.host).toBe('github');
		expect(result.label).toBe('nabilrn/MyPaas');
	});
});
