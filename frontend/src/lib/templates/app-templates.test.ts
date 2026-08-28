import { describe, expect, it } from 'vitest';
import { appTemplates, generateTemplateSecret, initialTemplateEnv, missingRequiredTemplateEnv, templateEnvValue } from './app-templates';

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
			for (const [service, resource] of Object.entries(template.serviceResources ?? {})) {
				expect(service.length).toBeGreaterThan(0);
				expect(resource.memoryLimitMb).toBeGreaterThan(0);
				expect(resource.cpuLimit).toBeGreaterThan(0);
				if (template.source.type === 'compose') {
					expect(service).not.toBe(template.source.mainService);
				}
			}
			for (const route of template.additionalRoutes ?? []) {
				expect(template.source.type).toBe('compose');
				expect(route.name).toMatch(/^[a-z0-9-]+$/);
				expect(route.service.length).toBeGreaterThan(0);
				expect(route.containerPort).toBeGreaterThan(0);
				expect(route.containerPort).toBeLessThanOrEqual(65535);
			}
		}
	});

	it('includes catalogued templates for the real application patterns added through phase 8', () => {
		for (const id of ['drawdb', 'meilisearch', 'directus', 'forgejo', 'paperless-ngx', 'openclaw', 'minio']) {
			expect(appTemplates.some((template) => template.id === id)).toBe(true);
		}
	});

	it('carries catalogued secondary service guards for multi-service templates', () => {
		const nocodb = appTemplates.find((template) => template.id === 'nocodb');
		expect(nocodb?.serviceResources).toEqual({
			worker: { memoryLimitMb: 768, cpuLimit: 0.75 },
			db: { memoryLimitMb: 768, cpuLimit: 0.5 },
			redis: { memoryLimitMb: 256, cpuLimit: 0.25 }
		});
		expect(appTemplates.find((template) => template.id === 'umami')?.serviceResources?.db).toEqual({ memoryLimitMb: 512, cpuLimit: 0.5 });
		expect(appTemplates.find((template) => template.id === 'ghost')?.serviceResources?.db).toEqual({ memoryLimitMb: 768, cpuLimit: 0.5 });
		expect(appTemplates.find((template) => template.id === 'paperless-ngx')?.serviceResources).toEqual({
			db: { memoryLimitMb: 768, cpuLimit: 0.5 },
			broker: { memoryLimitMb: 256, cpuLimit: 0.25 }
		});
		expect(appTemplates.find((template) => template.id === 'openclaw')?.serviceResources).toEqual({
			'openclaw-bootstrap': { memoryLimitMb: 512, cpuLimit: 0.5 }
		});
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

	it('blocks templates only when required user values are actually missing', () => {
		const directus = appTemplates.find((template) => template.id === 'directus');
		expect(directus).toBeTruthy();
		if (!directus) return;

		const values = initialTemplateEnv(directus);
		expect(missingRequiredTemplateEnv(directus, values)).toEqual(['ADMIN_EMAIL']);
		values.ADMIN_EMAIL = 'owner@example.com';
		expect(missingRequiredTemplateEnv(directus, values)).toEqual([]);
	});

	it('resolves managed public host and URL fields without user input', () => {
		const forgejo = appTemplates.find((template) => template.id === 'forgejo');
		expect(forgejo).toBeTruthy();
		if (!forgejo) return;

		const values = initialTemplateEnv(forgejo);
		const host = 'forgejo.mypaas.example';
		const url = `https://${host}`;
		expect(missingRequiredTemplateEnv(forgejo, values, url, host)).toEqual([]);

		const domainField = forgejo.env.find((field) => field.key === 'FORGEJO_DOMAIN');
		const rootURLField = forgejo.env.find((field) => field.key === 'FORGEJO_ROOT_URL');
		expect(domainField && templateEnvValue(domainField, values, url, host)).toBe(host);
		expect(rootURLField && templateEnvValue(rootURLField, values, url, host)).toBe(url);
	});

	it('resolves MinIO console route URL and route contract', () => {
		const minio = appTemplates.find((template) => template.id === 'minio');
		expect(minio).toBeTruthy();
		if (!minio) return;

		expect(minio.additionalRoutes).toEqual([{ name: 'console', service: 'minio', containerPort: 9001 }]);
		const values = initialTemplateEnv(minio);
		const consoleURL = 'https://minio-console.mypaas.example';
		const routeURLs = { console: consoleURL };
		expect(missingRequiredTemplateEnv(minio, values, 'https://minio.mypaas.example', 'minio.mypaas.example', routeURLs)).toEqual([]);
		const redirectField = minio.env.find((field) => field.key === 'MINIO_BROWSER_REDIRECT_URL');
		expect(redirectField && templateEnvValue(redirectField, values, '', '', routeURLs)).toBe(consoleURL);
	});

	it('generates secrets for Paperless-ngx, OpenClaw, and MinIO', () => {
		for (const id of ['paperless-ngx', 'openclaw', 'minio']) {
			const template = appTemplates.find((item) => item.id === id);
			expect(template).toBeTruthy();
			if (!template) continue;
			const values = initialTemplateEnv(template);
			for (const field of template.env.filter((item) => item.kind === 'secret')) {
				expect(values[field.key].length).toBeGreaterThan(15);
			}
		}
	});
});
