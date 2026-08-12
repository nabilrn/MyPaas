<script lang="ts">
	import { ChevronDown, ChevronUp, RefreshCw, User } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import ActionLink from '$components/ActionLink.svelte';
	import IconButton from '$components/IconButton.svelte';
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
	$: if (mounted && currentPage !== loadedPage) void load();

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

	function statusDotClass(status: unknown) {
		const code = Number(status);
		if (!Number.isFinite(code)) return 'bg-gray-400 dark:bg-gray-500';
		if (code >= 500) return 'bg-red-500';
		if (code >= 400) return 'bg-amber-500';
		if (code >= 200 && code < 300) return 'bg-emerald-500';
		return 'bg-gray-400 dark:bg-gray-500';
	}

	function statusTextClass(status: unknown) {
		const code = Number(status);
		if (!Number.isFinite(code)) return 'text-gray-600 dark:text-gray-300';
		if (code >= 500) return 'text-red-700 dark:text-red-300';
		if (code >= 400) return 'text-amber-700 dark:text-amber-200';
		return 'text-gray-700 dark:text-gray-300';
	}

	function formatDateTime(value: string) {
		return new Date(value).toLocaleString();
	}
</script>

<svelte:head><title>Audit Logs · MyPaas Admin</title></svelte:head>

<div class="page-shell py-6">
	<div class="mb-5 flex flex-wrap items-center justify-between gap-3 px-5">
		<p class="max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400">Authenticated control-plane changes across projects, deployments, environment variables, and admin access.</p>
		<div class="flex flex-wrap items-center gap-2">
			<ActionButton variant="secondary" size="sm" loading={loading} loadingLabel="Refreshing" on:click={load}><RefreshCw slot="icon" class="h-4 w-4" />Refresh</ActionButton>
			<ActionLink href="/admin/users" variant="secondary" size="sm"><User slot="icon" class="h-4 w-4" />User whitelist</ActionLink>
		</div>
	</div>

	<TableShell title="Event stream" description="Review what changed, which resource was touched, and the response returned by the control plane." {loading} loadingRows={3} {error} empty={rows.length === 0} emptyTitle="No audit logs yet." emptyDescription="Authenticated admin and deployment events will appear here after changes are made." on:retry={load}>
		<table class="data-table">
			<thead><tr><th>Action</th><th>Resource</th><th>Status</th><th>Time</th><th class="w-12"><span class="sr-only">Details</span></th></tr></thead>
			<tbody>
				{#each visibleRows as row}
					<tr class="align-top">
						<td><p class="font-mono text-sm font-medium text-gray-950 dark:text-white">{row.action}</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{row.ipAddress ?? 'unknown ip'}</p></td>
						<td><p class="text-sm text-gray-700 dark:text-gray-300">{row.resourceType ?? '—'}</p>{#if row.resourceId}<p class="mt-0.5 max-w-64 truncate font-mono text-xs text-gray-400 dark:text-gray-500" title={row.resourceId}>{row.resourceId}</p>{/if}</td>
						<td><span class={`inline-flex items-center gap-2 font-mono text-sm ${statusTextClass(row.metadata.status)}`}><span class={`status-dot ${statusDotClass(row.metadata.status)}`}></span>{String(row.metadata.status ?? '—')}</span></td>
						<td class="text-sm">{formatDateTime(row.createdAt)}</td>
						<td class="text-right"><IconButton label={`${expanded.has(row.id) ? 'Hide' : 'Show'} audit log details`} variant="ghost" on:click={() => toggle(row.id)}>{#if expanded.has(row.id)}<ChevronUp class="h-4 w-4" aria-hidden="true" />{:else}<ChevronDown class="h-4 w-4" aria-hidden="true" />{/if}</IconButton></td>
					</tr>
					{#if expanded.has(row.id)}
						<tr class="!bg-gray-50/70 dark:!bg-neutral-900/50"><td colspan="5" class="!p-4"><div class="grid gap-4 lg:grid-cols-[15rem_minmax(0,1fr)]"><div class="space-y-3 text-sm text-gray-500 dark:text-gray-400"><div><p class="text-xs font-medium text-gray-700 dark:text-gray-200">IP address</p><p class="mt-1 font-mono text-xs">{row.ipAddress ?? 'unknown'}</p></div><div><p class="text-xs font-medium text-gray-700 dark:text-gray-200">User agent</p><p class="mt-1 line-clamp-4 break-words text-xs">{row.userAgent ?? 'unknown'}</p></div></div><pre class="console-surface max-h-80 overflow-auto p-3">{JSON.stringify(row.metadata, null, 2)}</pre></div></td></tr>
					{/if}
				{/each}
			</tbody>
		</table>
		<svelte:fragment slot="footer"><Pagination bind:page={currentPage} {pageSize} totalShown={visibleRows.length} {hasNext} {loading} label="Audit logs" /></svelte:fragment>
	</TableShell>
</div>
