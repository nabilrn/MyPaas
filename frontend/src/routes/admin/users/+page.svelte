<script lang="ts">
	import { Plus, RefreshCw, Trash2 } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import Pagination from '$components/Pagination.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
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
	let addRole: 'owner' | 'collaborator' = 'collaborator';
	let adding = false;
	let savingUser = false;
	let removingUserId = '';
	let confirmRemoveUserId = '';

	$: pageStart = currentPage * pageSize;
	$: visibleUsers = users.slice(pageStart, pageStart + pageSize);
	$: hasNext = pageStart + pageSize < users.length;
	$: canAdd = Boolean(addEmail.trim() && !savingUser);
	$: addDisabledReason = addEmail.trim() ? '' : 'Email is required before adding a user.';

	onMount(load);

	async function load() {
		loading = true;
		error = '';
		try {
			users = await api.admin.listUsers();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load users';
		} finally {
			loading = false;
		}
	}

	async function handleAdd() {
		if (!addEmail.trim() || savingUser) return;
		savingUser = true;
		try {
			await api.admin.addUser({ email: addEmail.trim(), role: addRole });
			toast.success('User added');
			adding = false;
			addEmail = '';
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to add user');
		} finally {
			savingUser = false;
		}
	}

	function requestRemove(id: string) {
		confirmRemoveUserId = id;
	}

	async function handleRemove(id: string, email: string) {
		if (removingUserId) return;
		removingUserId = id;
		try {
			await api.admin.removeUser(id);
			users = users.filter((user) => user.id !== id);
			if (currentPage > 0 && currentPage * pageSize >= users.length) currentPage -= 1;
			confirmRemoveUserId = '';
			toast.success(`Removed ${email}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove user');
		} finally {
			removingUserId = '';
		}
	}

	function initial(email: string) {
		return email.trim().slice(0, 1).toUpperCase() || '?';
	}

	function formatDate(value: string | null | undefined) {
		return value ? new Date(value).toLocaleDateString() : '—';
	}
</script>

<svelte:head>
	<title>Users · MyPaas Admin</title>
</svelte:head>

<div class="page-shell py-6">
	<div class="mb-5 flex flex-wrap items-center justify-between gap-3">
		<p class="text-sm text-gray-500 dark:text-gray-400">Only whitelisted users can sign in with GitHub OAuth.</p>
		<div class="flex flex-wrap items-center gap-2">
			<ActionButton variant="secondary" size="sm" loading={loading} loadingLabel="Refreshing" on:click={load}>
				<RefreshCw slot="icon" class="h-4 w-4" />
				Refresh
			</ActionButton>
			<ActionButton variant="primary" size="sm" disabled={adding} on:click={() => (adding = true)}>
				<Plus slot="icon" class="h-4 w-4" />
				Add user
			</ActionButton>
		</div>
	</div>

	{#if adding}
		<SectionPanel title="Add user" description="Grant GitHub OAuth access to a collaborator or owner." className="mb-5">
			<form class="grid gap-3 md:grid-cols-[minmax(0,1fr)_12rem_auto_auto] md:items-start" on:submit|preventDefault={handleAdd}>
				<div>
					<label class="field-label" for="user-email">Email</label>
					<input id="user-email" type="email" bind:value={addEmail} placeholder="user@example.com" class="field w-full" />
					{#if addDisabledReason}<p class="field-hint">{addDisabledReason}</p>{/if}
				</div>
				<div>
					<label class="field-label" for="user-role">Role</label>
					<select id="user-role" bind:value={addRole} class="field w-full">
						<option value="collaborator">Collaborator</option>
						<option value="owner">Owner</option>
					</select>
				</div>
				<ActionButton variant="primary" type="submit" className="md:mt-[1.45rem]" loading={savingUser} loadingLabel="Adding" disabled={!canAdd}>
					<Plus slot="icon" class="h-4 w-4" />
					Add
				</ActionButton>
				<ActionButton variant="ghost" className="md:mt-[1.45rem]" on:click={() => (adding = false)} disabled={savingUser}>Cancel</ActionButton>
			</form>
		</SectionPanel>
	{/if}

	<TableShell
		title="Whitelisted users"
		description="Manage who can access this MyPaaS control plane."
		{loading}
		loadingRows={3}
		{error}
		empty={users.length === 0}
		emptyTitle="No users are whitelisted yet."
		emptyDescription="Add a collaborator or owner to allow GitHub OAuth sign-in."
		on:retry={load}
	>
		<table class="data-table">
			<thead>
				<tr>
					<th>User</th>
					<th>Role</th>
					<th>Last login</th>
					<th>Added</th>
					<th class="text-right">Action</th>
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
									<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-gray-200 bg-gray-50 text-xs font-semibold text-gray-500 dark:border-neutral-800 dark:bg-neutral-900 dark:text-gray-300">{initial(user.email)}</div>
								{/if}
								<div class="min-w-0">
									<p class="truncate text-sm font-medium text-gray-950 dark:text-white">{user.githubUsername ?? 'Not logged in yet'}</p>
									<p class="truncate text-xs text-gray-500 dark:text-gray-400">{user.email}</p>
								</div>
							</div>
						</td>
						<td>
							<span class="inline-flex items-center gap-2 text-sm capitalize text-gray-700 dark:text-gray-300">
								<span class="status-dot {user.role === 'owner' ? 'bg-gray-950 dark:bg-white' : 'bg-gray-400 dark:bg-gray-500'}"></span>
								{user.role}
							</span>
						</td>
						<td class="text-sm">{formatDate(user.lastLoginAt)}</td>
						<td class="text-sm">{formatDate(user.createdAt)}</td>
						<td class="text-right">
							<div class="flex justify-end gap-2">
								{#if confirmRemoveUserId === user.id}
									<ActionButton variant="ghost" size="xs" on:click={() => (confirmRemoveUserId = '')}>Cancel</ActionButton>
									<ActionButton variant="danger" size="xs" on:click={() => handleRemove(user.id, user.email)} disabled={removingUserId !== '' && removingUserId !== user.id} loading={removingUserId === user.id} loadingLabel="Removing">
										<Trash2 slot="icon" class="h-3.5 w-3.5" />
										Remove
									</ActionButton>
								{:else}
									<ActionButton variant="ghostDanger" size="xs" on:click={() => requestRemove(user.id)} disabled={removingUserId !== ''}>
										<Trash2 slot="icon" class="h-3.5 w-3.5" />
										Remove
									</ActionButton>
								{/if}
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<svelte:fragment slot="footer">
			<Pagination bind:page={currentPage} {pageSize} totalShown={visibleUsers.length} {hasNext} {loading} label="Users" />
		</svelte:fragment>
	</TableShell>
</div>
