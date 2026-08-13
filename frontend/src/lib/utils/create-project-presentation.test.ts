import { describe, expect, it } from 'vitest';

import {
	createProjectBlockingSummary,
	presentRepositoryInspectionError
} from './create-project-presentation';

describe('presentRepositoryInspectionError', () => {
	it('turns credential-oriented git output into a task-oriented message', () => {
		const raw = "validation failed: failed to inspect remote branches: fatal: could not read Username for 'https://github.com': No such device or address";
		expect(presentRepositoryInspectionError(raw)).toEqual({
			message: 'This repository is private or inaccessible to MyPaas. Use a public repository or provide a source MyPaas can access.',
			detail: raw
		});
	});

	it('separates repository-not-found copy from raw technical detail', () => {
		const result = presentRepositoryInspectionError('fatal: repository not found');
		expect(result.message).toMatch(/Repository not found/);
		expect(result.detail).toBe('fatal: repository not found');
	});

	it('uses a safe generic message for unknown inspection failures', () => {
		const result = presentRepositoryInspectionError('git exited with status 128');
		expect(result.message).toMatch(/could not inspect this repository/i);
		expect(result.detail).toBe('git exited with status 128');
	});
});

describe('createProjectBlockingSummary', () => {
	it('puts blocking Compose issues before missing environment values', () => {
		expect(createProjectBlockingSummary({
			composeBlockingMessages: ['Mounting /var/run/docker.sock into app containers is not allowed.'],
			missingRequiredEnvKeys: ['JWT_SECRET', 'POSTGRES_PASSWORD']
		})).toEqual([
			'Mounting /var/run/docker.sock into app containers is not allowed.',
			'Add required environment values: JWT_SECRET, POSTGRES_PASSWORD.'
		]);
	});

	it('keeps long required-env lists concise', () => {
		expect(createProjectBlockingSummary({
			missingRequiredEnvKeys: ['A', 'B', 'C', 'D', 'E', 'F']
		})).toEqual(['Add required environment values: A, B, C, D +2 more.']);
	});
});
