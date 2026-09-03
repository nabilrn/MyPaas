<script lang="ts">
	import { Plus, RefreshCw, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import Pagination from '$components/Pagination.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { User } from '$types';

	const pageSize = 10;
	let users: User[] = [];
	let loading = true;
	let error = '';
	let currentPage = 0;
	let addEmail = '';
	let adding = false;
	let savingUser = false;

	$: pageStart = currentPage * pageSize;
	$: visibleUsers = users.slice(pageStart, pageStart + pageSize);
	$: hasNext = pageStart + pageSize < users.length;
	$: canAdd = Boolean(addEmail.trim() && !savingUser);

	onMount(load);

	async function load() {
		loading = true;
		error = '';
		try {
			users = await api.admin.listUsers();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load owners';
		} finally {
			loading = false;
		}
	}

	async function handleAdd() {
		if (!addEmail.trim() || savingUser) return;
		savingUser = true;
		try {
			await api.admin.addUser({ email: addEmail.trim() });
			toast.success('Owner added');
			adding = false;
			addEmail = '';
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to add owner');
		} finally {
			savingUser = false;
		}
	}

	function closeAddOwner() {
		if (savingUser) return;
		adding = false;
		addEmail = '';
	}

	function initial(email: string) {
		return email.trim().slice(0, 1).toUpperCase() || '?';
	}

	function formatDate(value: string | null | undefined) {
		return value ? new Date(value).toLocaleDateString() : '—';
	}
</script>

<svelte:head>
	<title>Users · MyPaaS Admin</title>
</svelte:head>

<div class="page-shell">
	<TableShell
		{loading}
		loadingRows={3}
		{error}
		empty={users.length === 0}
		emptyTitle="No owners yet."
		emptyDescription="Add an owner to allow sign-in."
		on:retry={load}
	>
		<svelte:fragment slot="actions">
			<div class="flex items-center gap-2">
				<ActionButton variant="secondary" size="xs" loading={loading} loadingLabel="Refreshing" on:click={load}>
					<RefreshCw slot="icon" class="h-3.5 w-3.5" />
					Refresh
				</ActionButton>
				<ActionButton variant="primary" size="xs" on:click={() => (adding = true)}>
					<Plus slot="icon" class="h-3.5 w-3.5" />
					Add owner
				</ActionButton>
			</div>
		</svelte:fragment>

		<table class="data-table table-fixed min-w-[42rem]">
			<colgroup>
				<col class="w-[46%]" />
				<col class="w-[16%]" />
				<col class="w-[19%]" />
				<col class="w-[19%]" />
			</colgroup>
			<thead>
				<tr>
					<th>User</th>
					<th>Role</th>
					<th>Last login</th>
					<th>Added</th>
				</tr>
			</thead>
			<tbody>
				{#each visibleUsers as user}
					<tr>
						<td>
							<div class="flex min-w-0 items-center gap-3">
								{#if user.avatarUrl}
									<img src={user.avatarUrl} alt="" class="h-8 w-8 shrink-0 rounded-full object-cover" />
								{:else}
									<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-gray-200 text-xs font-semibold text-gray-500 dark:border-neutral-800 dark:text-gray-300">{initial(user.email)}</div>
								{/if}
								<div class="min-w-0">
									<p class="truncate text-sm font-medium text-gray-950 dark:text-white" title={user.githubUsername ?? 'Not logged in yet'}>{user.githubUsername ?? 'Not logged in yet'}</p>
									<p class="truncate text-xs text-gray-500 dark:text-gray-400" title={user.email}>{user.email}</p>
								</div>
							</div>
						</td>
						<td class="whitespace-nowrap text-center">
							<span class="inline-flex items-center gap-2 text-sm capitalize text-gray-700 dark:text-gray-300">
								<span class="status-dot bg-gray-950 dark:bg-white"></span>
								{user.role}
							</span>
						</td>
						<td class="whitespace-nowrap text-center text-sm tabular-nums">{formatDate(user.lastLoginAt)}</td>
						<td class="whitespace-nowrap text-center text-sm tabular-nums">{formatDate(user.createdAt)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<svelte:fragment slot="footer">
			<Pagination bind:page={currentPage} {pageSize} totalShown={visibleUsers.length} {hasNext} {loading} label="Owners" />
		</svelte:fragment>
	</TableShell>
</div>

{#if adding}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close add owner" on:click={closeAddOwner}></button>
		<div class="overlay relative w-full max-w-lg" role="dialog" aria-modal="true" aria-labelledby="add-owner-title">
			<div class="panel-header flex items-start justify-between gap-3">
				<h2 id="add-owner-title" class="panel-title">Add owner</h2>
				<ActionButton variant="ghost" size="xs" on:click={closeAddOwner} disabled={savingUser}><X slot="icon" class="h-4 w-4" />Close</ActionButton>
			</div>
			<form class="space-y-4 p-4" on:submit|preventDefault={handleAdd}>
				<div>
					<label class="field-label" for="user-email">Email</label>
					<input id="user-email" type="email" required bind:value={addEmail} placeholder="user@example.com" class="field w-full" />
				</div>
				<div class="flex justify-end gap-2">
					<ActionButton variant="ghost" on:click={closeAddOwner} disabled={savingUser}>Cancel</ActionButton>
					<ActionButton variant="primary" type="submit" loading={savingUser} loadingLabel="Adding" disabled={!canAdd}>Add owner</ActionButton>
				</div>
			</form>
		</div>
	</div>
{/if}
