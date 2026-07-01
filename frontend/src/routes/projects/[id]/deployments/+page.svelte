<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import StatusBadge from '$components/StatusBadge.svelte';
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
		if (!end) return '—';
		const ms = new Date(end).getTime() - new Date(start).getTime();
		const s  = Math.floor(ms / 1000);
		return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
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

<div class="rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
	<div class="border-b border-gray-200 px-5 py-4 dark:border-gray-800">
		<h2 class="font-semibold text-gray-900 dark:text-white">Deployment history</h2>
	</div>

	{#if loading}
		<p class="p-6 text-center text-sm text-gray-500 dark:text-gray-400">Loading deployments...</p>
	{:else if deployments.length === 0}
		<p class="p-6 text-center text-sm text-gray-500 dark:text-gray-400">No deployments yet.</p>
	{:else}
		<div class="divide-y divide-gray-100 dark:divide-gray-800">
			{#each deployments as d}
				<div class="px-5 py-4">
				<div class="flex items-start justify-between gap-4">
					<div class="min-w-0 flex-1">
						<div class="flex items-center gap-2">
							<span class="font-mono text-sm font-medium text-gray-900 dark:text-white">
								{d.commitSha?.slice(0, 8) ?? '—'}
							</span>
							<StatusBadge status={d.status} />
							<span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 capitalize dark:bg-gray-800 dark:text-gray-400">
								{d.triggeredBy}
							</span>
						</div>
						{#if d.commitMessage}
							<p class="mt-0.5 truncate text-sm text-gray-600 dark:text-gray-400">{d.commitMessage}</p>
						{/if}
						<p class="mt-1 text-xs text-gray-400">
							{new Date(d.startedAt).toLocaleString()} · {formatDuration(d.startedAt, d.finishedAt)}
						</p>
						{#if d.errorMsg}
							<p class="mt-1 text-xs text-red-600 dark:text-red-400">{d.errorMsg}</p>
						{/if}
					</div>

					<div class="flex shrink-0 gap-2">
					{#if d.buildLog}
						<button
							on:click={() => toggle(d.id)}
							class="rounded-md border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
						>
							{expanded.has(d.id) ? 'Hide log' : 'Show log'}
						</button>
					{/if}
					{#if d.status === 'running' || d.status === 'stopped'}
						<button
							on:click={() => handleRollback(d.id)}
							disabled={rollingBackId === d.id}
							class="shrink-0 rounded-md border border-gray-200 px-3 py-1.5 text-xs font-medium
								   text-gray-700 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"
						>
							{rollingBackId === d.id ? 'Rolling back...' : 'Rollback'}
						</button>
					{/if}
					</div>
				</div>
				{#if expanded.has(d.id) && d.buildLog}
					<pre class="mt-3 max-h-80 overflow-auto rounded-lg bg-gray-950 p-3 text-xs text-gray-100">{d.buildLog}</pre>
				{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
