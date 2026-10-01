import { describe, expect, it } from 'vitest';
import tableShell from '../components/TableShell.svelte?raw';
import hostResourceOverview from '../components/HostResourceOverview.svelte?raw';
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

	it('keeps merged host resources compact and responsive', () => {
		expect(hostResourceOverview).toContain('data-host-resource-overview');
		expect(hostResourceOverview).toContain('grid-cols-2');
		expect(hostResourceOverview).toContain('xl:grid-cols-4');
		expect(hostResourceOverview).toContain('h-36 overflow-hidden');
		expect(hostResourceOverview).toContain('data-storage-capacity');
	});
});