<script lang="ts">
	import { ChevronDown, ChevronUp, RotateCcw, X } from '@lucide/svelte';
	import { onMount, tick } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import IconButton from '$components/IconButton.svelte';
	import Pagination from '$components/Pagination.svelte';
	import StatusBadge from '$components/StatusBadge.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import { expandFocusedDeployment, normalizeDeploymentFocus, pinFocusedDeployment } from '$lib/utils/deploymentFocus';
	import { canRollbackDeployment, deploymentHistoryLabel, isPipelineActive } from '$lib/utils/deploymentHistory';
	import type { Deployment, Project } from '$types';

	const pageSize = 20;
	let deployments: Deployment[] = [];
	let project: Project | null = null;
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
	$: recoverableCount = visibleDeployments.filter((item) => canRollbackDeployment(item.status, item.id, project?.activeDeploymentId)).length;
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
	$: if (mounted && currentPage !== loadedPage && !loadInFlight) void load();

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

	function deploymentSource(deployment: Deployment): string {
		return deployment.commitSha?.slice(0, 8) ?? deployment.imageTag ?? '-';
	}

	function deploymentSummary(deployment: Deployment): string {
		return deployment.commitMessage || (deployment.imageTag ? 'Container image deployment' : 'No source metadata');
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
		const projectId = $page.params.id ?? '';
		const foreground = loadedPage === -1 || requestedPage !== loadedPage;
		if (foreground) loading = true;
		try {
			const [rows, currentProject] = await Promise.all([
				api.deployments.list(projectId, requestedPage, pageSize, true),
				api.projects.get(projectId)
			]);
			const focused = requestedPage === 0 ? await resolveFocusedDeployment(rows, requestedFocusId, projectId) : null;
			if (requestedPage !== currentPage || requestedFocusId !== focusId) return;
			deployments = pinFocusedDeployment(rows, focused);
			project = currentProject;
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
	description="Latest deployment attempts, source metadata, and rollback actions."
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
			<div class="alert-warning mx-4 mt-3 flex-wrap items-center justify-between">
				<span class="min-w-0 flex-1">{error}</span>
				<ActionButton variant="ghost" size="xs" on:click={load}>
					<RotateCcw slot="icon" class="h-3.5 w-3.5" />
					Retry
				</ActionButton>
			</div>
		{/if}
	</svelte:fragment>

	<div class="grid border-b border-gray-100 bg-gray-50/50 dark:border-neutral-800 dark:bg-neutral-900/40 sm:grid-cols-3">
		<div class="border-b border-gray-100 p-4 dark:border-neutral-800 sm:border-b-0 sm:border-r">
			<p class="metric-label">Active pipeline</p>
			<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{activeCount}</p>
		</div>
		<div class="border-b border-gray-100 p-4 dark:border-neutral-800 sm:border-b-0 sm:border-r">
			<p class="metric-label">Recoverable targets</p>
			<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{recoverableCount}</p>
		</div>
		<div class="p-4">
			<p class="metric-label">Failed attempts</p>
			<p class="metric-value mt-1 text-xl font-semibold {failedCount > 0 ? 'text-red-600 dark:text-red-300' : 'text-gray-950 dark:text-white'}">{failedCount}</p>
		</div>
	</div>

	<div class="divide-y divide-gray-100 dark:divide-neutral-800">
		{#each visibleDeployments as d}
			<div id={`deployment-${d.id}`} class={`scroll-mt-6 px-4 py-3 transition-colors ${focusId === d.id ? 'bg-gray-50 dark:bg-neutral-900/70' : ''}`} aria-current={focusId === d.id ? 'true' : undefined}>
				<div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_11rem_6rem_auto] lg:items-center">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
							<span class="max-w-full truncate font-mono text-sm font-semibold text-gray-950 dark:text-white" title={deploymentSource(d)}>{deploymentSource(d)}</span>
							<StatusBadge status={d.status} label={deploymentHistoryLabel(d.status, d.id, project?.activeDeploymentId, project?.status)} />
							<span class="text-xs capitalize text-gray-500 dark:text-gray-400">{d.triggeredBy}</span>
						</div>
						<p class="mt-0.5 truncate text-sm text-gray-600 dark:text-gray-400">{deploymentSummary(d)}</p>
						{#if d.errorMsg}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{d.errorMsg}</p>{/if}
					</div>
					<p class="text-sm text-gray-500 dark:text-gray-400">{formatDate(d.startedAt)}</p>
					<p class="metric-value font-mono text-sm text-gray-500 dark:text-gray-400">{formatDuration(d.startedAt, d.finishedAt)}</p>
					<div class="flex shrink-0 flex-wrap gap-2 lg:justify-end">
						<IconButton label={`${expanded.has(d.id) ? 'Hide' : 'Show'} deployment log for ${deploymentSource(d)}`} variant="ghost" on:click={() => toggle(d.id)}>
							{#if expanded.has(d.id)}<ChevronUp class="h-4 w-4" aria-hidden="true" />{:else}<ChevronDown class="h-4 w-4" aria-hidden="true" />{/if}
						</IconButton>
						{#if canRollbackDeployment(d.status, d.id, project?.activeDeploymentId)}
							{#if confirmRollbackId === d.id}
								<ActionButton variant="ghost" size="xs" on:click={() => (confirmRollbackId = '')} disabled={rollingBackId === d.id}>
									<X slot="icon" class="h-3.5 w-3.5" />
									Cancel
								</ActionButton>
								<ActionButton variant="danger" size="xs" on:click={() => handleRollback(d.id)} disabled={rollingBackId !== '' && rollingBackId !== d.id} loading={rollingBackId === d.id} loadingLabel="Rolling back">
									<RotateCcw slot="icon" class="h-3.5 w-3.5" />
									Confirm rollback
								</ActionButton>
							{:else}
								<ActionButton variant="ghostDanger" size="xs" on:click={() => requestRollback(d.id)} disabled={rollingBackId !== ''}>
									<RotateCcw slot="icon" class="h-3.5 w-3.5" />
									Rollback
								</ActionButton>
							{/if}
						{/if}
					</div>
				</div>

				{#if expanded.has(d.id)}
					<div class="console-surface mt-3 overflow-hidden">
						<div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-800 px-3 py-2">
							<p class="font-mono text-xs font-semibold uppercase tracking-wide text-gray-300">Deployment output</p>
							<p class="text-xs text-gray-500">
								{#if isPipelineActive(d.status)}
									{d.buildLog ? 'Live · refreshes every 3 seconds' : 'Waiting for output'}
								{:else}
									{d.buildLog ? 'Final output' : 'No output captured'}
								{/if}
							</p>
						</div>
						{#if d.buildLog}
							<pre class="max-h-80 overflow-auto p-3">{d.buildLog}</pre>
						{:else}
							<div class="px-3 py-6 text-center text-sm text-gray-400" role={isPipelineActive(d.status) ? 'status' : undefined}>
								{isPipelineActive(d.status) ? `Pipeline is ${d.status}. Deployment output will appear here automatically.` : 'This deployment did not produce build output.'}
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
