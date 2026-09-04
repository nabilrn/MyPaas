import { describe, expect, it } from 'vitest';
import adminLayout from '../../routes/admin/+layout.svelte?raw';
import adminSettings from '../../routes/admin/settings/+page.svelte?raw';
import adminBackup from '../../routes/admin/backup/+page.svelte?raw';
import adminMigration from '../../routes/admin/migration/+page.svelte?raw';
import adminMcp from '../../routes/admin/mcp/+page.svelte?raw';
import adminUsers from '../../routes/admin/users/+page.svelte?raw';
import adminAudit from '../../routes/admin/audit-logs/+page.svelte?raw';
import designContract from '../../../DESIGN.md?raw';

describe('administration route spacing contract', () => {
	it('uses the same parent, readable-content, and title top rhythm as project detail', () => {
		expect(adminLayout).toContain('min-w-0 px-3.5 py-3');
		expect(adminLayout).toContain('px-5 pt-4 pb-3');
		expect(adminLayout).toContain('.admin-content > .page-shell > section > h2');
		expect(adminLayout).toContain('padding-inline: 1.25rem');
		expect(adminLayout).toContain('padding-left: 1rem !important');
		expect(adminLayout).toContain('.admin-content > .page-shell > details');
		expect(adminLayout).not.toContain('padding-left: 0 !important');
		expect(adminLayout).not.toContain('padding-right: 0 !important');
	});

	it('keeps every admin child route inside the shared parent-owned header contract', () => {
		for (const route of [adminSettings, adminBackup, adminMigration, adminMcp, adminUsers, adminAudit]) {
			expect(route).toContain('class="page-shell"');
			expect(route).not.toContain('class="px-5 pt-4"');
		}
		expect(adminSettings).toContain('px-4 py-3');
		expect(adminSettings).toContain('max-w-5xl');
		expect(adminBackup).toContain('border-y border-[color:var(--workspace-divider)]');
		expect(adminMigration).toContain('border-[color:var(--workspace-divider)]');
		expect(adminMigration).toContain('max-w-4xl');
		expect(adminMcp).toContain('border-[color:var(--workspace-divider)]');
		expect(adminMcp).toContain('max-w-5xl');
		expect(adminUsers).toContain('TableShell');
		expect(adminAudit).toContain('TableShell');
	});

	it('documents administration as part of the canonical gutter contract', () => {
		expect(designContract).toContain('### Administration gutters');
		expect(designContract).toContain('parent main: `px-3.5 py-3`');
		expect(designContract).toContain('route heading: `px-5`');
		expect(designContract).toContain('body/setting rows: `px-4`');
		expect(designContract).toContain('Compact Administration content');
	});
});
