import { describe, expect, it } from 'vitest';
import { appTemplates, generateTemplateSecret, initialTemplateEnv, missingRequiredTemplateEnv } from './app-templates';

describe('app templates', () => {
	it('keeps template ids unique and source contracts explicit', () => {
		const ids = appTemplates.map((template) => template.id);
		expect(new Set(ids).size).toBe(ids.length);
		for (const template of appTemplates) {
			expect(template.appPort).toBeGreaterThan(0);
			expect(template.memoryLimitMb).toBeGreaterThan(0);
			expect(template.cpuLimit).toBeGreaterThan(0);
			expect(template.compatibility.catalogId).toBe(template.id);
			expect(template.compatibility.status).toBe('catalogued-pattern');
			if (template.source.type === 'compose') {
				expect(template.source.baseDirectory).toMatch(/^templates\/manifests\//);
				expect(template.source.mainService.length).toBeGreaterThan(0);
			}
			if (template.source.type === 'dockerfile') {
				expect(template.source.repoUrl).toMatch(/^https:\/\/github\.com\//);
				expect(template.source.branch.length).toBeGreaterThan(0);
			}
		}
	});

	it('includes catalogued templates for the new real application patterns', () => {
		for (const id of ['drawdb', 'meilisearch', 'directus']) {
			expect(appTemplates.some((template) => template.id === id)).toBe(true);
		}
	});

	it('does not ship fixed values for secret fields', () => {
		for (const template of appTemplates) {
			const first = initialTemplateEnv(template);
			const second = initialTemplateEnv(template);
			for (const field of template.env.filter((item) => item.kind === 'secret')) {
				expect(first[field.key]).toBeTruthy();
				expect(second[field.key]).toBeTruthy();
				expect(first[field.key]).not.toBe(second[field.key]);
			}
		}
	});

	it('generates requested hex secrets with deterministic length', () => {
		const value = generateTemplateSecret({
			key: 'TEST_SECRET',
			label: 'Test',
			kind: 'secret',
			description: 'test',
			bytes: 16,
			format: 'hex'
		});
		expect(value).toMatch(/^[a-f0-9]{32}$/);
	});

	it('blocks templates only when required values are actually missing', () => {
		const directus = appTemplates.find((template) => template.id === 'directus');
		expect(directus).toBeTruthy();
		if (!directus) return;

		const values = initialTemplateEnv(directus);
		expect(missingRequiredTemplateEnv(directus, values)).toEqual(['ADMIN_EMAIL']);
		values.ADMIN_EMAIL = 'owner@example.com';
		expect(missingRequiredTemplateEnv(directus, values)).toEqual([]);
	});
});
