import { describe, expect, it } from 'vitest';
import databasePage from '../../routes/projects/[id]/database/+page.svelte?raw';
import settingsWorkspace from '../components/SettingsWorkspace.svelte?raw';

describe('database studio density contract', () => {
	it('uses one shared visible-item limit for table navigation and row data', () => {
		expect(databasePage).toContain('const studioPageSize = 12;');
		expect(databasePage).toContain('let pageSize = studioPageSize;');
		expect(databasePage).toContain('pagedTables = filteredTables.slice');
		expect(databasePage).toContain('bind:page={tablePageIndex}');
		expect(databasePage).toContain('pageSize={studioPageSize}');
		expect(databasePage).toContain('bind:page={pageIndex}');
		expect(databasePage).not.toContain('max-h-[34rem] overflow-auto');
	});

	it('keeps connection metadata and controls in one compact desktop strip', () => {
		expect(databasePage).toContain('data-db-connection-strip');
		expect(databasePage).toContain('lg:flex-nowrap');
		expect(databasePage).toContain('data-db-table-page');
		expect(databasePage).toContain('data-db-row-page');
	});

	it('does not render the danger loading state as a bordered surface card', () => {
		expect(settingsWorkspace).toContain("data-settings-section='danger'] :global(.surface)");
		expect(settingsWorkspace).toContain('background: transparent');
		expect(settingsWorkspace).toContain('box-shadow: none');
	});
});
