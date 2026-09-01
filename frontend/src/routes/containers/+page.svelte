<script lang="ts">
	import { Search, Trash2 } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import IconButton from '$components/IconButton.svelte';
	import Pagination from '$components/Pagination.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { loadRuntimeContainers, removeRuntimeContainer, type RuntimeContainer } from '$lib/api/container-inventory';
	import { beginMainContentLoading } from '$stores/main-loading';
	import { toast } from '$stores/toast';

	let rows: RuntimeContainer[] = [];
	let loading = false;
	let deletingID = '';
	let error = '';
	let query = '';
	let stateFilter = 'all';
	let runtimeFilter = 'all';
	let pageIndex = 0;
	let pageSize = 10;

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

	async function load() {
		if (loading) return;
		loading = true;
		const finishMainLoading = rows.length === 0 ? beginMainContentLoading() : null;
		error = '';
		try {
			const next = await loadRuntimeContainers();
			rows = next.sort((a, b) => {
				const runningOrder = Number(b.state === 'running') - Number(a.state === 'running');
				if (runningOrder !== 0) return runningOrder;
				return (a.composeProject || '').localeCompare(b.composeProject || '') || a.name.localeCompare(b.name);
			});
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load host container inventory';
		} finally {
			finishMainLoading?.();
			loading = false;
		}
	}

	async function removeContainer(row: RuntimeContainer) {
		if (!canRemove(row) || deletingID) return;
		if (!window.confirm(`Remove stopped container ${row.name}?`)) return;
		deletingID = row.id;
		try {
			await removeRuntimeContainer(row.id);
			rows = rows.filter((item) => item.id !== row.id);
			toast.success('Container removed');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove container');
		} finally {
			deletingID = '';
		}
	}

	function canRemove(row: RuntimeContainer) {
		return !['running', 'paused', 'restarting'].includes(row.state);
	}

	function resetPage() {
		pageIndex = 0;
	}

	function changePageSize(event: Event) {
		pageSize = Number((event.currentTarget as HTMLSelectElement).value);
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
		loading={false}
		{error}
		empty={filteredRows.length === 0}
		emptyTitle={rows.length === 0 ? 'No containers found.' : 'No containers match the current filters.'}
		emptyDescription={rows.length === 0 ? 'The Docker-compatible runtime currently reports no containers.' : 'Clear search or filters to see the host inventory.'}
		on:retry={() => load()}
	>
		<svelte:fragment slot="notice">
			<div class="grid gap-3 border-b border-gray-100/70 px-4 py-3 dark:border-neutral-900 md:grid-cols-[minmax(16rem,1fr)_12rem_14rem_auto] md:items-end lg:px-5">
				<label class="block min-w-0" for="container-search">
					<span class="field-label">Search</span>
					<div class="relative">
						<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" aria-hidden="true" />
						<input id="container-search" class="field w-full !pl-9 font-mono" placeholder="name, image, project, status…" bind:value={query} on:input={resetPage} />
					</div>
				</label>

				<label class="block" for="container-state">
					<span class="field-label">State</span>
					<select id="container-state" class="field w-full" bind:value={stateFilter} on:change={resetPage}>
						<option value="all">All states</option>
						{#each stateOptions as state}<option value={state}>{state}</option>{/each}
					</select>
				</label>

				<label class="block" for="container-runtime">
					<span class="field-label">Compose / runtime</span>
					<select id="container-runtime" class="field w-full font-mono" bind:value={runtimeFilter} on:change={resetPage}>
						<option value="all">All runtime groups</option>
						{#each runtimeOptions as runtime}<option value={runtime}>{runtime}</option>{/each}
					</select>
				</label>

				<label class="block" for="container-page-size">
					<span class="field-label">Rows</span>
					<select id="container-page-size" class="field" value={pageSize} on:change={changePageSize}>
						<option value="10">10</option>
						<option value="20">20</option>
						<option value="50">50</option>
						<option value="100">100</option>
					</select>
				</label>
			</div>
		</svelte:fragment>

		<table class="data-table table-fixed min-w-[68rem]">
			<colgroup>
				<col class="w-[16%]" />
				<col class="w-[18%]" />
				<col class="w-[23%]" />
				<col class="w-[10%]" />
				<col class="w-[8%]" />
				<col class="w-[12%]" />
				<col class="w-[9%]" />
				<col class="w-[4%]" />
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
					<th><span class="sr-only">Actions</span></th>
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
						<td class="text-right">
							{#if canRemove(row)}
								<IconButton label={`Remove ${row.name}`} variant="ghostDanger" loading={deletingID === row.id} disabled={Boolean(deletingID) && deletingID !== row.id} on:click={() => void removeContainer(row)}>
									<Trash2 class="h-4 w-4" aria-hidden="true" />
								</IconButton>
							{/if}
						</td>
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
