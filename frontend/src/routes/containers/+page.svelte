<script lang="ts">
	import { Search } from '@lucide/svelte';
	import { onDestroy, onMount } from 'svelte';
	import Pagination from '$components/Pagination.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { loadRuntimeContainers, mergeRuntimeContainerTelemetry, type RuntimeContainer } from '$lib/api/container-inventory';
	import { beginMainContentLoading } from '$stores/main-loading';

	let rows: RuntimeContainer[] = [];
	let loading = false;
	let telemetryError = '';
	let error = '';
	let query = '';
	let stateFilter = 'all';
	let runtimeFilter = 'all';
	let pageIndex = 0;
	let pageSize = 10;
	let metadataController: AbortController | null = null;
	let telemetryController: AbortController | null = null;

	$: stateOptions = [...new Set(rows.map((row) => row.state || 'unknown'))].sort();
	$: runtimeOptions = [...new Set(rows.map((row) => row.composeProject || 'standalone'))].sort();
	$: searchTerm = query.trim().toLowerCase();
	$: filteredRows = rows.filter((row) => {
		const searchable = [row.name, row.id, row.image, row.composeProject, row.service, row.state, row.status].join(' ').toLowerCase();
		const matchesSearch = !searchTerm || searchable.includes(searchTerm);
		const matchesState = stateFilter === 'all' || (row.state || 'unknown') === stateFilter;
		const runtime = row.composeProject || 'standalone';
		const matchesRuntime = runtimeFilter === 'all' || runtime === runtimeFilter;
		return matchesSearch && matchesState && matchesRuntime;
	});
	$: maxPage = Math.max(0, Math.ceil(filteredRows.length / pageSize) - 1);
	$: if (pageIndex > maxPage) pageIndex = maxPage;
	$: visibleRows = filteredRows.slice(pageIndex * pageSize, (pageIndex + 1) * pageSize);
	$: hasNext = (pageIndex + 1) * pageSize < filteredRows.length;
	onMount(() => {
		void load();
	});

	onDestroy(() => {
		metadataController?.abort();
		telemetryController?.abort();
	});

	async function load() {
		if (loading) return;
		loading = true;
		metadataController?.abort();
		const controller = new AbortController();
		metadataController = controller;
		const finishMainLoading = rows.length === 0 ? beginMainContentLoading() : null;
		error = '';
		try {
			const next = await loadRuntimeContainers({ telemetry: false, signal: controller.signal });
			if (controller.signal.aborted) return;
			rows = next.sort((a, b) => {
				const runningOrder = Number(b.state === 'running') - Number(a.state === 'running');
				if (runningOrder !== 0) return runningOrder;
				return (a.composeProject || '').localeCompare(b.composeProject || '') || a.name.localeCompare(b.name);
			});
			void loadTelemetry();
		} catch (err) {
			if (controller.signal.aborted) return;
			error = err instanceof Error ? err.message : 'Failed to load host container inventory';
		} finally {
			finishMainLoading?.();
			loading = false;
		}
	}

	async function loadTelemetry() {
		telemetryController?.abort();
		const controller = new AbortController();
		telemetryController = controller;
		telemetryError = '';
		try {
			const telemetryRows = await loadRuntimeContainers({ signal: controller.signal });
			if (controller.signal.aborted) return;
			rows = mergeRuntimeContainerTelemetry(rows, telemetryRows);
		} catch (err) {
			if (controller.signal.aborted) return;
			telemetryError = err instanceof Error ? err.message : 'Live container telemetry is unavailable';
			rows = rows.map((row) => ({ ...row, cpu: 0, memoryMb: 0, memoryLimitMb: 0, metricsAvailable: false }));
		}
	}

	function resetPage() {
		pageIndex = 0;
	}

	function formatMemory(value: number) {
		if (!Number.isFinite(value)) return '—';
		return value >= 1024 ? `${(value / 1024).toFixed(2)} GB` : `${value.toFixed(0)} MB`;
	}

	function stateDot(state: string) {
		switch (state) {
			case 'running': return 'bg-emerald-500';
			case 'paused': return 'bg-amber-500';
			case 'dead': return 'bg-red-500';
			case 'restarting': return 'bg-blue-500';
			default: return 'bg-gray-400 dark:bg-gray-600';
		}
	}

	function runtimeGroup(row: RuntimeContainer) {
		if (row.composeProject && row.service) return `${row.composeProject} / ${row.service}`;
		if (row.composeProject) return row.composeProject;
		if (row.service) return row.service;
		return 'standalone';
	}
</script>

<svelte:head>
	<title>Containers · MyPaas</title>
</svelte:head>

<div class="page-shell">
	<TableShell
		title="Containers"
		description="Read-only host runtime inventory. Manage application lifecycle from each project."
		loading={false}
		{error}
		empty={filteredRows.length === 0}
		emptyTitle={rows.length === 0 ? 'No containers found.' : 'No containers match the current filters.'}
		emptyDescription={rows.length === 0 ? 'The Docker-compatible runtime currently reports no containers.' : 'Clear search or filters to see the host inventory.'}
		on:retry={() => load()}
	>
		<svelte:fragment slot="actions">
			<div class="grid w-full gap-2 md:grid-cols-2 xl:grid-cols-[minmax(16rem,1fr)_11rem_14rem_5.5rem]">
				<div class="relative min-w-0">
					<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<input
						id="container-search"
						class="field w-full !pl-9 font-mono"
						placeholder="Search name, image, project, status…"
						aria-label="Search containers"
						bind:value={query}
						on:input={resetPage}
					/>
				</div>

				<select id="container-state" class="field w-full" aria-label="Filter containers by state" bind:value={stateFilter} on:change={resetPage}>
					<option value="all">All states</option>
					{#each stateOptions as state}<option value={state}>{state}</option>{/each}
				</select>

				<select id="container-runtime" class="field w-full font-mono" aria-label="Filter containers by runtime group" bind:value={runtimeFilter} on:change={resetPage}>
					<option value="all">All runtime groups</option>
					{#each runtimeOptions as runtime}<option value={runtime}>{runtime}</option>{/each}
				</select>

				<select id="container-page-size" class="field w-full" aria-label="Rows per page" bind:value={pageSize} on:change={resetPage}>
					<option value={10}>10 rows</option>
					<option value={20}>20 rows</option>
					<option value={50}>50 rows</option>
					<option value={100}>100 rows</option>
				</select>
			</div>
		</svelte:fragment>

		<svelte:fragment slot="notice">
			{#if telemetryError}
				<div class="border-b border-amber-200/70 bg-amber-50/70 px-4 py-2 text-xs text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-200" role="status">
					Live CPU/RAM telemetry unavailable; container metadata is still current.
				</div>
			{/if}
		</svelte:fragment>

		<table class="data-table table-fixed min-w-[64rem]">
			<colgroup>
				<col class="w-[17%]" />
				<col class="w-[19%]" />
				<col class="w-[24%]" />
				<col class="w-[11%]" />
				<col class="w-[8%]" />
				<col class="w-[12%]" />
				<col class="w-[9%]" />
			</colgroup>
			<thead>
				<tr>
					<th>Container</th>
					<th>Compose / service</th>
					<th>Image</th>
					<th>State</th>
					<th class="text-right">CPU</th>
					<th class="text-right">Memory</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				{#each visibleRows as row (row.id)}
					<tr>
						<td>
							<div class="min-w-0">
								<p class="truncate font-mono text-[13px] font-medium text-gray-950 dark:text-white" title={row.name}>{row.name}</p>
								<p class="mt-0.5 truncate font-mono text-xs text-gray-400" title={row.id}>{row.id.slice(0, 12)}</p>
							</div>
						</td>
						<td><p class="truncate font-mono text-[13px] text-gray-700 dark:text-gray-300" title={runtimeGroup(row)}>{runtimeGroup(row)}</p></td>
						<td><p class="truncate font-mono text-[13px] text-gray-600 dark:text-gray-300" title={row.image}>{row.image || '—'}</p></td>
						<td class="whitespace-nowrap">
							<span class="inline-flex items-center gap-2 text-sm capitalize text-gray-700 dark:text-gray-300">
								<span class={`status-dot ${stateDot(row.state)}`}></span>{row.state || 'unknown'}
							</span>
						</td>
						<td class="whitespace-nowrap text-right font-mono text-[13px] tabular-nums">{row.metricsAvailable ? `${row.cpu.toFixed(2)}%` : '—'}</td>
						<td class="whitespace-nowrap text-right font-mono text-[13px] tabular-nums">
							{#if row.metricsAvailable}
								{formatMemory(row.memoryMb)}{#if row.memoryLimitMb > 0} / {formatMemory(row.memoryLimitMb)}{/if}
							{:else}—{/if}
						</td>
						<td><p class="truncate text-[13px] text-gray-500 dark:text-gray-400" title={row.status}>{row.status || '—'}</p></td>
					</tr>
				{/each}
			</tbody>
		</table>

		<svelte:fragment slot="footer">
			{#if filteredRows.length > 0}
				<Pagination bind:page={pageIndex} {pageSize} totalShown={visibleRows.length} {hasNext} loading={loading} label="Containers" />
			{/if}
		</svelte:fragment>
	</TableShell>
</div>
