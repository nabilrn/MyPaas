<script lang="ts">
	import { Download, LocateFixed, Minus, Plus, RefreshCw, Search } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { api } from '$api';
	import { layoutERD, type ERDDirection, type ERDEdge, type ERDNode } from '$lib/dbstudio/erd';
	import {
		buildERDSVG,
		buildSchemaSQL,
		downloadPDF,
		downloadPNG,
		downloadSQL,
		downloadSVG,
		presetRatio,
		relationPath,
		type ERDPagePreset
	} from '$lib/dbstudio/export';
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
	let direction: ERDDirection = 'LR';
	const density = 'comfortable' as const;
	let showDataTypes = true;
	let showRelationLabels = true;
	let tableSearch = '';
	let zoom = 1;
	let pagePreset: ERDPagePreset = 'a4-landscape';
	let exporting = '';

	const pagePresets: Array<{ value: ERDPagePreset; label: string }> = [
		{ value: '1:1', label: '1:1' },
		{ value: '4:3', label: '4:3' },
		{ value: '3:2', label: '3:2' },
		{ value: '16:9', label: '16:9' },
		{ value: 'a4-landscape', label: 'A4' }
	];
	const exportFormats = ['svg', 'png', 'pdf', 'sql'] as const;

	$: graph = layoutERD(details, { direction, density });
	$: selectedDetails = details.find((item) => item.name === selectedTable && item.schema === selectedSchema) ?? details[0] ?? null;
	$: searchTerm = tableSearch.trim().toLowerCase();
	$: matchingNodeIDs = new Set(graph.nodes.filter((node) => !searchTerm || `${node.detail.schema}.${node.detail.name}`.toLowerCase().includes(searchTerm)).map((node) => node.id));

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
			fitGraph();
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
		tableSearch = '';
		await loadSchema();
	}

	function edgeGeometry(edge: ERDEdge) {
		return relationPath(edge, graph, direction, density);
	}

	function nodeClasses(node: ERDNode) {
		const selected = selectedTable === node.detail.name;
		const muted = searchTerm && !matchingNodeIDs.has(node.id);
		return `border-gray-300 bg-white dark:border-neutral-700 dark:bg-neutral-950 ${selected ? 'ring-2 ring-gray-950 dark:ring-white' : 'hover:border-gray-500 dark:hover:border-gray-500'} ${muted ? 'opacity-25' : 'opacity-100'}`;
	}

	function visibleColumnCount(node: ERDNode) {
		return Math.min(7, node.detail.columns.length);
	}

	function fitGraph() {
		const targetWidth = 1120;
		const ratio = presetRatio(pagePreset);
		const targetHeight = ratio ? targetWidth / ratio : 700;
		zoom = Math.min(1.15, Math.max(0.45, Math.min(targetWidth / Math.max(graph.width, 1), targetHeight / Math.max(graph.height, 1))));
	}

	function changeZoom(delta: number) {
		zoom = Math.min(1.5, Math.max(0.4, Number((zoom + delta).toFixed(2))));
	}

	function focusFirstMatch() {
		const match = graph.nodes.find((node) => matchingNodeIDs.has(node.id));
		if (match) selectedTable = match.detail.name;
	}

	function choosePreset(preset: ERDPagePreset) {
		pagePreset = preset;
		fitGraph();
	}

	function exportSVGSource() {
		return buildERDSVG(graph, direction, {
			preset: pagePreset,
			density,
			showDataTypes,
			showRelationLabels,
			title: `${selectedSchema} schema`
		});
	}

	async function exportDiagram(format: (typeof exportFormats)[number]) {
		if (exporting) return;
		exporting = format;
		error = '';
		const base = `${selectedSchema || 'database'}-erd`;
		try {
			if (format === 'sql') {
				const driver = status?.connection?.driver;
				if (!driver) throw new Error('Database driver is unavailable');
				downloadSQL(buildSchemaSQL(details, driver), `${selectedSchema || 'schema'}.sql`);
				return;
			}
			const svg = exportSVGSource();
			if (format === 'svg') downloadSVG(svg, `${base}.svg`);
			if (format === 'png') await downloadPNG(svg, `${base}.png`);
			if (format === 'pdf') await downloadPDF(svg, `${base}.pdf`, pagePreset);
		} catch (err) {
			error = err instanceof Error ? err.message : `Failed to export ${format.toUpperCase()}`;
		} finally {
			exporting = '';
		}
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
	<SectionPanel title="Schema & ERD"><EmptyState title="No database connection found." description="Configure PostgreSQL, MySQL, or MariaDB first." /></SectionPanel>
{:else if !status.connected}
	<SectionPanel title="Schema & ERD"><EmptyState title="Database unavailable." description={status.message} /></SectionPanel>
{:else}
	<div class="space-y-4">
		<div class="surface flex flex-wrap items-end gap-3 p-3">
			<div class="min-w-40">
				<label class="field-label" for="metadata-schema">Schema</label>
				<select id="metadata-schema" class="field w-full font-mono" value={selectedSchema} on:change={handleSchemaChange}>{#each schemas as schema}<option value={schema.name}>{schema.name}</option>{/each}</select>
			</div>
			<div class="min-w-52 flex-1">
				<label class="field-label" for="erd-search">Find table</label>
				<div class="relative">
					<Search class="pointer-events-none absolute left-3 top-1/2 z-10 h-4 w-4 -translate-y-1/2 text-gray-400" />
					<input id="erd-search" class="field w-full !pl-9 font-mono" placeholder="users, audit_logs…" bind:value={tableSearch} on:keydown={(event) => event.key === 'Enter' && focusFirstMatch()} />
				</div>
			</div>
			<div>
				<label class="field-label" for="erd-direction">Direction</label>
				<select id="erd-direction" class="field" bind:value={direction} on:change={fitGraph}><option value="LR">Landscape flow</option><option value="TB">Portrait flow</option></select>
			</div>
			<div>
				<p class="field-label">Page ratio</p>
				<div class="flex overflow-hidden rounded-md border border-gray-200 bg-white dark:border-neutral-800 dark:bg-neutral-950">
					{#each pagePresets as preset}
						<button type="button" class={`border-r border-gray-200 px-2.5 py-2 text-[11px] last:border-r-0 dark:border-neutral-800 ${pagePreset === preset.value ? 'bg-gray-950 text-white dark:bg-white dark:text-gray-950' : 'text-gray-600 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-neutral-900'}`} on:click={() => choosePreset(preset.value)}>{preset.label}</button>
					{/each}
				</div>
			</div>
			<ActionButton variant="secondary" size="sm" on:click={() => void loadSchema()} loading={loadingSchema} loadingLabel="Refreshing"><RefreshCw slot="icon" class="h-4 w-4" />Refresh</ActionButton>
			{#if error}<div class="alert-danger basis-full">{error}</div>{/if}
		</div>

		{#if loadingSchema}
			<div class="surface h-96 animate-pulse"></div>
		{:else if details.length === 0}
			<SectionPanel title="Relationships"><EmptyState title="No tables found." description="Choose another schema or create database tables first." /></SectionPanel>
		{:else}
			<div class="surface overflow-hidden">
				<div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-3 py-2 dark:border-neutral-800">
					<div class="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400"><span>{graph.nodes.length} tables</span><span>·</span><span>{graph.edges.length} relations</span><span>·</span><span>{pagePresets.find((preset) => preset.value === pagePreset)?.label} landscape</span></div>
					<div class="flex flex-wrap items-center gap-2">
						<label class="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-300"><input type="checkbox" bind:checked={showDataTypes} /> Types</label>
						<label class="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-300"><input type="checkbox" bind:checked={showRelationLabels} /> Relation labels</label>
						<div class="ml-1 flex items-center overflow-hidden rounded-md border border-gray-200 dark:border-neutral-800">
							<button class="p-1.5 hover:bg-gray-50 dark:hover:bg-neutral-900" aria-label="Zoom out" on:click={() => changeZoom(-0.1)}><Minus class="h-3.5 w-3.5" /></button>
							<button class="border-x border-gray-200 px-2 py-1 text-[11px] tabular-nums dark:border-neutral-800" on:click={fitGraph}>{Math.round(zoom * 100)}%</button>
							<button class="p-1.5 hover:bg-gray-50 dark:hover:bg-neutral-900" aria-label="Zoom in" on:click={() => changeZoom(0.1)}><Plus class="h-3.5 w-3.5" /></button>
						</div>
						<button class="flex items-center gap-1 rounded-md border border-gray-200 px-2 py-1 text-[11px] hover:bg-gray-50 dark:border-neutral-800 dark:hover:bg-neutral-900" on:click={fitGraph}><LocateFixed class="h-3.5 w-3.5" /> Fit</button>
						<div class="ml-1 flex items-center gap-1 border-l border-gray-200 pl-2 dark:border-neutral-800">
							<Download class="mr-1 h-3.5 w-3.5 text-gray-400" />
							{#each exportFormats as format}
								<button type="button" class="rounded border border-gray-200 px-2 py-1 text-[10px] font-semibold uppercase hover:bg-gray-50 disabled:opacity-50 dark:border-neutral-800 dark:hover:bg-neutral-900" disabled={Boolean(exporting)} on:click={() => void exportDiagram(format)}>{exporting === format ? '…' : format}</button>
							{/each}
						</div>
					</div>
				</div>

				<div class="max-h-[72vh] min-h-[440px] overflow-auto bg-gray-50/50 p-4 dark:bg-neutral-950/40">
					<div class="relative mx-auto" style={`width:${graph.width * zoom}px;height:${graph.height * zoom}px;min-width:100%`}>
						<div class="absolute left-0 top-0 origin-top-left" style={`width:${graph.width}px;height:${graph.height}px;transform:scale(${zoom})`}>
							<svg class="pointer-events-none absolute inset-0" width={graph.width} height={graph.height} aria-hidden="true">
								<defs><marker id="erd-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" class="fill-gray-600 dark:fill-gray-400" /></marker></defs>
								{#each graph.edges as edge (edge.id)}
									{@const relation = edgeGeometry(edge)}
									<path d={relation.path} fill="none" class="stroke-gray-500 dark:stroke-gray-500" stroke-width="2" marker-end="url(#erd-arrow)" />
									{#if showRelationLabels}<text x={relation.labelX} y={relation.labelY - 6} text-anchor="middle" class="fill-gray-700 text-[10px] [paint-order:stroke] stroke-white stroke-[4px] dark:fill-gray-300 dark:stroke-neutral-950">{edge.foreignKey.column} → {edge.foreignKey.referencedColumn}</text>{/if}
								{/each}
							</svg>

							{#each graph.nodes as node (node.id)}
								<button type="button" class={`absolute overflow-hidden rounded-lg border text-left shadow-sm transition ${nodeClasses(node)}`} style={`left:${node.x}px;top:${node.y}px;width:${node.width}px;height:${node.height}px`} on:click={() => (selectedTable = node.detail.name)}>
									<div class="border-b border-gray-200 px-3 py-2 dark:border-neutral-800"><p class="truncate font-mono text-[10px] text-gray-400 dark:text-gray-500">{node.detail.schema}</p><p class="truncate text-sm font-semibold text-gray-950 dark:text-white">{node.detail.name}</p></div>
									<div class="space-y-1 px-3 py-2">
										{#each node.detail.columns.slice(0, visibleColumnCount(node)) as column}<div class="flex items-center justify-between gap-2 text-[11px]"><span class="truncate font-mono text-gray-700 dark:text-gray-300">{column.primaryKey ? 'PK ' : ''}{column.name}</span>{#if showDataTypes}<span class="shrink-0 text-gray-400 dark:text-gray-500">{column.dataType}</span>{/if}</div>{/each}
										{#if node.detail.columns.length > visibleColumnCount(node)}<p class="text-[10px] text-gray-400 dark:text-gray-500">+{node.detail.columns.length - visibleColumnCount(node)} more</p>{/if}
									</div>
								</button>
							{/each}
						</div>
					</div>
				</div>
			</div>

			{#if selectedDetails}
				<div class="grid gap-4 xl:grid-cols-2">
					<SectionPanel title={`${selectedDetails.schema}.${selectedDetails.name}`}>
						<div class="overflow-x-auto"><table class="w-full text-left text-xs"><thead class="text-gray-500 dark:text-gray-400"><tr><th class="pb-2 pr-3">Column</th><th class="pb-2 pr-3">Type</th><th class="pb-2">Flags</th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-neutral-800">{#each selectedDetails.columns as column}<tr><td class="py-2 pr-3 font-mono text-gray-950 dark:text-white">{column.name}</td><td class="py-2 pr-3 font-mono text-gray-600 dark:text-gray-300">{column.dataType}</td><td class="py-2 text-gray-500 dark:text-gray-400">{[column.primaryKey ? 'PK' : '', column.nullable ? 'nullable' : 'required', column.autoGenerated ? 'generated' : ''].filter(Boolean).join(' · ')}</td></tr>{/each}</tbody></table></div>
						<div class="mt-5 border-t border-gray-100 pt-4 dark:border-neutral-800"><p class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Foreign keys</p>{#if selectedDetails.foreignKeys.length === 0}<p class="text-sm text-gray-500 dark:text-gray-400">No foreign keys.</p>{:else}<div class="space-y-2">{#each selectedDetails.foreignKeys as foreignKey}<div class="rounded-md border border-gray-200 p-3 text-xs dark:border-neutral-800"><p class="font-mono text-gray-950 dark:text-white">{foreignKey.column} → {foreignKey.referencedSchema}.{foreignKey.referencedTable}.{foreignKey.referencedColumn}</p><p class="mt-1 text-gray-500 dark:text-gray-400">{foreignKey.name} · update {foreignKey.onUpdate} · delete {foreignKey.onDelete}</p></div>{/each}</div>{/if}</div>
					</SectionPanel>

					<div class="space-y-4">
						<SectionPanel title="Indexes">{#if selectedDetails.indexes.length === 0}<p class="text-sm text-gray-500 dark:text-gray-400">No indexes reported.</p>{:else}<div class="space-y-2">{#each selectedDetails.indexes as index}<div class="rounded-md border border-gray-200 p-3 dark:border-neutral-800"><div class="flex flex-wrap items-center justify-between gap-2"><p class="font-mono text-xs font-medium text-gray-950 dark:text-white">{index.name}</p><p class="text-[11px] text-gray-500 dark:text-gray-400">{[index.primary ? 'PRIMARY' : '', index.unique ? 'UNIQUE' : '', index.method].filter(Boolean).join(' · ')}</p></div><p class="mt-1 font-mono text-xs text-gray-600 dark:text-gray-300">{index.columns.join(', ') || 'expression index'}</p></div>{/each}</div>{/if}</SectionPanel>
						<SectionPanel title="Constraints">{#if selectedDetails.constraints.length === 0}<p class="text-sm text-gray-500 dark:text-gray-400">No constraints reported.</p>{:else}<div class="space-y-2">{#each selectedDetails.constraints as constraint}<div class="rounded-md border border-gray-200 p-3 dark:border-neutral-800"><div class="flex flex-wrap items-center justify-between gap-2"><p class="font-mono text-xs font-medium text-gray-950 dark:text-white">{constraint.name}</p><p class="text-[11px] text-gray-500 dark:text-gray-400">{constraint.type}</p></div>{#if constraint.columns.length > 0}<p class="mt-1 font-mono text-xs text-gray-600 dark:text-gray-300">{constraint.columns.join(', ')}</p>{/if}{#if constraint.definition}<p class="mt-1 break-words font-mono text-[11px] text-gray-500 dark:text-gray-400">{constraint.definition}</p>{/if}</div>{/each}</div>{/if}</SectionPanel>
					</div>
				</div>
			{/if}
		{/if}
	</div>
{/if}
