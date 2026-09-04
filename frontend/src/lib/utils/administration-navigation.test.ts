import { describe, expect, it } from 'vitest';
import adminLayout from '../../routes/admin/+layout.svelte?raw';
import adminSettingsPage from '../../routes/admin/settings/+page.svelte?raw';
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
import projectSourceSettingsPage from '../../routes/projects/[id]/settings/source/+page.svelte?raw';
import projectResourcesSettingsPage from '../../routes/projects/[id]/settings/resources/+page.svelte?raw';
import projectWebhookSettingsPage from '../../routes/projects/[id]/settings/webhook/+page.svelte?raw';
import projectDangerSettingsPage from '../../routes/projects/[id]/settings/danger/+page.svelte?raw';
import adminSidebar from '../components/AdminSidebar.svelte?raw';
import appHeader from '../components/AppHeader.svelte?raw';
import navbar from '../components/Navbar.svelte?raw';
import projectDetailSidebar from '../components/ProjectDetailSidebar.svelte?raw';
import projectNewSidebar from '../components/ProjectNewSidebar.svelte?raw';
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
	it('keeps one Administration entry in global navigation', () => {
		expect(administrationNavigationItem).toMatchObject({ label: 'Administration', href: '/admin/settings' });
		expect(navbar).toContain('administrationNavigationItem');
		expect(appHeader).toContain('primaryNavigationItems');
		for (const route of ['/admin/users', '/admin/audit-logs', '/admin/mcp', '/admin/backup', '/admin/migration']) {
			expect(navbar).not.toContain(route);
			expect(appHeader).not.toContain(route);
		}
	});

	it('marks Administration active throughout the route family', () => {
		for (const pathname of ['/admin', '/admin/settings', '/admin/users', '/admin/backup', '/admin/migration', '/admin/mcp', '/admin/audit-logs']) {
			expect(isAdministrationPath(pathname)).toBe(true);
			expect(isAdministrationNavItemActive(administrationNavItems[0], pathname)).toBe(pathname === '/admin/settings');
		}
		expect(isAdministrationPath('/projects')).toBe(false);
	});

	it('defines the shared Administration sidebar groups', () => {
		expect(administrationNavGroups.map((group) => group.label)).toEqual(['Platform', 'Operations', 'Integrations', 'Activity']);
		expect(administrationNavItems.map((item) => [item.label, item.href])).toEqual([
			['General', '/admin/settings'],
			['Users', '/admin/users'],
			['Backup', '/admin/backup'],
			['Migration', '/admin/migration'],
			['MCP', '/admin/mcp'],
			['Audit logs', '/admin/audit-logs']
		]);
		expect(administrationNavItemForPath('/admin/users').label).toBe('Users');
		expect(appHeader).toContain('administrationNavItemForPath');
	});

	it('uses the shared compact Administration shell', () => {
		expect(adminLayout).toContain('lg:grid-cols-[12rem_minmax(0,1fr)]');
		expect(adminLayout).toContain('min-w-0 px-3.5 py-3');
		expect(adminLayout).toContain('border-[color:var(--workspace-divider)]');
		expect(adminSidebar).toContain('administrationNavGroups');
		expect(adminSidebar).toContain('border-l-2');
		expect(createProjectPage).not.toContain('AdminSidebar');
		expect(projectLogsPage).not.toContain('AdminSidebar');
	});

	it('keeps redesigned Administration pages full-width and task-first', () => {
		expect(adminSettingsPage).toContain('admin-general-workspace w-full');
		expect(adminSettingsPage).toContain('ConfirmActionDialog');
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
		expect(adminMcpPage).toContain('Observability');
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
	it('keeps Create Project as one four-step route', () => {
		expect(createProjectLayout).toContain('ProjectNewSidebar');
		expect(createProjectLayout).toContain('lg:grid-cols-[12rem_minmax(0,1fr)]');
		for (const label of ['Source', 'Configuration', 'Environment', 'Review']) expect(projectNewSidebar).toContain(`label: '${label}'`);
	});

	it('keeps the existing project configuration leaves before consolidation', () => {
		expect(projectLayout).toContain('ProjectDetailSidebar');
		for (const label of ['Overview', 'Deployments', 'Logs', 'Environment', 'Database', 'General', 'Source', 'Resources', 'Webhook', 'Danger zone']) {
			expect(projectDetailSidebar).toContain(`label: '${label}'`);
		}
		for (const pageSource of [projectSettingsPage, projectSourceSettingsPage, projectResourcesSettingsPage, projectWebhookSettingsPage, projectDangerSettingsPage]) {
			expect(pageSource).toContain('ProjectSettingsSection');
		}
		expect(projectSettingsSection).not.toContain('ProjectEnvironmentSettings');
	});
});
