<script lang="ts">
	import { Clock3, Lock, Pencil, RefreshCw, Save, Unlock, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import {
		activeDatabaseNowColumns,
		buildTouchedValues,
		isEditableMutationColumn,
		isNumericMutationColumn,
		isTemporalColumn,
		mutationValueKind
	} from '$lib/dbstudio/mutation';
	import type { DBStudioColumn, DBStudioRowPage, DBStudioSchema, DBStudioStatus, DBStudioTable } from '$types';

	const studioBatchSize = 50;
	const rowSearchDebounceMs = 250;

	let status: DBStudioStatus | null = null;
	let schemas: DBStudioSchema[] = [];
	let tables: DBStudioTable[] = [];
	let rows: DBStudioRowPage | null = null;
	let selectedSchema = '';
	let selectedTable = '';
	let tableSearch = '';
	let rowSearch = '';
	let rowSearchTimer: number | null = null;
	let searchPending = false;
	let loadingTables = false;
	let loadingRows = false;
	let loadingMoreRows = false;
	let rowsRequest = 0;
	let error = '';
	let tableError = '';
	let rowsError = '';
	let startingWrite = false;
	let revokingWrite = false;
	let mutating = false;
	let editing = false;
	let selectedRow: Record<string, unknown> | null = null;
	let draftValues: Record<string, string> = {};
	let touchedValues: Record<string, boolean> = {};
	let nullValues: Record<string, boolean> = {};
	let databaseNowValues: Record<string, boolean> = {};
	let now = Date.now();

	$: projectId = $page.params.id ?? '';
	$: connection = status?.connection ?? null;
	$: writeAccess = status?.writeAccess ?? null;
	$: writeExpiresAt = writeAccess ? new Date(writeAccess.expiresAt).getTime() : 0;
	$: writeActive = Boolean(writeAccess && writeExpiresAt > now);
	$: writeRemaining = writeActive ? formatDuration(writeExpiresAt - now) : '';
	$: columns = rows?.columns ?? [];
	$: primaryColumns = columns.filter((column) => column.primaryKey);
	$: editableColumns = columns.filter(isEditableMutationColumn);
	$: filteredTables = tables.filter((table) => table.name.toLowerCase().includes(tableSearch.trim().toLowerCase()));
	$: selectedTableLabel = selectedSchema && selectedTable ? `${selectedSchema}.${selectedTable}` : 'No table selected';
	$: mutationHasChanges = Object.values(touchedValues).some(Boolean) || Object.values(databaseNowValues).some(Boolean);
	$: canEditSelectedTable = writeActive && primaryColumns.length > 0 && editableColumns.length > 0;

	onMount(() => {
		void load();
		const timer = window.setInterval(() => {
			now = Date.now();
		}, 1000);
		return () => {
			window.clearInterval(timer);
			if (rowSearchTimer !== null) window.clearTimeout(rowSearchTimer);
		};
	});

	async function load() {
		error = '';
		try {
			status = await api.dbStudio.status(projectId);
			if (!status.connected) return;
			schemas = await api.dbStudio.schemas(projectId);
			selectedSchema = selectedSchema || schemas[0]?.name || '';
			if (selectedSchema) await loadTables();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load database studio';
		}
	}

	async function loadTables() {
		if (!selectedSchema) return;
		loadingTables = true;
		tableError = '';
		try {
			tables = await api.dbStudio.tables(projectId, selectedSchema);
			const previousTable = selectedTable;
			selectedTable = tables.some((table) => table.name === selectedTable) ? selectedTable : tables[0]?.name || '';
			if (selectedTable !== previousTable) resetRowSearch();
			if (selectedTable) await loadRows(true);
			else rows = null;
		} catch (err) {
			tableError = err instanceof Error ? err.message : 'Failed to load tables';
		} finally {
			loadingTables = false;
		}
	}

	async function loadRows(reset = true) {
		if (!selectedSchema || !selectedTable) return;
		if (!reset && (searchPending || loadingRows || loadingMoreRows || !rows?.hasMore)) return;
		const requestId = ++rowsRequest;
		if (reset) {
			loadingRows = true;
			rowsError = '';
		} else {
			loadingMoreRows = true;
		}
		try {
			const offset = reset ? 0 : (rows?.rows.length ?? 0);
			const nextPage = await api.dbStudio.rows(projectId, selectedSchema, selectedTable, studioBatchSize, offset, {
				search: rowSearch.trim()
			});
			if (requestId !== rowsRequest) return;
			if (reset || !rows) {
				rows = nextPage;
			} else {
				rows = {
					...nextPage,
					offset: 0,
					rows: [...rows.rows, ...nextPage.rows]
				};
			}
		} catch (err) {
			if (requestId !== rowsRequest) return;
			rowsError = err instanceof Error ? err.message : 'Failed to load rows';
		} finally {
			if (requestId === rowsRequest) {
				loadingRows = false;
				loadingMoreRows = false;
			}
		}
	}

	function handleRowViewportScroll(event: Event) {
		const viewport = event.currentTarget as HTMLElement;
		if (viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight > 180) return;
		void loadRows(false);
	}

	async function handleSchemaChange(event: Event) {
		selectedSchema = (event.currentTarget as HTMLSelectElement).value;
		selectedTable = '';
		rows = null;
		resetRowSearch();
		await loadTables();
	}

	function handleTableSearch(event: Event) {
		tableSearch = (event.currentTarget as HTMLInputElement).value;
	}

	function handleRowSearch(event: Event) {
		rowSearch = (event.currentTarget as HTMLInputElement).value;
		searchPending = true;
		if (rowSearchTimer !== null) window.clearTimeout(rowSearchTimer);
		rowSearchTimer = window.setTimeout(() => {
			rowSearchTimer = null;
			searchPending = false;
			void loadRows(true);
		}, rowSearchDebounceMs);
	}

	async function chooseTable(table: DBStudioTable) {
		if (selectedTable === table.name && selectedSchema === table.schema) return;
		selectedSchema = table.schema;
		selectedTable = table.name;
		rows = null;
		resetRowSearch();
		await loadRows(true);
	}

	function resetRowSearch() {
		if (rowSearchTimer !== null) {
			window.clearTimeout(rowSearchTimer);
			rowSearchTimer = null;
		}
		searchPending = false;
		rowSearch = '';
	}

	async function startWriteMode() {
		startingWrite = true;
		try {
			const session = await api.dbStudio.startWriteSession(projectId, 15);
			status = status ? { ...status, writeAccess: session } : await api.dbStudio.status(projectId);
			toast.success('Write mode enabled for 15 minutes');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to enable write mode');
		} finally {
			startingWrite = false;
		}
	}

	async function revokeWriteMode() {
		if (!writeAccess) return;
		revokingWrite = true;
		try {
			await api.dbStudio.revokeWriteSession(projectId, writeAccess.id);
			if (status) status = { ...status, writeAccess: null };
			closeEdit();
			toast.success('Write mode revoked');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to revoke write mode');
		} finally {
			revokingWrite = false;
		}
	}

	function resetMutationDraft() {
		touchedValues = {};
		nullValues = {};
		databaseNowValues = {};
	}

	function openEdit(row: Record<string, unknown>) {
		editing = true;
		selectedRow = row;
		draftValues = Object.fromEntries(editableColumns.map((column) => [column.name, valueToInput(row[column.name], column)]));
		resetMutationDraft();
	}

	function closeEdit() {
		editing = false;
		selectedRow = null;
		draftValues = {};
		resetMutationDraft();
	}

	function setMutationValue(name: string, value: string) {
		draftValues = { ...draftValues, [name]: value };
		touchedValues = { ...touchedValues, [name]: true };
		nullValues = { ...nullValues, [name]: false };
		databaseNowValues = { ...databaseNowValues, [name]: false };
	}

	function useDatabaseNow(name: string) {
		databaseNowValues = { ...databaseNowValues, [name]: true };
		touchedValues = { ...touchedValues, [name]: false };
		nullValues = { ...nullValues, [name]: false };
	}

	function useNull(name: string) {
		touchedValues = { ...touchedValues, [name]: true };
		nullValues = { ...nullValues, [name]: true };
		databaseNowValues = { ...databaseNowValues, [name]: false };
	}

	function keepExisting(name: string) {
		const column = columns.find((item) => item.name === name);
		draftValues = { ...draftValues, [name]: valueToInput(selectedRow?.[name], column) };
		touchedValues = { ...touchedValues, [name]: false };
		nullValues = { ...nullValues, [name]: false };
		databaseNowValues = { ...databaseNowValues, [name]: false };
	}

	async function submitMutation() {
		if (!editing || !selectedRow || !selectedSchema || !selectedTable || !mutationHasChanges) return;
		mutating = true;
		const payload = {
			schema: selectedSchema,
			table: selectedTable,
			values: buildTouchedValues(draftValues, touchedValues, nullValues),
			databaseNow: activeDatabaseNowColumns(databaseNowValues),
			primaryKey: primaryKeyPayload(selectedRow)
		};
		try {
			await api.dbStudio.update(projectId, payload);
			toast.success('Database row saved');
			closeEdit();
			await loadRows(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save database row');
		} finally {
			mutating = false;
		}
	}

	function primaryKeyPayload(row: Record<string, unknown>) {
		return Object.fromEntries(primaryColumns.map((column) => [column.name, row[column.name]]));
	}

	function valueToInput(value: unknown, column?: DBStudioColumn) {
		if (value === null || value === undefined) return '';
		if (column && mutationValueKind(column) === 'boolean') {
			return value === true || value === 1 || String(value).toLowerCase() === 'true' ? 'true' : 'false';
		}
		if (typeof value === 'object') return JSON.stringify(value);
		return String(value);
	}

	function formatCell(value: unknown) {
		if (value === null || value === undefined) return 'NULL';
		if (typeof value === 'object') return JSON.stringify(value);
		return String(value);
	}

	function formatDuration(ms: number) {
		const total = Math.max(0, Math.floor(ms / 1000));
		const minutes = Math.floor(total / 60);
		const seconds = total % 60;
		return `${minutes}m ${seconds.toString().padStart(2, '0')}s`;
	}

	function driverLabel(value: string) {
		if (value === 'postgres') return 'PostgreSQL';
		if (value === 'mariadb') return 'MariaDB';
		if (value === 'sqlite') return 'SQLite';
		return 'MySQL';
	}
</script>

{#if error}
	<div class="workspace-section">
		<ErrorState title="Could not load Database Studio" message={error} on:retry={() => void load()} />
	</div>
{:else if status && !status.configured}
	<SectionPanel title="Connection" description="No supported database connection was detected from this project's environment variables.">
		<EmptyState title="No database connection found." description="Add DATABASE_URL, a SQLite path setting, or DB_HOST, DB_PORT, DB_NAME, DB_USER, and DB_PASSWORD in Environment, then redeploy or refresh this page." />
	</SectionPanel>
{:else if status}
	<div class="border-b border-[color:var(--workspace-divider)]" data-db-connection-strip>
		<div class="flex min-w-0 flex-wrap items-center gap-x-5 gap-y-2 px-4 py-2.5 lg:flex-nowrap">
			<div class="shrink-0 pr-1"><p class="text-sm font-semibold text-gray-950 dark:text-white">Connection</p></div>
			<div class="inline-flex shrink-0 items-center gap-2"><span class="metric-label">Status</span><span class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-800 dark:text-gray-200"><span class="status-dot {status.connected ? 'bg-emerald-500' : 'bg-red-500'}"></span>{status.connected ? 'Connected' : 'Unavailable'}</span></div>
			<div class="inline-flex shrink-0 items-center gap-2"><span class="metric-label">Driver</span><span class="font-mono text-sm text-gray-950 dark:text-white">{connection ? driverLabel(connection.driver) : '—'}</span></div>
			<div class="flex min-w-0 flex-1 items-center gap-2"><span class="metric-label shrink-0">Database</span><span class="min-w-0 truncate font-mono text-sm text-gray-950 dark:text-white" title={connection?.database ?? ''}>{connection?.database ?? '—'}</span></div>
			<div class="inline-flex shrink-0 items-center gap-2"><span class="metric-label">Write</span><span class="inline-flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-gray-300"><span class="status-dot {writeActive ? 'bg-amber-500' : 'bg-gray-400 dark:bg-gray-500'}"></span>{writeActive ? `Active · ${writeRemaining}` : 'Read-only'}</span></div>
			<div class="ml-auto flex shrink-0 items-center gap-2">
				<ActionButton variant="secondary" size="sm" on:click={() => void load()} disabled={loadingRows || loadingTables}><RefreshCw slot="icon" class="h-4 w-4" />Refresh</ActionButton>
				{#if writeActive}
					<ActionButton variant="ghostDanger" size="sm" on:click={revokeWriteMode} loading={revokingWrite} loadingLabel="Revoking"><Lock slot="icon" class="h-4 w-4" />Disable write</ActionButton>
				{:else}
					<ActionButton variant="primary" size="sm" on:click={startWriteMode} loading={startingWrite} loadingLabel="Enabling"><Unlock slot="icon" class="h-4 w-4" />Enable write</ActionButton>
				{/if}
			</div>
		</div>
		{#if !status.connected}<div class="border-t border-red-100 bg-red-50 px-4 py-2.5 text-sm text-red-700 dark:border-red-950 dark:bg-red-950/20 dark:text-red-300">{status.message}</div>{/if}
	</div>

	{#if status.connected}
		<div class="grid h-[calc(100vh-15rem)] min-h-[30rem] min-w-0 overflow-hidden border-b border-[color:var(--workspace-divider)] bg-white dark:bg-neutral-950 lg:grid-cols-[16rem_minmax(0,1fr)]" data-db-studio-workspace>
			<aside class="flex min-h-0 min-w-0 flex-col border-b border-[color:var(--workspace-divider)] lg:border-b-0 lg:border-r">
				<div class="shrink-0 border-b border-[color:var(--workspace-divider)] px-3 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Tables</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Choose one table at a time.</p>
				</div>
				<div class="shrink-0 border-b border-[color:var(--workspace-divider)] p-3" data-db-table-controls>
					<label class="field-label" for="schema">Schema</label>
					<select id="schema" value={selectedSchema} on:change={handleSchemaChange} class="field w-full">{#each schemas as schema}<option value={schema.name}>{schema.name}</option>{/each}</select>
					<label class="field-label mt-3" for="table-search">Search tables</label>
					<input id="table-search" value={tableSearch} placeholder="Table name" class="field w-full" on:input={handleTableSearch} />
				</div>
				<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain p-2" data-db-table-scroll>
					{#if tableError}
						<ErrorState message={tableError} on:retry={() => void loadTables()} />
					{:else if loadingTables}
						<div class="space-y-1" aria-label="Loading tables">{#each [1, 2, 3, 4, 5, 6] as _}<div class="h-9 bg-gray-50 dark:bg-neutral-900"></div>{/each}</div>
					{:else if filteredTables.length === 0}
						<EmptyState title="No tables found." compact />
					{:else}
						{#each filteredTables as table}
							<button type="button" class="app-focus flex min-h-9 w-full items-center rounded-md px-3 py-2 text-left font-mono text-sm transition-colors {selectedSchema === table.schema && selectedTable === table.name ? 'bg-gray-100 font-medium text-gray-950 dark:bg-neutral-800 dark:text-white' : 'text-gray-600 hover:bg-gray-50 hover:text-gray-950 dark:text-gray-300 dark:hover:bg-neutral-900 dark:hover:text-white'}" on:click={() => void chooseTable(table)}>
								<span class="truncate">{table.name}</span>
							</button>
						{/each}
					{/if}
				</div>
			</aside>

			<section class="flex min-h-0 min-w-0 flex-col">
				<div class="shrink-0 border-b border-[color:var(--workspace-divider)] px-4 py-2.5">
					<h2 class="truncate text-sm font-semibold text-gray-950 dark:text-white" title={selectedTableLabel}>{selectedTableLabel}</h2>
					<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
						{primaryColumns.length > 0
							? `Primary key: ${primaryColumns.map((column) => column.name).join(', ')}`
							: 'Update disabled because this table has no primary key.'}
					</p>
				</div>

				<div class="shrink-0 border-b border-[color:var(--workspace-divider)] px-3 py-2" data-db-row-toolbar>
					<div class="flex w-full items-center gap-2">
						<input
							value={rowSearch}
							class="field min-w-[14rem] flex-1"
							placeholder="Search rows in this table"
							aria-label="Search database rows"
							on:input={handleRowSearch}
						/>
						<ActionButton variant="secondary" size="sm" on:click={() => void loadRows(true)} disabled={!selectedTable || loadingRows}><RefreshCw slot="icon" class="h-4 w-4" />Refresh rows</ActionButton>
					</div>
				</div>

				<div class="min-h-0 flex-1 overflow-auto overscroll-contain" on:scroll={handleRowViewportScroll} data-db-row-scroll>
					{#if rowsError && !loadingRows}
						<ErrorState message={rowsError} on:retry={() => void loadRows(true)} />
					{:else if loadingRows && !rows}
						<div class="space-y-px p-2" aria-label="Loading rows">{#each [1, 2, 3, 4, 5, 6, 7, 8] as _}<div class="h-10 bg-gray-50 dark:bg-neutral-900"></div>{/each}</div>
					{:else if !rows || rows.rows.length === 0}
						<EmptyState
							title={rowSearch.trim() ? 'No rows match this search.' : 'No rows in this table.'}
							description={rowSearch.trim() ? 'Search is limited to the currently selected table.' : ''}
							compact
						/>
					{:else}
						<table class="data-table min-w-full" data-db-row-table>
							<thead class="sticky top-0 z-10 bg-white dark:bg-neutral-950"><tr>{#each columns as column}<th><span class="inline-flex items-center gap-1.5">{column.name}{#if column.primaryKey}<span class="font-mono text-xs font-semibold text-amber-700 dark:text-amber-300">PK</span>{/if}{#if (column.enumValues?.length ?? 0) > 0}<span class="font-mono text-xs text-gray-400 dark:text-gray-500">ENUM</span>{/if}</span></th>{/each}<th class="w-16 text-right">Actions</th></tr></thead>
							<tbody>
								{#each rows.rows as row, rowIndex}
									<tr>
										{#each columns as column}<td class="max-w-72 truncate font-mono text-xs text-gray-700 dark:text-gray-200" title={formatCell(row[column.name])}>{formatCell(row[column.name])}</td>{/each}
										<td class="text-right">
											<IconButton
												label={`Edit database row ${rowIndex + 1}`}
												variant="ghost"
												on:click={() => openEdit(row)}
												disabled={!canEditSelectedTable}
											><Pencil class="h-4 w-4" aria-hidden="true" /></IconButton>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
						{#if loadingMoreRows}<div class="border-t border-[color:var(--workspace-divider)] px-4 py-2 text-center text-xs text-gray-500 dark:text-gray-400">Loading more rows…</div>{/if}
					{/if}
				</div>
				<div class="flex min-h-9 shrink-0 items-center justify-between gap-3 border-t border-[color:var(--workspace-divider)] px-3 text-xs text-gray-500 dark:text-gray-400" data-db-row-status>
					<span>{rows?.rows.length ?? 0} row{(rows?.rows.length ?? 0) === 1 ? '' : 's'} loaded</span>
					{#if rows?.hasMore}<span>Scroll to load more</span>{/if}
				</div>
			</section>
		</div>
	{/if}
{/if}

{#if editing}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-gray-950/50 p-4">
		<div class="overlay w-full max-w-2xl overflow-hidden">
			<div class="panel-header flex items-start justify-between gap-3">
				<div><h2 class="panel-title">Edit row</h2><p class="panel-description font-mono">{selectedTableLabel}</p></div>
				<IconButton label="Close database row dialog" variant="ghost" on:click={closeEdit} disabled={mutating}><X class="h-4 w-4" aria-hidden="true" /></IconButton>
			</div>
			<div class="max-h-[70vh] overflow-auto p-4">
				<div class="mb-4 flex items-start gap-2 border-b border-[color:var(--workspace-divider)] pb-3 text-xs text-gray-500 dark:text-gray-400">
					<Clock3 class="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
					<span>Stable write mode only edits recognized scalar columns. Primary keys, generated columns, and complex types stay read-only. Temporal columns can use database time but not manual browser time.</span>
				</div>

				{#if editableColumns.length === 0}
					<EmptyState title="No safe editable columns." description="This table only contains generated, primary-key, or unsupported column types." compact />
				{:else}
					<div class="grid gap-4 sm:grid-cols-2">
						{#each editableColumns as column}
							<div class="min-w-0">
								<label class="field-label" for={`db-value-${column.name}`}>
									{column.name} <span class="font-normal text-gray-400">({column.dataType})</span>
								</label>

								{#if databaseNowValues[column.name]}
									<div class="field flex w-full items-center justify-between gap-3 font-mono text-sm">
										<span class="inline-flex min-w-0 items-center gap-2"><Clock3 class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />Database time</span>
										<button type="button" class="app-focus shrink-0 text-xs font-sans font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white" on:click={() => keepExisting(column.name)}>Keep existing</button>
									</div>
								{:else if nullValues[column.name]}
									<div class="field flex w-full items-center justify-between gap-3 font-mono text-sm">
										<span>NULL</span>
										<button type="button" class="app-focus shrink-0 text-xs font-sans font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white" on:click={() => keepExisting(column.name)}>Keep existing</button>
									</div>
								{:else if isTemporalColumn(column)}
									<div class="field flex w-full items-center text-sm text-gray-500 dark:text-gray-400">Keep existing</div>
								{:else if mutationValueKind(column) === 'enum'}
									<select
										id={`db-value-${column.name}`}
										class="field w-full font-mono"
										value={draftValues[column.name] ?? ''}
										on:change={(event) => setMutationValue(column.name, (event.currentTarget as HTMLSelectElement).value)}
									>
										{#each column.enumValues ?? [] as value}<option {value}>{value}</option>{/each}
									</select>
								{:else if mutationValueKind(column) === 'boolean'}
									<select
										id={`db-value-${column.name}`}
										class="field w-full font-mono"
										value={(draftValues[column.name] ?? '').toLowerCase()}
										on:change={(event) => setMutationValue(column.name, (event.currentTarget as HTMLSelectElement).value)}
									>
										<option value="true">true</option>
										<option value="false">false</option>
									</select>
								{:else}
									<input
										id={`db-value-${column.name}`}
										value={draftValues[column.name] ?? ''}
										class="field w-full font-mono"
										inputmode={isNumericMutationColumn(column) ? 'decimal' : undefined}
										on:input={(event) => setMutationValue(column.name, (event.currentTarget as HTMLInputElement).value)}
									/>
								{/if}

								<div class="mt-1.5 flex min-h-5 flex-wrap items-center gap-x-3 gap-y-1 text-xs">
									{#if isTemporalColumn(column) && !databaseNowValues[column.name]}
										<button type="button" class="app-focus font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white" on:click={() => useDatabaseNow(column.name)}>Use database time</button>
									{/if}
									{#if column.nullable && !nullValues[column.name]}
										<button type="button" class="app-focus font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white" on:click={() => useNull(column.name)}>Set NULL</button>
									{/if}
									{#if touchedValues[column.name] || databaseNowValues[column.name]}
										<button type="button" class="app-focus font-medium text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white" on:click={() => keepExisting(column.name)}>Keep existing</button>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</div>
			<div class="flex items-center justify-between gap-3 border-t border-gray-100 p-4 dark:border-neutral-800">
				<div class="min-w-0 text-xs text-gray-500 dark:text-gray-400">
					{#if !mutationHasChanges}Change a supported field, choose database time, or set NULL before saving.{/if}
				</div>
				<div class="flex shrink-0 gap-2">
					<ActionButton variant="ghost" on:click={closeEdit} disabled={mutating}><X slot="icon" class="h-4 w-4" />Cancel</ActionButton>
					<ActionButton variant="primary" on:click={() => void submitMutation()} disabled={!mutationHasChanges} loading={mutating} loadingLabel="Saving"><Save slot="icon" class="h-4 w-4" />Save row</ActionButton>
				</div>
			</div>
		</div>
	</div>
{/if}
