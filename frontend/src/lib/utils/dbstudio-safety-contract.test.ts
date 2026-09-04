import { describe, expect, it } from 'vitest';
import databaseLayout from '../../routes/projects/[id]/database/+layout.svelte?raw';
import databasePage from '../../routes/projects/[id]/database/+page.svelte?raw';

describe('database studio safety contract', () => {
	it('keeps row deletion outside the current UI maturity boundary', () => {
		expect(databaseLayout).toContain('Write mode supports insert and update only.');
		expect(databaseLayout).toContain("button[aria-label^='Delete database row']");
		expect(databaseLayout).toContain('display: none !important');
	});

	it('keeps write mode explicit and database-time mutation opt-in', () => {
		expect(databasePage).toContain('Enable write');
		expect(databasePage).toContain('api.dbStudio.insert');
		expect(databasePage).toContain('api.dbStudio.update');
		expect(databasePage).toContain('let databaseNowValues: Record<string, boolean> = {}');
		expect(databasePage).toContain('function useDatabaseNow(name: string)');
	});
});
