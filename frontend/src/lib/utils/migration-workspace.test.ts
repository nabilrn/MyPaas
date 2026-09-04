import { describe, expect, it } from 'vitest';
import migrationPage from '../../routes/admin/migration/+page.svelte?raw';
import migrationIllustration from '../components/MigrationTransferIllustration.svelte?raw';

describe('migration workspace', () => {
	it('uses the full administration canvas with a real transfer workflow', () => {
		expect(migrationPage).toContain('migration-workspace w-full');
		expect(migrationPage).not.toContain('max-w-4xl');
		expect(migrationPage).toContain('Move MyPaaS to another VM');
		expect(migrationPage).toContain('Captured state');
		expect(migrationPage).toContain('Runtime safety');
		expect(migrationPage).toContain('ConfirmActionDialog');
		expect(migrationPage).toContain('MigrationTransferIllustration');
	});

	it('keeps the VM transfer illustration lightweight and token-driven', () => {
		expect(migrationIllustration).toContain('<svg');
		expect(migrationIllustration).toContain('Current VM');
		expect(migrationIllustration).toContain('Migration package');
		expect(migrationIllustration).toContain('New VM');
		expect(migrationIllustration).toContain('stroke="currentColor"');
		expect(migrationIllustration).not.toContain('<linearGradient');
		expect(migrationIllustration).not.toContain('<image');
	});
});
