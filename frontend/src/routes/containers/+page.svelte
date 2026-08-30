<script lang="ts">
	import { RefreshCw } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { loadRuntimeContainers, type RuntimeContainer } from '$lib/api/container-inventory';

	let rows: RuntimeContainer[] = [];
	let loading = true;
	let refreshing = false;
	let error = '';

	$: runningCount = rows.filter((row) => row.state === 'running').length;
	$: totalCpu = rows.reduce((sum, row) => sum + (row.metricsAvailable ? row.cpu : 0), 0);
	$: totalMemoryMb = rows.reduce((sum, row) => sum + (row.metricsAvailable ? row.memoryMb : 0), 0);

	onMount(() => {
		void load();
		const timer = window.setInterval(() => void load(true), 5000);
		return () => window.clearInterval(timer);
	});

	async function load(background = false) {
		if (refreshing) return;
		refreshing = true;
		if (!background && rows.length === 0) loading = true;
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
			loading = false;
			refreshing = false;
		}
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

<div class="page-shell py-6">
	<div class="mb-5 flex flex-wrap items-center justify-between gap-3 px-5">
		<div>
			<p class="text-sm text-gray-500 dark:text-gray-400">Host-wide Docker-compatible runtime view, including MyPaaS system containers and application containers.</p>
			{#if rows.length > 0}
				<p class="mt-1 text-xs text-gray-400 dark:text-gray-500">{rows.length} total · {runningCount} running · {totalCpu.toFixed(1)}% CPU · {formatMemory(totalMemoryMb)} RAM</p>
			{/if}
		</div>
		<ActionButton variant="secondary" size="sm" loading={refreshing} loadingLabel="Refreshing" on:click={() => load()}>
			<RefreshCw slot="icon" class="h-4 w-4" />
			Refresh
		</ActionButton>
	</div>

	<TableShell
		title="Host containers"
		description="Every container visible through the host runtime is listed. Live CPU and memory are sampled for running containers every five seconds."
		{loading}
		loadingRows={7}
		{error}
		empty={rows.length === 0}
		emptyTitle="No containers found."
		emptyDescription="The Docker-compatible runtime currently reports no containers."
		on:retry={() => load()}
	>
		<table class="data-table">
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
				{#each rows as row (row.id)}
					<tr>
						<td>
							<div class="min-w-0">
								<p class="font-mono text-xs font-medium text-gray-950 dark:text-white">{row.name}</p>
								<p class="mt-0.5 max-w-40 truncate font-mono text-[10px] text-gray-400" title={row.id}>{row.id.slice(0, 12)}</p>
							</div>
						</td>
						<td class="font-mono text-xs text-gray-700 dark:text-gray-300">{runtimeGroup(row)}</td>
						<td><p class="max-w-64 truncate font-mono text-xs text-gray-600 dark:text-gray-300" title={row.image}>{row.image || '—'}</p></td>
						<td>
							<span class="inline-flex items-center gap-2 text-sm capitalize text-gray-700 dark:text-gray-300">
								<span class={`status-dot ${stateDot(row.state)}`}></span>{row.state || 'unknown'}
							</span>
						</td>
						<td class="text-right font-mono text-xs">{row.metricsAvailable ? `${row.cpu.toFixed(2)}%` : '—'}</td>
						<td class="text-right font-mono text-xs">
							{#if row.metricsAvailable}
								{formatMemory(row.memoryMb)}{#if row.memoryLimitMb > 0} / {formatMemory(row.memoryLimitMb)}{/if}
							{:else}—{/if}
						</td>
						<td class="max-w-72 text-xs text-gray-500 dark:text-gray-400">{row.status || '—'}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</TableShell>
</div>
