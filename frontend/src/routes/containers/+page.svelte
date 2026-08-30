<script lang="ts">
	import { RefreshCw } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api } from '$api';
	import type { ContainerMetrics, Project } from '$types';

	type ContainerRow = ContainerMetrics & {
		projectId: string;
		projectName: string;
		deployMode: Project['deployMode'];
	};

	let rows: ContainerRow[] = [];
	let loading = true;
	let refreshing = false;
	let error = '';

	$: totalCpu = rows.reduce((sum, row) => sum + row.cpu, 0);
	$: totalMemoryMb = rows.reduce((sum, row) => sum + row.memoryMb, 0);

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
			const projects = (await api.projects.list()).filter(
				(project) => project.status === 'running' && project.deployMode !== 'static'
			);
			const snapshots = await Promise.allSettled(projects.map((project) => api.metrics.snapshot(project.id)));
			const nextRows: ContainerRow[] = [];
			for (let index = 0; index < projects.length; index += 1) {
				const result = snapshots[index];
				if (!result || result.status !== 'fulfilled') continue;
				for (const metric of result.value.items ?? []) {
					nextRows.push({
						...metric,
						projectId: projects[index].id,
						projectName: projects[index].name,
						deployMode: projects[index].deployMode
					});
				}
			}
			rows = nextRows.sort((a, b) => a.projectName.localeCompare(b.projectName) || a.service.localeCompare(b.service));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load container metrics';
		} finally {
			loading = false;
			refreshing = false;
		}
	}

	function formatMemory(value: number) {
		if (!Number.isFinite(value)) return '—';
		return value >= 1024 ? `${(value / 1024).toFixed(2)} GB` : `${value.toFixed(0)} MB`;
	}
</script>

<svelte:head>
	<title>Containers · MyPaas</title>
</svelte:head>

<div class="page-shell py-6">
	<div class="mb-5 flex flex-wrap items-center justify-between gap-3 px-5">
		<div>
			<p class="text-sm text-gray-500 dark:text-gray-400">Live resource view for containers owned by running MyPaaS projects.</p>
			{#if rows.length > 0}
				<p class="mt-1 text-xs text-gray-400 dark:text-gray-500">{rows.length} containers · {totalCpu.toFixed(1)}% CPU · {formatMemory(totalMemoryMb)} RAM</p>
			{/if}
		</div>
		<ActionButton variant="secondary" size="sm" loading={refreshing} loadingLabel="Refreshing" on:click={() => load()}>
			<RefreshCw slot="icon" class="h-4 w-4" />
			Refresh
		</ActionButton>
	</div>

	<TableShell
		title="Running containers"
		description="CPU, memory, and uptime refresh automatically every five seconds. Lifecycle controls stay on each project page."
		{loading}
		loadingRows={5}
		{error}
		empty={rows.length === 0}
		emptyTitle="No running containers."
		emptyDescription="Deploy or start a container-backed project to see it here."
		on:retry={() => load()}
	>
		<table class="data-table">
			<thead>
				<tr>
					<th>Project</th>
					<th>Container / service</th>
					<th>Runtime</th>
					<th>Status</th>
					<th class="text-right">CPU</th>
					<th class="text-right">Memory</th>
					<th class="text-right">Uptime</th>
				</tr>
			</thead>
			<tbody>
				{#each rows as row}
					<tr>
						<td><a class="font-medium text-gray-950 hover:underline dark:text-white" href={`/projects/${row.projectId}`}>{row.projectName}</a></td>
						<td class="font-mono text-xs text-gray-700 dark:text-gray-300">{row.service}</td>
						<td class="text-sm capitalize">{row.deployMode}</td>
						<td><span class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><span class="status-dot bg-emerald-500"></span>Running</span></td>
						<td class="text-right font-mono text-xs">{row.cpu.toFixed(2)}%</td>
						<td class="text-right font-mono text-xs">{formatMemory(row.memoryMb)}{#if row.memoryLimitMb > 0} / {formatMemory(row.memoryLimitMb)}{/if}</td>
						<td class="text-right font-mono text-xs">{row.uptime || '—'}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</TableShell>
</div>
