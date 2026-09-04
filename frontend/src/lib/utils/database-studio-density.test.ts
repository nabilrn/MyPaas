import { describe, expect, it } from 'vitest';
import databaseLayout from '../../routes/projects/[id]/database/+layout.svelte?raw';
import databasePage from '../../routes/projects/[id]/database/+page.svelte?raw';
import settingsWorkspace from '../components/SettingsWorkspace.svelte?raw';

describe('database studio density contract', () => {
	it('uses a bounded split workspace instead of visible pagination controls', () => {
		expect(databasePage).toContain('data-db-studio-workspace');
		expect(databasePage).toContain('h-[calc(100vh-15rem)]');
		expect(databasePage).toContain('data-db-table-scroll');
		expect(databasePage).toContain('data-db-row-scroll');
		expect(databasePage).toContain('overflow-y-auto');
		expect(databasePage).toContain('overflow-auto');
		expect(databasePage).not.toContain("import Pagination from '$components/Pagination.svelte'");
		expect(databasePage).not.toContain('bind:page=');
	});

	it('keeps row fetching incremental while the table header remains sticky', () => {
		expect(databasePage).toContain('const studioBatchSize = 50;');
		expect(databasePage).toContain('await api.dbStudio.rows');
		expect(databasePage).toContain('rows: [...rows.rows, ...nextPage.rows]');
		expect(databasePage).toContain('handleRowViewportScroll');
		expect(databasePage).toContain('Scroll to load more');
		expect(databasePage).toContain('sticky top-0 z-10');
	});

	it('keeps connection metadata and controls in one compact desktop strip', () => {
		expect(databasePage).toContain('data-db-connection-strip');
		expect(databasePage).toContain('lg:flex-nowrap');
		expect(databasePage).toContain('data-db-table-controls');
		expect(databasePage).toContain('data-db-row-toolbar');
	});

	it('uses one workspace surface and stroke-only table selection', () => {
		expect(databaseLayout).toContain('.database-data-shell');
		expect(databaseLayout).toContain('[data-db-studio-workspace]');
		expect(databaseLayout).toContain('background: transparent !important');
		expect(databaseLayout).toContain('button.bg-gray-100');
		expect(databaseLayout).toContain('box-shadow: inset 2px 0 0 var(--app-border-strong)');
		expect(databaseLayout).toContain('tbody tr:hover');
	});

	it('does not render the danger loading state as a bordered surface card', () => {
		expect(settingsWorkspace).toContain("data-settings-section='danger'] :global(.surface)");
		expect(settingsWorkspace).toContain('background: transparent');
		expect(settingsWorkspace).toContain('box-shadow: none');
	});
});
