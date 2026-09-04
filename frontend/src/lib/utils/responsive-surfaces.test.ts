import { describe, expect, it } from 'vitest';
import tableShell from '../components/TableShell.svelte?raw';
import storageMetric from '../components/StorageCapacityMetric.svelte?raw';
import databasePage from '../../routes/projects/[id]/database/+page.svelte?raw';
import databaseSchemaPage from '../../routes/projects/[id]/database/schema/+page.svelte?raw';

describe('responsive data surfaces', () => {
	it('keeps TableShell content horizontally reachable on narrow viewports', () => {
		expect(tableShell).toContain('table-scroll-region');
		expect(tableShell).toContain('overflow-x: auto !important');
		expect(tableShell).toContain('overscroll-behavior-x: contain');
	});

	it('keeps standalone database tables inside horizontal scroll regions', () => {
		expect(databasePage).toContain('data-db-row-scroll');
		expect(databasePage).toContain('overflow-auto');
		expect(databaseSchemaPage).toContain('overflow-x-auto');
	});

	it('keeps storage readable and in the four-column desktop host resource grid', () => {
		expect(storageMetric).toContain('data-storage-capacity');
		expect(storageMetric).toContain('grid-template-columns: repeat(4, minmax(0, 1fr))');
		expect(storageMetric).toContain('h-4 overflow-hidden rounded-sm border');
		expect(storageMetric).toContain('h-9 w-9');
		expect(storageMetric).not.toContain('Windows');
	});
});