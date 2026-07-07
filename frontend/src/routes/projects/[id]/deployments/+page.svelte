<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import Pagination from '$components/Pagination.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Deployment } from '$types';

	const pageSize = 20;
	let deployments: Deployment[] = [];
	let loading = true;
	let error = '';
	let expanded = new Set<string>();
	let rollingBackId = '';
	let confirmRollbackId = '';
	let currentPage = 0;
	let hasNext = false;
	let mounted = false;
	let loadedPage = -1;

	$: visibleDeployments = deployments.slice(0, pageSize);
	$: activeCount = visibleDeployments.filter((item) => ['queued', 'cloning', 'building', 'starting'].includes(item.status)).length;
	$: healthyCount = visibleDeployments.filter((item) => ['running', 'stopped', 'rolled_back'].includes(item.status)).length;
	$: failedCount = visibleDeployments.filter((item) => item.status === 'failed').length;
	$: if (mounted && currentPage !== loadedPage) {
		void load();
	}

	function requestRollback(id: string) {
		confirmRollbackId = id;
	}

	async function handleRollback(id: string) {
		rollingBackId = id;
		try {
			await api.deployments.rollback(id);
			toast.success('Rollback completed');
			confirmRollbackId = '';
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to rollback deployment');
		} finally {
			rollingBackId = '';
		}
	}

	function formatDuration(start: string, end: string | null): string {
		if (!end) return '-';
		const ms = new Date(end).getTime() - new Date(start).getTime();
		const s  = Math.floor(ms / 1000);
		return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
	}

	function formatDate(value: string) {
		return new Date(value).toLocaleString();
	}

	onMount(() => {
		mounted = true;
		void load();
		const id = setInterval(load, 3000);
		return () => clearInterval(id);
	});

	async function load() {
		try {
			const rows = await api.deployments.list($page.params.id, currentPage, pageSize, true);
			deployments = rows;
			hasNext = rows.length > pageSize;
			loadedPage = currentPage;
			error = '';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load deployments';
		} finally {
			loading = false;
		}
	}

	function toggle(id: string) {
		expanded.has(id) ? expanded.delete(id) : expanded.add(id);
		expanded = new Set(expanded);
	}
</script>

<svelte:head>
	<title>Deployments · MyPaas</title>
</svelte:head>

<TableShell
	title="Deployment history"
	description="Latest build attempts, commit metadata, and rollback actions."
	{loading}
	loadingRows={3}
	error={error && deployments.length === 0 ? error : ''}
	empty={deployments.length === 0}
	emptyTitle="No deployments yet."
	emptyDescription="Trigger a deploy from the project actions panel to create the first deployment record."
	contentClass=""
	on:retry={load}
>
	<svelte:fragment slot="notice">
		{#if error}
			<div class="border-b border-amber-200 bg-amber-50 px-5 py-2 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-200">
				{error}
				<ActionButton variant="ghost" size="xs" on:click={load} className="ml-2 min-h-0 px-1 py-0 text-amber-800 hover:bg-amber-100 dark:text-amber-100 dark:hover:bg-amber-900/40">
					Retry
				</ActionButton>
			</div>
		{/if}
	</svelte:fragment>

	<div class="grid border-b border-gray-100 bg-gray-50/60 dark:border-gray-800 dark:bg-gray-900/50 sm:grid-cols-3">
		<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-b-0 sm:border-r">
			<p class="metric-label">Active pipeline</p>
			<p class="mt-1 font-mono text-lg font-semibold text-gray-950 dark:text-white">{activeCount}</p>
		</div>
		<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-b-0 sm:border-r">
			<p class="metric-label">Recoverable targets</p>
			<p class="mt-1 font-mono text-lg font-semibold text-gray-950 dark:text-white">{healthyCount}</p>
		</div>
		<div class="px-5 py-3">
			<p class="metric-label">Failed attempts</p>
			<p class="mt-1 font-mono text-lg font-semibold {failedCount > 0 ? 'text-red-600 dark:text-red-300' : 'text-gray-950 dark:text-white'}">{failedCount}</p>
		</div>
	</div>

	<div class="divide-y divide-gray-100 dark:divide-gray-800">
		{#each visibleDeployments as d}
			<div class="px-5 py-4">
				<div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_9rem_8rem_auto] lg:items-center">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<span class="font-mono text-sm font-semibold text-gray-950 dark:text-white">
								{d.commitSha?.slice(0, 8) ?? '-'}
							</span>
							<StatusBadge status={d.status} />
							<span class="rounded border border-gray-200 px-1.5 py-0.5 text-[11px] font-medium capitalize text-gray-500 dark:border-gray-800 dark:text-gray-400">
								{d.triggeredBy}
							</span>
						</div>
						<p class="mt-1 truncate text-sm text-gray-600 dark:text-gray-400">{d.commitMessage || 'No commit message'}</p>
						{#if d.errorMsg}
							<p class="mt-1 text-xs text-red-600 dark:text-red-300">{d.errorMsg}</p>
						{/if}
					</div>
					<p class="text-xs text-gray-500 dark:text-gray-400">{formatDate(d.startedAt)}</p>
					<p class="font-mono text-xs text-gray-500 dark:text-gray-400">{formatDuration(d.startedAt, d.finishedAt)}</p>
					<div class="flex shrink-0 gap-2 lg:justify-end">
						{#if d.buildLog}
							<ActionButton variant="secondary" size="xs" on:click={() => toggle(d.id)}>
								{expanded.has(d.id) ? 'Hide log' : 'Show log'}
							</ActionButton>
						{/if}
						{#if d.status === 'running' || d.status === 'stopped'}
							{#if confirmRollbackId === d.id}
								<ActionButton variant="ghost" size="xs" on:click={() => (confirmRollbackId = '')}>
									Cancel
								</ActionButton>
								<ActionButton
									variant="danger"
									size="xs"
									on:click={() => handleRollback(d.id)}
									disabled={rollingBackId !== '' && rollingBackId !== d.id}
									loading={rollingBackId === d.id}
									loadingLabel="Rolling back..."
								>
									Confirm rollback
								</ActionButton>
							{:else}
								<ActionButton
									variant="secondary"
									size="xs"
									on:click={() => requestRollback(d.id)}
									disabled={rollingBackId !== ''}
								>
									Rollback
								</ActionButton>
							{/if}
						{/if}
					</div>
				</div>
				{#if expanded.has(d.id) && d.buildLog}
					<pre class="mt-4 max-h-80 overflow-auto rounded-md border border-gray-800 bg-gray-950 p-3 text-xs leading-5 text-gray-100">{d.buildLog}</pre>
				{/if}
			</div>
		{/each}
	</div>

	<svelte:fragment slot="footer">
		<Pagination bind:page={currentPage} {pageSize} totalShown={visibleDeployments.length} {hasNext} {loading} label="Deployments" />
	</svelte:fragment>
</TableShell>
