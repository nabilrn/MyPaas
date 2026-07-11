<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import IconButton from '$components/IconButton.svelte';
	import Pagination from '$components/Pagination.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Deployment } from '$types';
	import { expandFocusedDeployment, normalizeDeploymentFocus, pinFocusedDeployment } from '$lib/utils/deploymentFocus';

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
	let loadInFlight = false;
	let focusId = '';
	let appliedFocusId = '';
	let revealedFocusId = '';

	$: visibleDeployments = deployments.slice(0, pageSize);
	$: activeCount = visibleDeployments.filter((item) => isPipelineActive(item.status)).length;
	$: healthyCount = visibleDeployments.filter((item) => ['running', 'stopped', 'rolled_back'].includes(item.status)).length;
	$: failedCount = visibleDeployments.filter((item) => item.status === 'failed').length;
	$: focusId = normalizeDeploymentFocus($page.url.searchParams.get('focus'));
	$: if (focusId !== appliedFocusId) {
		appliedFocusId = focusId;
		loadedPage = -1;
		revealedFocusId = '';
		if (focusId) {
			currentPage = 0;
			expanded = expandFocusedDeployment(expanded, focusId);
		}
	}
	$: if (mounted && currentPage !== loadedPage && !loadInFlight) {
		void load();
	}

	function isPipelineActive(status: Deployment['status']) {
		return status === 'queued' || status === 'cloning' || status === 'building' || status === 'starting';
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
		const s = Math.floor(ms / 1000);
		return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
	}

	function formatDate(value: string) {
		return new Date(value).toLocaleString();
	}

	onMount(() => {
		mounted = true;
		void load();
		const id = setInterval(() => void load(), 3000);
		return () => clearInterval(id);
	});

	async function load() {
		if (loadInFlight) return;
		loadInFlight = true;
		const requestedPage = currentPage;
		const requestedFocusId = focusId;
		const projectId = $page.params.id;
		const foreground = loadedPage === -1 || requestedPage !== loadedPage;
		if (foreground) loading = true;
		try {
			const rows = await api.deployments.list(projectId, requestedPage, pageSize, true);
			const focused = requestedPage === 0 ? await resolveFocusedDeployment(rows, requestedFocusId, projectId) : null;
			if (requestedPage !== currentPage || requestedFocusId !== focusId) return;
			deployments = pinFocusedDeployment(rows, focused);
			hasNext = rows.length > pageSize;
			loadedPage = requestedPage;
			error = '';
			void revealFocusedDeployment();
		} catch (err) {
			if (requestedPage === currentPage && requestedFocusId === focusId) {
				error = err instanceof Error ? err.message : 'Failed to load deployments';
				loadedPage = requestedPage;
			}
		} finally {
			if (foreground) loading = false;
			loadInFlight = false;
		}
	}

	async function resolveFocusedDeployment(rows: Deployment[], requestedFocusId: string, projectId: string): Promise<Deployment | null> {
		if (!requestedFocusId) return null;
		const visible = rows.find((item) => item.id === requestedFocusId);
		if (visible) return visible;
		try {
			const focused = await api.deployments.get(requestedFocusId);
			return focused.projectId === projectId ? focused : null;
		} catch {
			return null;
		}
	}

	async function revealFocusedDeployment() {
		const targetId = focusId;
		if (!targetId || targetId === revealedFocusId || !deployments.some((item) => item.id === targetId)) return;
		await tick();
		if (targetId !== focusId) return;
		const target = document.getElementById(`deployment-${targetId}`);
		if (!target) return;
		target.scrollIntoView({ block: 'nearest' });
		revealedFocusId = targetId;
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
				<ActionButton variant="ghost" size="xs" on:click={load} className="ml-2 min-h-0 px-1 py-0 text-amber-800 hover:bg-amber-100 dark:text-amber-100 dark:hover:bg-amber-900/40">Retry</ActionButton>
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
			<div
				id={`deployment-${d.id}`}
				class={`scroll-mt-6 px-5 py-4 transition-colors ${focusId === d.id ? 'bg-brand-50/60 dark:bg-brand-900/20' : ''}`}
				aria-current={focusId === d.id ? 'true' : undefined}
			>
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
						<IconButton label={`${expanded.has(d.id) ? 'Hide' : 'Show'} build log for ${d.commitSha?.slice(0, 8) ?? 'deployment'}`} on:click={() => toggle(d.id)}>
							{#if expanded.has(d.id)}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7" />
								</svg>
							{:else}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
								</svg>
							{/if}
						</IconButton>
						{#if d.status === 'running' || d.status === 'stopped'}
							{#if confirmRollbackId === d.id}
								<ActionButton variant="ghost" size="xs" on:click={() => (confirmRollbackId = '')}>Cancel</ActionButton>
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
								<IconButton label={`Rollback deployment ${d.commitSha?.slice(0, 8) ?? d.id}`} variant="danger" on:click={() => requestRollback(d.id)} disabled={rollingBackId !== ''}>
									<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" d="M3 12a9 9 0 1015.5-6.2M3 4v5h5" />
									</svg>
								</IconButton>
							{/if}
						{/if}
					</div>
				</div>
				{#if expanded.has(d.id)}
					<div class="mt-4 overflow-hidden rounded-md border border-gray-800 bg-gray-950">
						<div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-800 px-3 py-2">
							<p class="font-mono text-[11px] font-semibold uppercase tracking-wider text-gray-300">Build output</p>
							<p class="text-[11px] text-gray-500">
								{#if isPipelineActive(d.status)}
									{d.buildLog ? 'Live, refreshes every 3 seconds' : 'Waiting for output'}
								{:else}
									{d.buildLog ? 'Final output' : 'No output captured'}
								{/if}
							</p>
						</div>
						{#if d.buildLog}
							<pre class="max-h-80 overflow-auto p-3 text-xs leading-5 text-gray-100">{d.buildLog}</pre>
						{:else}
							<div class="px-3 py-6 text-center text-xs leading-5 text-gray-400" role={isPipelineActive(d.status) ? 'status' : undefined}>
								{isPipelineActive(d.status) ? `Pipeline is ${d.status}. Build output will appear here automatically.` : 'This deployment did not produce build output.'}
							</div>
						{/if}
					</div>
				{/if}
			</div>
		{/each}
	</div>

	<svelte:fragment slot="footer">
		<Pagination bind:page={currentPage} {pageSize} totalShown={visibleDeployments.length} {hasNext} {loading} label="Deployments" />
	</svelte:fragment>
</TableShell>
