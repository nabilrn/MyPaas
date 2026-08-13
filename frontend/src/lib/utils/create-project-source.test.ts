import { describe, expect, it } from 'vitest';

import {
	createProjectEnvironmentCopy,
	retainUserProvidedEnvironmentDrafts
} from './create-project-source';

describe('createProjectEnvironmentCopy', () => {
	it('keeps repository detection language for Git sources', () => {
		const copy = createProjectEnvironmentCopy('git');
		expect(copy.sectionDescription).toMatch(/repository/i);
		expect(copy.emptyState).toMatch(/detected/i);
		expect(copy.noRequiredSummary).toMatch(/Scan complete/);
	});

	it('does not imply registry images are scanned for environment variables', () => {
		const copy = createProjectEnvironmentCopy('registry');
		expect(copy.setupSummary).not.toMatch(/detected/i);
		expect(copy.sectionDescription).toMatch(/not scanned/i);
		expect(copy.sectionDescription).not.toMatch(/detected from the repository/i);
		expect(copy.emptyState).toMatch(/have been added/i);
		expect(copy.noRequiredSummary).not.toMatch(/scan/i);
		expect(copy.portRequirement).toMatch(/Set the port/i);
	});
});

describe('retainUserProvidedEnvironmentDrafts', () => {
	it('drops repository-discovered variables when switching to registry', () => {
		expect(retainUserProvidedEnvironmentDrafts([
			{ key: 'JWT_SECRET', source: '.env.example', value: '', sensitive: true },
			{ key: 'PORT', source: 'compose.yaml', value: '3000', sensitive: false }
		])).toEqual([]);
	});

	it('preserves manual and imported env values while removing repository metadata', () => {
		expect(retainUserProvidedEnvironmentDrafts([
			{
				key: 'API_KEY',
				source: 'manual, .env.example',
				value: 'keep-me',
				sensitive: true,
				defaultValue: 'stale-default',
				services: ['api'],
				conflict: { values: [] }
			},
			{ key: 'FEATURE_FLAG', source: 'env-file', value: 'true', sensitive: false }
		])).toEqual([
			{
				key: 'API_KEY',
				source: 'manual',
				value: 'keep-me',
				sensitive: true
			},
			{
				key: 'FEATURE_FLAG',
				source: 'env-file',
				value: 'true',
				sensitive: false
			}
		]);
	});
});
