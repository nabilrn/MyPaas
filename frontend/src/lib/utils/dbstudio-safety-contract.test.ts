import { describe, expect, it } from 'vitest';
import databaseLayout from '../../routes/projects/[id]/database/+layout.svelte?raw';
import databasePage from '../../routes/projects/[id]/database/+page.svelte?raw';

describe('database studio stable safety contract', () => {
	it('keeps insert and delete outside the stable UI boundary', () => {
		expect(databaseLayout).toContain('Write mode supports safe row updates only.');
		expect(databasePage).not.toContain('Insert row');
		expect(databasePage).not.toContain('Delete database row');
		expect(databasePage).not.toContain('api.dbStudio.insert');
		expect(databasePage).not.toContain('api.dbStudio.delete');
	});

	it('uses one live table-scoped string search without filter actions', () => {
		expect(databasePage).toContain('Search rows in this table');
		expect(databasePage).toContain('rowSearchDebounceMs = 250');
		expect(databasePage).toContain('search: rowSearch.trim()');
		expect(databasePage).not.toContain('Apply filters');
		expect(databasePage).not.toContain('Reset');
		expect(databasePage).not.toContain('enumFilters');
	});

	it('keeps safe updates explicit and typed', () => {
		expect(databasePage).toContain('Enable write');
		expect(databasePage).toContain('api.dbStudio.update');
		expect(databasePage).toContain('Set NULL');
		expect(databasePage).toContain('Use database time');
		expect(databasePage).toContain('isEditableMutationColumn');
	});
});
