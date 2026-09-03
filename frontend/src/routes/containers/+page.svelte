<script lang="ts">
	import { ChevronDown, ChevronUp, Search } from '@lucide/svelte';
	import { onDestroy, onMount } from 'svelte';
	import IconButton from '$components/IconButton.svelte';
	import Pagination from '$components/Pagination.svelte';
	import SelectMenu from '$components/SelectMenu.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { loadRuntimeContainers, mergeRuntimeContainerTelemetry, type RuntimeContainer } from '$lib/api/container-inventory';
	import { beginMainContentLoading } from '$stores/main-loading';

	const telemetryRefreshMs = 15_000;

	let rows: RuntimeContainer[] = [];
	let loading = false;
	let telemetryError = '';
	let error = '';
	let query = '';
	let stateFilter = 'all';
	let runtimeFilter = 'all';
	let pageIndex = 0;
	let pageSize = 10;
	let expanded = new Set<string>();
	let metadataController: AbortController | null = null;
	let telemetryController: AbortController | null = null;
	let telemetryPoll: ReturnType<typeof setInterval> | undefined;

	$: stateValues = [...new Set(rows.map((row) => row.state || 'unknown'))].sort();
	$: runtimeValues = [...new Set(rows.map((row) => row.composeProject || 'standalone'))].sort();
	$: stateOptions = [{ value: 'all', label: 'All states' }, ...stateValues.map((state) => ({ value: state, label: capitalize(state) }))];
	$: runtimeOptions = [{ value: 'all', label: 'All runtime groups' }, ...runtimeValues.map((runtime) => ({ value: runtime, label: runtime }))];
	$: pageSizeOptions = [10, 20, 50, 100].map((size) => ({ value: String(size), label: `${size} rows` }));
	$: searchTerm = query.trim().toLowerCase();
	$: filteredRows = rows.filter((row) => {
		const networkSearch = row.networks.map((network) => `${network.name} ${network.ipAddress}`).join(' ');
		const searchable = [row.name, row.id, row.image, row.composeProject, row.service, row.state, row.status, row.health, row.ports, networkSearch].join(' ').toLowerCase();
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
		telemetryPoll = setInterval(() => void loadTelemetry(), telemetryRefreshMs);
	});

	onDestroy(() => {
		metadataController?.abort();
		telemetryController?.abort();
		if (telemetryPoll) clearInterval(telemetryPoll);
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
		if (rows.length === 0) return;
		telemetryController?.abort();
		const controller = new AbortController();
		telemetryController = controller;
		try {
			const telemetryRows = await loadRuntimeContainers({ signal: controller.signal });
			if (controller.signal.aborted) return;
			rows = mergeRuntimeContainerTelemetry(rows, telemetryRows);
			const runningRows = rows.filter((row) => row.state === 'running');
			const missingSamples = runningRows.filter((row) => !row.metricsAvailable).length;
			telemetryError = missingSamples === 0
				? ''
				: missingSamples === runningRows.length
					? 'CPU/RAM telemetry has not produced a sample yet. Container metadata is still current.'
					: `CPU/RAM telemetry is unavailable for ${missingSamples} running container${missingSamples === 1 ? '' : 's'}.`;
		} catch (err) {
			if (controller.signal.aborted) return;
			telemetryError = err instanceof Error ? err.message : 'Live CPU/RAM telemetry is unavailable';
		}
	}

	function resetPage() {
		pageIndex = 0;
	}

	function chooseState(value: string) {
		stateFilter = value;
		resetPage();
	}

	function chooseRuntime(value: string) {
		runtimeFilter = value;
		resetPage();
	}

	function choosePageSize(value: string) {
		const parsed = Number(value);
		if (Number.isFinite(parsed) && parsed > 0) pageSize = parsed;
		resetPage();
	}

	function toggleDetails(id: string) {
		expanded.has(id) ? expanded.delete(id) : expanded.add(id);
		expanded = new Set(expanded);
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

	function healthClass(health: string) {
		switch (health) {
			case 'healthy': return 'text-emerald-700 dark:text-emerald-300';
			case 'unhealthy': return 'text-red-700 dark:text-red-300';
			case 'starting': return 'text-amber-700 dark:text-amber-200';
			default: return 'text-gray-500 dark:text-gray-400';
		}
	}

	function runtimeGroup(row: RuntimeContainer) {
		if (row.composeProject && row.service) return `${row.composeProject} / ${row.service}`;
		if (row.composeProject) return row.composeProject;
		if (row.service) return row.service;
		return 'standalone';
	}

	function compactImage(value: string) {
		let image = value.trim().replace(/^docker\.io\/library\//, '').replace(/^docker\.io\//, '');
		const separator = image.lastIndexOf(':');
		if (separator <= image.lastIndexOf('/')) return image;
		const tag = image.slice(separator + 1);
		if (/^[a-f0-9]{20,}$/i.test(tag)) image = `${image.slice(0, separator + 1)}${tag.slice(0, 12)}`;
		return image;
	}

	function capitalize(value: string) {
		return value ? `${value.charAt(0).toUpperCase()}${value.slice(1)}` : value;
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
			<div class="grid w-full gap-2 md:grid-cols-2 xl:grid-cols-[minmax(18rem,1fr)_10rem_13rem_7rem]">
				<div class="relative min-w-0">
					<Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<input
						id="container-search"
						class="field w-full !pl-9 font-mono"
						placeholder="Search container, image, project, port…"
						aria-label="Search containers"
						bind:value={query}
						on:input={resetPage}
					/>
				</div>

				<SelectMenu value={stateFilter} options={stateOptions} ariaLabel="Filter containers by state" on:change={(event) => chooseState(event.detail)} />
				<SelectMenu value={runtimeFilter} options={runtimeOptions} ariaLabel="Filter containers by runtime group" on:change={(event) => chooseRuntime(event.detail)} />
				<SelectMenu value={String(pageSize)} options={pageSizeOptions} ariaLabel="Rows per page" on:change={(event) => choosePageSize(event.detail)} />
			</div>
		</svelte:fragment>

		<svelte:fragment slot="notice">
			{#if telemetryError}
				<div class="border-b border-amber-200/70 px-4 py-2 text-xs text-amber-800 dark:border-amber-900/50 dark:text-amber-200" role="status">
					{telemetryError}
				</div>
			{/if}
		</svelte:fragment>

		<table class="data-table table-fixed min-w-[90rem]">
			<colgroup>
				<col class="w-[15%]" />
				<col class="w-[15%]" />
				<col class="w-[19%]" />
				<col class="w-[10%]" />
				<col class="w-[6%]" />
				<col class="w-[10%]" />
				<col class="w-[7%]" />
				<col class="w-[10%]" />
				<col class="w-[6%]" />
				<col class="w-[2%]" />
			</colgroup>
			<thead>
				<tr>
					<th>Container</th>
					<th>Project / service</th>
					<th>Image</th>
					<th>State</th>
					<th class="text-right">CPU</th>
					<th class="text-right">Memory</th>
					<th class="text-right">Restarts</th>
					<th>Ports</th>
					<th>Uptime</th>
					<th class="text-right"><span class="sr-only">Details</span></th>
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
						<td><p class="truncate font-mono text-[13px] text-gray-600 dark:text-gray-300" title={row.image}>{compactImage(row.image) || '—'}</p></td>
						<td>
							<div class="min-w-0">
								<span class="inline-flex items-center gap-2 text-sm capitalize text-gray-700 dark:text-gray-300">
									<span class={`status-dot ${stateDot(row.state)}`}></span>{row.state || 'unknown'}
								</span>
								{#if row.health}<p class={`mt-0.5 text-xs capitalize ${healthClass(row.health)}`}>{row.health}</p>{/if}
							</div>
						</td>
						<td class="whitespace-nowrap text-right font-mono text-[13px] tabular-nums">{row.metricsAvailable ? `${row.cpu.toFixed(2)}%` : '—'}</td>
						<td class="whitespace-nowrap text-right font-mono text-[13px] tabular-nums">
							{#if row.metricsAvailable}
								{formatMemory(row.memoryMb)}{#if row.memoryLimitMb > 0} / {formatMemory(row.memoryLimitMb)}{/if}
							{:else}—{/if}
						</td>
						<td class="whitespace-nowrap text-right font-mono text-[13px] tabular-nums">{row.detailsAvailable ? row.restartCount : '—'}</td>
						<td><p class="truncate font-mono text-xs text-gray-600 dark:text-gray-300" title={row.ports}>{row.ports || '—'}</p></td>
						<td class="whitespace-nowrap text-[13px] text-gray-500 dark:text-gray-400" title={row.status}>{row.state === 'running' ? row.uptime || '—' : '—'}</td>
						<td class="text-right">
							<IconButton label={`${expanded.has(row.id) ? 'Hide' : 'Show'} ${row.name} details`} variant="ghost" on:click={() => toggleDetails(row.id)}>
								{#if expanded.has(row.id)}<ChevronUp class="h-4 w-4" aria-hidden="true" />{:else}<ChevronDown class="h-4 w-4" aria-hidden="true" />{/if}
							</IconButton>
						</td>
					</tr>
					{#if expanded.has(row.id)}
						<tr>
							<td colspan="10" class="!p-0">
								<div class="grid gap-5 border-t border-gray-100/70 px-4 py-4 dark:border-neutral-900 md:grid-cols-3 lg:px-5">
									<div class="min-w-0">
										<p class="text-xs font-medium text-gray-500 dark:text-gray-400">Identity</p>
										<p class="mt-2 break-all font-mono text-xs text-gray-700 dark:text-gray-300">{row.id}</p>
										<p class="mt-2 break-all font-mono text-xs text-gray-500 dark:text-gray-400">{row.image || '—'}</p>
									</div>
									<div class="min-w-0">
										<p class="text-xs font-medium text-gray-500 dark:text-gray-400">Runtime</p>
										<p class="mt-2 text-sm text-gray-700 dark:text-gray-300">{row.status || '—'}</p>
										<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Restarts: {row.detailsAvailable ? row.restartCount : 'unavailable'}</p>
									</div>
									<div class="min-w-0">
										<p class="text-xs font-medium text-gray-500 dark:text-gray-400">Network</p>
										<p class="mt-2 break-all font-mono text-xs text-gray-700 dark:text-gray-300">{row.ports || 'No published ports'}</p>
										{#if row.networks.length > 0}
											<div class="mt-2 space-y-1">
												{#each row.networks as network}
													<p class="font-mono text-xs text-gray-500 dark:text-gray-400">{network.name}{network.ipAddress ? ` · ${network.ipAddress}` : ''}</p>
												{/each}
											</div>
										{:else}<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">Network details unavailable.</p>{/if}
									</div>
								</div>
							</td>
						</tr>
					{/if}
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
