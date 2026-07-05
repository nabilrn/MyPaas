<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$api';
	import type { AuditLog } from '$types';

	let rows: AuditLog[] = [];
	let loading = true;
	let error = '';
	let expanded = new Set<string>();

	onMount(load);

	async function load() {
		loading = true;
		error = '';
		try {
			rows = await api.admin.listAuditLogs();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load audit logs';
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
	<title>Audit Logs · MyPaas Admin</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-4 py-7 sm:px-6">
	<div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<p class="text-xs font-medium uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">Admin</p>
			<h1 class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">Audit logs</h1>
			<p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
				Recent authenticated changes across projects, deployments, env vars, and admin users.
			</p>
		</div>
		<a
			href="/admin/users"
			class="inline-flex min-h-9 items-center justify-center rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-800 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-900"
		>
			Users
		</a>
	</div>

	<div class="surface overflow-hidden">
		{#if loading}
			<p class="p-5 text-sm text-gray-500 dark:text-gray-400">Loading audit logs...</p>
		{:else if error}
			<div class="p-5 text-sm text-red-600 dark:text-red-400">
				{error}
				<button on:click={load} class="ml-3 font-medium underline">Retry</button>
			</div>
		{:else if rows.length === 0}
			<p class="p-5 text-sm text-gray-500 dark:text-gray-400">No audit logs yet.</p>
		{:else}
			<table class="min-w-full divide-y divide-gray-100 dark:divide-gray-800">
				<thead>
					<tr class="bg-gray-50/70 dark:bg-gray-900/70">
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Action</th>
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Resource</th>
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Status</th>
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Time</th>
						<th class="px-5 py-3"></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100 dark:divide-gray-800">
					{#each rows as row}
						<tr class="align-top hover:bg-gray-50/80 dark:hover:bg-gray-900/70">
							<td class="px-5 py-4">
								<p class="font-mono text-sm font-medium text-gray-950 dark:text-white">{row.action}</p>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{row.ipAddress ?? 'unknown ip'}</p>
							</td>
							<td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">
								{row.resourceType ?? '—'}
								{#if row.resourceId}
									<span class="block max-w-48 truncate font-mono text-xs text-gray-400">{row.resourceId}</span>
								{/if}
							</td>
							<td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">
								{String(row.metadata.status ?? '—')}
							</td>
							<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
								{new Date(row.createdAt).toLocaleString()}
							</td>
							<td class="px-5 py-4 text-right">
								<button
									type="button"
									on:click={() => toggle(row.id)}
									class="text-sm font-medium text-gray-600 hover:text-gray-950 dark:text-gray-300 dark:hover:text-white"
								>
									{expanded.has(row.id) ? 'Hide' : 'Details'}
								</button>
							</td>
						</tr>
						{#if expanded.has(row.id)}
							<tr class="bg-gray-50 dark:bg-gray-950/40">
								<td colspan="5" class="px-5 py-4">
									<pre class="overflow-auto rounded-md bg-gray-950 p-3 text-xs text-gray-100">{JSON.stringify(row.metadata, null, 2)}</pre>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>
