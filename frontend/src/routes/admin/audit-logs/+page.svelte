<script lang="ts">
	import { onMount } from 'svelte';
	import IconButton from '$components/IconButton.svelte';
	import PageHeader from '$components/PageHeader.svelte';
	import Pagination from '$components/Pagination.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api } from '$api';
	import type { AuditLog } from '$types';

	const pageSize = 25;
	let rows: AuditLog[] = [];
	let loading = true;
	let error = '';
	let expanded = new Set<string>();
	let currentPage = 0;
	let hasNext = false;
	let mounted = false;
	let loadedPage = -1;

	$: visibleRows = rows.slice(0, pageSize);
	$: if (mounted && currentPage !== loadedPage) {
		void load();
	}

	onMount(() => {
		mounted = true;
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			rows = await api.admin.listAuditLogs(currentPage, pageSize, true);
			hasNext = rows.length > pageSize;
			loadedPage = currentPage;
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

	function statusClass(status: unknown) {
		const code = Number(status);
		if (!Number.isFinite(code)) {
			return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300';
		}
		if (code >= 500) {
			return 'border-red-500/30 bg-red-50 text-red-700 dark:border-red-500/40 dark:bg-red-950/30 dark:text-red-200';
		}
		if (code >= 400) {
			return 'border-yellow-500/30 bg-yellow-50 text-yellow-800 dark:border-yellow-500/40 dark:bg-yellow-950/30 dark:text-yellow-100';
		}
		if (code >= 200 && code < 300) {
			return 'border-brand-500/30 bg-brand-50 text-brand-900 dark:border-brand-500/40 dark:bg-brand-500/10 dark:text-brand-100';
		}
		return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300';
	}

	function formatDateTime(value: string) {
		return new Date(value).toLocaleString();
	}
</script>

<svelte:head>
	<title>Audit Logs · MyPaas Admin</title>
</svelte:head>

<div class="page-shell py-6">
	<PageHeader title="Audit logs" description="Recent authenticated changes across projects, deployments, env vars, and admin users.">
		<svelte:fragment slot="actions">
			<IconButton label="Refresh audit logs" variant="brand" {loading} on:click={load}>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M20 11a8.1 8.1 0 00-15.5-3M4 4v4h4m-4 5a8.1 8.1 0 0015.5 3M20 20v-4h-4" />
				</svg>
			</IconButton>
			<IconButton label="User whitelist" href="/admin/users" variant="default">
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2M12 11a4 4 0 100-8 4 4 0 000 8" />
				</svg>
			</IconButton>
		</svelte:fragment>
	</PageHeader>

	<TableShell
		title="Event stream"
		description="Review what changed, which resource was touched, and the response code returned by the control plane."
		{loading}
		loadingRows={3}
		{error}
		empty={rows.length === 0}
		emptyTitle="No audit logs yet."
		emptyDescription="Authenticated admin and deployment events will appear here after changes are made."
		on:retry={load}
	>
		<table class="min-w-full divide-y divide-gray-100 dark:divide-gray-800">
			<thead>
				<tr class="bg-gray-50/70 dark:bg-gray-900/70">
					<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Action</th>
					<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Resource</th>
					<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Status</th>
					<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Time</th>
					<th class="px-5 py-3"></th>
				</tr>
			</thead>
			<tbody class="divide-y divide-gray-100 dark:divide-gray-800">
				{#each visibleRows as row}
					<tr class="align-top hover:bg-gray-50/80 dark:hover:bg-gray-900/70">
						<td class="px-5 py-4">
							<p class="font-mono text-sm font-medium text-gray-950 dark:text-white">{row.action}</p>
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{row.ipAddress ?? 'unknown ip'}</p>
						</td>
						<td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">
							{row.resourceType ?? '—'}
							{#if row.resourceId}
								<span class="block max-w-56 truncate font-mono text-xs text-gray-400" title={row.resourceId}>{row.resourceId}</span>
							{/if}
						</td>
						<td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">
							<span class={`inline-flex rounded-md border px-2 py-1 font-mono text-xs font-medium ${statusClass(row.metadata.status)}`}>
								{String(row.metadata.status ?? '—')}
							</span>
						</td>
						<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
							{formatDateTime(row.createdAt)}
						</td>
						<td class="px-5 py-4 text-right">
							<IconButton label={`${expanded.has(row.id) ? 'Hide' : 'Show'} audit log details`} variant="ghost" on:click={() => toggle(row.id)}>
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									{#if expanded.has(row.id)}
										<path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7" />
									{:else}
										<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
									{/if}
								</svg>
							</IconButton>
						</td>
					</tr>
					{#if expanded.has(row.id)}
						<tr class="bg-gray-50 dark:bg-gray-950/40">
							<td colspan="5" class="px-5 py-4">
								<div class="grid gap-3 lg:grid-cols-[14rem_minmax(0,1fr)]">
									<div class="space-y-2 text-xs text-gray-500 dark:text-gray-400">
										<p>
											<span class="block font-medium text-gray-700 dark:text-gray-200">IP address</span>
											<span class="font-mono">{row.ipAddress ?? 'unknown'}</span>
										</p>
										<p>
											<span class="block font-medium text-gray-700 dark:text-gray-200">User agent</span>
											<span class="line-clamp-4 break-words">{row.userAgent ?? 'unknown'}</span>
										</p>
									</div>
									<pre class="max-h-80 overflow-auto rounded-md border border-gray-800 bg-gray-950 p-3 text-xs leading-5 text-gray-100">{JSON.stringify(row.metadata, null, 2)}</pre>
								</div>
							</td>
						</tr>
					{/if}
				{/each}
			</tbody>
		</table>
		<svelte:fragment slot="footer">
			<Pagination bind:page={currentPage} {pageSize} totalShown={visibleRows.length} {hasNext} {loading} label="Audit logs" />
		</svelte:fragment>
	</TableShell>
</div>
