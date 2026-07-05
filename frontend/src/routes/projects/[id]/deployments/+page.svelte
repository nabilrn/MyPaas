<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { Deployment } from '$types';

	let deployments: Deployment[] = [];
	let loading = true;
	let expanded = new Set<string>();
	let rollingBackId = '';

	async function handleRollback(id: string) {
		if (!window.confirm('Rollback to this deployment?')) return;
		rollingBackId = id;
		try {
			await api.deployments.rollback(id);
			toast.success('Rollback completed');
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
		void load();
		const id = setInterval(load, 3000);
		return () => clearInterval(id);
	});

	async function load() {
		try {
			deployments = await api.deployments.list($page.params.id);
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

<section class="surface overflow-hidden">
	<div class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-gray-800">
		<div>
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Deployment history</h2>
			<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Latest build attempts, commit metadata, and rollback actions.</p>
		</div>
	</div>

	{#if loading}
		<div class="space-y-3 p-5">
			{#each [1, 2, 3] as _}
				<div class="h-14 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
			{/each}
		</div>
	{:else if deployments.length === 0}
		<p class="p-6 text-center text-sm text-gray-500 dark:text-gray-400">No deployments yet.</p>
	{:else}
		<div class="divide-y divide-gray-100 dark:divide-gray-800">
			{#each deployments as d}
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
								<button
									on:click={() => toggle(d.id)}
									class="inline-flex min-h-8 items-center rounded-md border border-gray-300 bg-white px-2.5 py-1.5 text-xs font-medium text-gray-800 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-900"
								>
									{expanded.has(d.id) ? 'Hide log' : 'Show log'}
								</button>
							{/if}
							{#if d.status === 'running' || d.status === 'stopped'}
								<ActionButton
									variant="secondary"
									size="xs"
									on:click={() => handleRollback(d.id)}
									disabled={rollingBackId !== '' && rollingBackId !== d.id}
									loading={rollingBackId === d.id}
									loadingLabel="Rolling back..."
								>
									Rollback
								</ActionButton>
							{/if}
						</div>
					</div>
					{#if expanded.has(d.id) && d.buildLog}
						<pre class="mt-4 max-h-80 overflow-auto rounded-md border border-gray-800 bg-gray-950 p-3 text-xs leading-5 text-gray-100">{d.buildLog}</pre>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</section>
