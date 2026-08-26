<script lang="ts">
	import { RefreshCw } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { layoutERD } from '$lib/dbstudio/erd';
	import type { DBStudioSchema, DBStudioStatus, DBStudioTable, DBStudioTableDetails } from '$types';

	let status: DBStudioStatus | null = null;
	let schemas: DBStudioSchema[] = [];
	let tables: DBStudioTable[] = [];
	let details: DBStudioTableDetails[] = [];
	let selectedSchema = '';
	let selectedTable = '';
	let loading = true;
	let loadingSchema = false;
	let error = '';

	$: graph = layoutERD(details);
	$: selectedDetails = details.find((item) => item.name === selectedTable && item.schema === selectedSchema) ?? details[0] ?? null;

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			status = await api.dbStudio.status($page.params.id ?? '');
			if (!status.connected) return;
			schemas = await api.dbStudio.schemas($page.params.id ?? '');
			selectedSchema = selectedSchema || schemas[0]?.name || '';
			if (selectedSchema) await loadSchema();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load database schema';
		} finally {
			loading = false;
		}
	}

	async function loadSchema() {
		if (!selectedSchema) return;
		loadingSchema = true;
		error = '';
		try {
			tables = await api.dbStudio.tables($page.params.id ?? '', selectedSchema);
			details = await Promise.all(tables.map((table) => api.dbStudio.tableDetails($page.params.id ?? '', table.schema, table.name)));
			selectedTable = details.some((item) => item.name === selectedTable) ? selectedTable : details[0]?.name || '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load schema metadata';
			details = [];
		} finally {
			loadingSchema = false;
		}
	}

	async function handleSchemaChange(event: Event) {
		selectedSchema = (event.currentTarget as HTMLSelectElement).value;
		selectedTable = '';
		await loadSchema();
	}

	function edgeX(node: { x: number; width: number }) {
		return node.x + node.width / 2;
	}

	function edgeY(node: { y: number; height: number }) {
		return node.y + node.height / 2;
	}
</script>

<svelte:head>
	<title>Schema & ERD · MyPaas</title>
</svelte:head>

{#if loading}
	<div class="surface h-80 animate-pulse"></div>
{:else if error && !status}
	<div class="surface overflow-hidden"><ErrorState title="Could not load schema metadata" message={error} on:retry={() => void load()} /></div>
{:else if !status?.configured}
	<SectionPanel title="Schema & ERD" description="No supported database connection was detected."><EmptyState title="No database connection found." description="Configure PostgreSQL, MySQL, or MariaDB in the project environment first." /></SectionPanel>
{:else if !status.connected}
	<SectionPanel title="Schema & ERD" description="The database connection is configured but unavailable."><EmptyState title="Database unavailable." description={status.message} /></SectionPanel>
{:else}
	<div class="space-y-4">
		<SectionPanel title="Schema & ERD" description="Read-only relationship, index, and constraint metadata for the selected database schema.">
			<svelte:fragment slot="actions"><ActionButton variant="secondary" size="sm" on:click={() => void loadSchema()} loading={loadingSchema} loadingLabel="Refreshing"><RefreshCw slot="icon" class="h-4 w-4" />Refresh</ActionButton></svelte:fragment>
			<div class="max-w-sm">
				<label class="field-label" for="metadata-schema">Schema</label>
				<select id="metadata-schema" class="field w-full font-mono" value={selectedSchema} on:change={handleSchemaChange}>{#each schemas as schema}<option value={schema.name}>{schema.name}</option>{/each}</select>
			</div>
			{#if error}<div class="alert-danger mt-4">{error}</div>{/if}
		</SectionPanel>

		{#if loadingSchema}
			<div class="surface h-96 animate-pulse"></div>
		{:else if details.length === 0}
			<SectionPanel title="Relationships" description="No base tables were found in this schema."><EmptyState title="No tables found." description="Choose another schema or create database tables first." /></SectionPanel>
		{:else}
			<SectionPanel title="Relationship graph" description={`${graph.nodes.length} table${graph.nodes.length === 1 ? '' : 's'} · ${graph.edges.length} foreign-key relation${graph.edges.length === 1 ? '' : 's'}`} contentClass="p-0">
				<div class="overflow-auto bg-gray-50/50 p-3 dark:bg-neutral-950/40">
					<div class="relative" style={`width:${graph.width}px;height:${graph.height}px;min-width:100%`}>
						<svg class="pointer-events-none absolute inset-0" width={graph.width} height={graph.height} aria-hidden="true">
							<defs><marker id="erd-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" class="fill-gray-400 dark:fill-gray-600" /></marker></defs>
							{#each graph.edges as edge (edge.id)}
								<line x1={edgeX(edge.from)} y1={edgeY(edge.from)} x2={edgeX(edge.to)} y2={edgeY(edge.to)} class="stroke-gray-300 dark:stroke-gray-700" stroke-width="1.5" marker-end="url(#erd-arrow)" />
							{/each}
						</svg>
						{#each graph.nodes as node (node.id)}
							<button type="button" class={`absolute overflow-hidden rounded-lg border bg-white text-left shadow-sm transition ${selectedTable === node.detail.name ? 'border-gray-950 ring-1 ring-gray-950 dark:border-white dark:ring-white' : 'border-gray-200 hover:border-gray-400 dark:border-neutral-800 dark:bg-neutral-950 dark:hover:border-gray-600'}`} style={`left:${node.x}px;top:${node.y}px;width:${node.width}px;height:${node.height}px`} on:click={() => (selectedTable = node.detail.name)}>
								<div class="border-b border-gray-100 px-3 py-2 dark:border-neutral-800"><p class="truncate font-mono text-xs text-gray-500 dark:text-gray-400">{node.detail.schema}</p><p class="truncate text-sm font-semibold text-gray-950 dark:text-white">{node.detail.name}</p></div>
								<div class="space-y-1 px-3 py-2">
									{#each node.detail.columns.slice(0, 5) as column}<div class="flex items-center justify-between gap-2 text-[11px]"><span class="truncate font-mono text-gray-700 dark:text-gray-300">{column.primaryKey ? 'PK ' : ''}{column.name}</span><span class="shrink-0 text-gray-400 dark:text-gray-500">{column.dataType}</span></div>{/each}
									{#if node.detail.columns.length > 5}<p class="text-[10px] text-gray-400 dark:text-gray-500">+{node.detail.columns.length - 5} more</p>{/if}
								</div>
							</button>
						{/each}
					</div>
				</div>
			</SectionPanel>

			{#if selectedDetails}
				<div class="grid gap-4 xl:grid-cols-2">
					<SectionPanel title={`${selectedDetails.schema}.${selectedDetails.name}`} description="Columns and foreign-key relationships.">
						<div class="overflow-x-auto"><table class="w-full text-left text-xs"><thead class="text-gray-500 dark:text-gray-400"><tr><th class="pb-2 pr-3">Column</th><th class="pb-2 pr-3">Type</th><th class="pb-2">Flags</th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-neutral-800">{#each selectedDetails.columns as column}<tr><td class="py-2 pr-3 font-mono text-gray-950 dark:text-white">{column.name}</td><td class="py-2 pr-3 font-mono text-gray-600 dark:text-gray-300">{column.dataType}</td><td class="py-2 text-gray-500 dark:text-gray-400">{[column.primaryKey ? 'PK' : '', column.nullable ? 'nullable' : 'required', column.autoGenerated ? 'generated' : ''].filter(Boolean).join(' · ')}</td></tr>{/each}</tbody></table></div>
						<div class="mt-5 border-t border-gray-100 pt-4 dark:border-neutral-800"><p class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Foreign keys</p>{#if selectedDetails.foreignKeys.length === 0}<p class="text-sm text-gray-500 dark:text-gray-400">No foreign keys.</p>{:else}<div class="space-y-2">{#each selectedDetails.foreignKeys as foreignKey}<div class="rounded-md border border-gray-200 p-3 text-xs dark:border-neutral-800"><p class="font-mono text-gray-950 dark:text-white">{foreignKey.column} → {foreignKey.referencedSchema}.{foreignKey.referencedTable}.{foreignKey.referencedColumn}</p><p class="mt-1 text-gray-500 dark:text-gray-400">{foreignKey.name} · update {foreignKey.onUpdate} · delete {foreignKey.onDelete}</p></div>{/each}</div>{/if}</div>
					</SectionPanel>

					<div class="space-y-4">
						<SectionPanel title="Indexes" description="Primary, unique, and secondary indexes discovered from database metadata.">{#if selectedDetails.indexes.length === 0}<p class="text-sm text-gray-500 dark:text-gray-400">No indexes reported.</p>{:else}<div class="space-y-2">{#each selectedDetails.indexes as index}<div class="rounded-md border border-gray-200 p-3 dark:border-neutral-800"><div class="flex flex-wrap items-center justify-between gap-2"><p class="font-mono text-xs font-medium text-gray-950 dark:text-white">{index.name}</p><p class="text-[11px] text-gray-500 dark:text-gray-400">{[index.primary ? 'PRIMARY' : '', index.unique ? 'UNIQUE' : '', index.method].filter(Boolean).join(' · ')}</p></div><p class="mt-1 font-mono text-xs text-gray-600 dark:text-gray-300">{index.columns.join(', ') || 'expression index'}</p></div>{/each}</div>{/if}</SectionPanel>
						<SectionPanel title="Constraints" description="Database-enforced primary, unique, foreign-key, and check constraints.">{#if selectedDetails.constraints.length === 0}<p class="text-sm text-gray-500 dark:text-gray-400">No constraints reported.</p>{:else}<div class="space-y-2">{#each selectedDetails.constraints as constraint}<div class="rounded-md border border-gray-200 p-3 dark:border-neutral-800"><div class="flex flex-wrap items-center justify-between gap-2"><p class="font-mono text-xs font-medium text-gray-950 dark:text-white">{constraint.name}</p><p class="text-[11px] text-gray-500 dark:text-gray-400">{constraint.type}</p></div>{#if constraint.columns.length > 0}<p class="mt-1 font-mono text-xs text-gray-600 dark:text-gray-300">{constraint.columns.join(', ')}</p>{/if}{#if constraint.definition}<p class="mt-1 break-words font-mono text-[11px] text-gray-500 dark:text-gray-400">{constraint.definition}</p>{/if}</div>{/each}</div>{/if}</SectionPanel>
					</div>
				</div>
			{/if}
		{/if}
	</div>
{/if}
