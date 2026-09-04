import { describe, expect, it } from 'vitest';
import projectsPage from '../../routes/projects/+page.svelte?raw';
import webhookStatus from '../components/ProjectWebhookStatus.svelte?raw';

describe('projects webhook inventory status', () => {
	it('adds webhook connection evidence to the desktop inventory and compact project rows', () => {
		expect(projectsPage).toContain('<th>Webhook</th>');
		expect(projectsPage).toContain('ProjectWebhookStatus projectId={project.id}');
		expect(projectsPage).toContain("applicable={project.sourceType === 'git'}");
		expect(projectsPage).toContain('<span>Webhook</span><ProjectWebhookStatus');
	});

	it('uses the existing verified delivery status contract without polling every project refresh', () => {
		expect(webhookStatus).toContain('api.projects.webhookStatus(projectId)');
		expect(webhookStatus).toContain('const webhookStatusTtlMs = 60_000');
		expect(webhookStatus).toContain("status === 'connected'");
		expect(webhookStatus).toContain("status === 'issue'");
		expect(webhookStatus).toContain("status === 'unverified'");
		expect(webhookStatus).toContain('/settings/webhook');
	});

	it('keeps the inventory surface flat and treats non-git projects as not applicable', () => {
		expect(webhookStatus).toContain('{#if !applicable}');
		expect(webhookStatus).not.toContain('rounded-md');
		expect(webhookStatus).not.toContain('border');
	});
});
