import { describe, expect, it } from 'vitest';
import adminLayout from '../../routes/admin/+layout.svelte?raw';
import adminSettingsPage from '../../routes/admin/settings/+page.svelte?raw';
import adminUpdatePage from '../../routes/admin/update/+page.svelte?raw';
import adminUsersPage from '../../routes/admin/users/+page.svelte?raw';
import adminBackupPage from '../../routes/admin/backup/+page.svelte?raw';
import adminMigrationPage from '../../routes/admin/migration/+page.svelte?raw';
import adminMcpPage from '../../routes/admin/mcp/+page.svelte?raw';
import adminAuditPage from '../../routes/admin/audit-logs/+page.svelte?raw';
import createProjectPage from '../../routes/projects/new/+page.svelte?raw';
import createProjectLayout from '../../routes/projects/new/+layout.svelte?raw';
import projectLogsPage from '../../routes/projects/[id]/logs/+page.svelte?raw';
import projectLayout from '../../routes/projects/[id]/+layout.svelte?raw';
import projectSettingsPage from '../../routes/projects/[id]/settings/+page.svelte?raw';
import projectWebhookPage from '../../routes/projects/[id]/settings/webhook/+page.svelte?raw';
import projectSourceRedirect from '../../routes/projects/[id]/settings/source/+page.ts?raw';
import projectResourcesRedirect from '../../routes/projects/[id]/settings/resources/+page.ts?raw';
import projectDangerSettingsPage from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';
import adminSidebar from '../components/AdminSidebar.svelte?raw';
import appHeader from '../components/AppHeader.svelte?raw';
import navbar from '../components/Navbar.svelte?raw';
import projectDetailSidebar from '../components/ProjectDetailSidebar.svelte?raw';
import projectCombinedSettings from '../components/ProjectCombinedSettings.svelte?raw';
import projectWebhookSettings from '../components/ProjectWebhookSettings.svelte?raw';
import projectSettingsSection from '../components/ProjectSettingsSection.svelte?raw';
import {
	administrationNavGroups,
	administrationNavItemForPath,
	administrationNavItems,
	isAdministrationNavItemActive,
	isAdministrationPath
} from '../navigation/administration';
import { administrationNavigationItem } from '../navigation/primary';

describe('administration navigation contract', () => {
	it('keeps one Administration entry in the global navigation', () => {
		expect(administrationNavigationItem).toMatchObject({ label: 'Administration', href: '/admin/settings' });
		expect(navbar).toContain('administrationNavigationItem');
		expect(appHeader).toContain('primaryNavigationItems');
		for (const route of ['/admin/update', '/admin/users', '/admin/audit-logs', '/admin/mcp', '/admin/backup', '/admin/migration']) {
			expect(navbar).not.toContain(route);
			expect(appHeader).not.toContain(route);
		}
	});

	it('marks Administration active throughout the admin route family', () => {
		for (const pathname of ['/admin', '/admin/settings', '/admin/update', '/admin/users', '/admin/backup', '/admin/migration', '/admin/mcp', '/admin/audit-logs']) {
			expect(isAdministrationPath(pathname)).toBe(true);
			expect(isAdministrationNavItemActive(administrationNavItems[0], pathname)).toBe(pathname === '/admin/settings');
		}
		expect(isAdministrationPath('/projects')).toBe(false);
	});

	it('keeps the shared secondary administration groups and breadcrumbs', () => {
		expect(administrationNavGroups.map((group) => group.label)).toEqual(['Platform', 'Operations', 'Integrations', 'Activity']);
		expect(administrationNavItems.map((item) => [item.label, item.href])).toEqual([
			['General', '/admin/settings'],
			['Users', '/admin/users'],
			['System update', '/admin/update'],
			['Backup', '/admin/backup'],
			['Migration', '/admin/migration'],
			['MCP', '/admin/mcp'],
			['Audit logs', '/admin/audit-logs']
		]);
		expect(administrationNavItemForPath('/admin/users').label).toBe('Users');
		expect(administrationNavItemForPath('/admin/update').label).toBe('System update');
		expect(appHeader).toContain('administrationNavItemForPath');
	});

	it('uses the canonical administration shell', () => {
		expect(adminLayout).toContain('lg:grid-cols-[12rem_minmax(0,1fr)]');
		expect(adminLayout).toContain('min-w-0 px-3.5 py-3');
		expect(adminLayout).toContain('border-[color:var(--workspace-divider)]');
		expect(adminSidebar).toContain('administrationNavGroups');
		expect(adminSidebar).toContain('border-l-2');
		expect(createProjectPage).not.toContain('AdminSidebar');
		expect(projectLogsPage).not.toContain('AdminSidebar');
	});

	it('keeps the redesigned Administration pages task-first and full-width', () => {
		expect(adminSettingsPage).toContain('admin-general-workspace w-full');
		expect(adminSettingsPage).toContain('ConfirmActionDialog');
		expect(adminUpdatePage).toContain('admin-update-workspace w-full');
		expect(adminUpdatePage).toContain('Update progress');
		expect(adminUpdatePage).toContain('api.admin.triggerUpdate');
		expect(adminBackupPage).toContain('admin-backup-workspace w-full');
		expect(adminBackupPage).toContain('Cloudflare R2');
		expect(adminBackupPage).toContain('Cloudflare R2 docs');
		expect(adminBackupPage).toContain('/api/admin/settings/s3?validate=1');
		expect(adminMigrationPage).toContain('migration-workspace w-full');
		expect(adminMigrationPage).toContain('MigrationTransferIllustration');
		expect(adminMigrationPage).toContain('Runtime safety');
		expect(adminMcpPage).toContain('admin-mcp-workspace w-full');
		expect(adminMcpPage).toContain('Supported clients');
		expect(adminMcpPage).toContain('Agent capabilities');
		expect(adminMcpPage).toContain('Connect a client');
		expect(adminMcpPage).toContain('ConfirmActionDialog');
		expect(adminUsersPage).toContain('role="dialog"');
		expect(adminAuditPage).toContain('Audit logs copied');
	});

	it('keeps owner dialog keyboard mechanics explicit', () => {
		expect(adminUsersPage).toContain('trapDialogFocus');
		expect(adminUsersPage).toContain("event.key === 'Escape'");
		expect(adminUsersPage).toContain('tabindex="-1"');
	});
});

describe('project secondary navigation contract', () => {
	it('keeps Create Project full-width without a secondary sidebar', () => {
		expect(createProjectLayout).not.toContain('ProjectNewSidebar');
		expect(createProjectLayout).not.toContain('<aside');
		expect(createProjectLayout).not.toContain('lg:grid-cols-[12rem_minmax(0,1fr)]');
		expect(createProjectLayout).toContain('min-h-[calc(100vh-3.5rem)]');
	});

	it('uses one compact project-detail secondary sidebar', () => {
		expect(projectLayout).toContain('ProjectDetailSidebar');
		expect(projectLayout).toContain('lg:grid-cols-[12rem_minmax(0,1fr)]');
		for (const label of ['Overview', 'Deployments', 'Logs', 'Environment', 'Database', 'Settings', 'Webhook', 'Danger zone']) {
			expect(projectDetailSidebar).toContain(`label: '${label}'`);
		}
		for (const removedLabel of ['General', 'Source', 'Resources']) {
			expect(projectDetailSidebar).not.toContain(`label: '${removedLabel}'`);
		}
		expect(projectDetailSidebar).not.toContain("label: 'Integrations'");
	});

	it('keeps Settings compact while Webhook remains a dedicated configuration workspace', () => {
		expect(projectSettingsPage).toContain('ProjectCombinedSettings');
		expect(projectSettingsPage).toContain('section="settings"');
		expect(projectCombinedSettings).toContain('project-settings-workspace w-full');
		expect(projectCombinedSettings).toContain('lg:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)]');
		expect(projectCombinedSettings).toContain('>Source<');
		expect(projectCombinedSettings).toContain('>Resources<');
		expect(projectCombinedSettings).not.toContain('>General<');
		expect(projectCombinedSettings).not.toContain('>Webhook<');
		expect(projectCombinedSettings).toContain('api.projects.inspectRepository');
		expect(projectCombinedSettings).toContain('api.projects.composeResources');
		expect(projectCombinedSettings).not.toContain('api.projects.regenerateWebhookSecret');
		expect(projectWebhookPage).toContain('ProjectWebhookSettings');
		expect(projectWebhookSettings).toContain('api.projects.webhookStatus');
		expect(projectWebhookSettings).toContain('api.projects.regenerateWebhookSecret');
		expect(projectWebhookSettings).toContain('ConfirmActionDialog');
	});

	it('redirects only the legacy Source and Resources routes while keeping Webhook and Danger zone separate', () => {
		for (const redirectSource of [projectSourceRedirect, projectResourcesRedirect]) {
			expect(redirectSource).toContain("redirect(307, `/projects/${params.id}/settings`)");
		}
		expect(projectWebhookPage).toContain('section="webhook"');
		expect(projectDangerSettingsPage).toContain('ProjectSettingsSection');
		expect(projectDangerSettingsPage).toContain('section="danger"');
		expect(projectSettingsSection).toContain("section: 'general' | 'source' | 'resources' | 'webhook' | 'danger'");
	});
});
