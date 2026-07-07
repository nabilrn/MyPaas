<script lang="ts">
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import IconButton from '$components/IconButton.svelte';
	import PageHeader from '$components/PageHeader.svelte';
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
			users = users.filter((u) => u.id !== id);
			if (currentPage > 0 && currentPage * pageSize >= users.length) {
				currentPage -= 1;
			}
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
		return value ? new Date(value).toLocaleDateString() : '-';
	}

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
</script>

<svelte:head>
	<title>Users · MyPaas Admin</title>
</svelte:head>

<div class="page-shell py-6">
	<PageHeader
		title="User whitelist"
		description="Only listed users can sign in via GitHub OAuth."
	>
		<svelte:fragment slot="actions">
			<IconButton label="Refresh users" variant="brand" loading={loading} on:click={load}>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M20 11a8.1 8.1 0 00-15.5-3M4 4v4h4m-4 5a8.1 8.1 0 0015.5 3M20 20v-4h-4" />
				</svg>
			</IconButton>
			<IconButton label="Add user" variant="primary" disabled={adding} on:click={() => (adding = true)}>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
				</svg>
			</IconButton>
		</svelte:fragment>
	</PageHeader>

	{#if adding}
		<SectionPanel
			title="Add user to whitelist"
			description="Whitelisted users can authenticate with GitHub OAuth after they are added here."
			className="mb-5"
		>
			<form class="grid gap-3 md:grid-cols-[minmax(0,1fr)_12rem_auto_auto] md:items-start" on:submit|preventDefault={handleAdd}>
				<input type="email" bind:value={addEmail} placeholder="user@example.com" class="field w-full" />
				<select bind:value={addRole} class="field w-full">
					<option value="collaborator">Collaborator</option>
					<option value="owner">Owner</option>
				</select>
				<ActionButton variant="primary" type="submit" loading={savingUser} loadingLabel="Adding..." disabled={!canAdd}>
					Add
				</ActionButton>
				<ActionButton variant="ghost" on:click={() => (adding = false)} disabled={savingUser}>
					Cancel
				</ActionButton>
				{#if addDisabledReason}
					<p class="text-xs text-gray-500 dark:text-gray-400 md:col-span-4">{addDisabledReason}</p>
				{/if}
			</form>
		</SectionPanel>
	{/if}

	<TableShell
		title="Whitelisted users"
		description="Manage who can access the deployment control plane."
		{loading}
		loadingRows={3}
		{error}
		empty={users.length === 0}
		emptyTitle="No users are whitelisted yet."
		emptyDescription="Add a collaborator or owner to allow GitHub OAuth sign-in."
		on:retry={load}
	>
			<table class="min-w-full divide-y divide-gray-100 dark:divide-gray-800">
				<thead>
					<tr class="bg-gray-50/70 dark:bg-gray-900/70">
						<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">User</th>
						<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Role</th>
						<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Last login</th>
						<th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Added</th>
						<th class="px-5 py-3"></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100 dark:divide-gray-800">
					{#each visibleUsers as user}
						<tr class="hover:bg-gray-50/80 dark:hover:bg-gray-900/70">
							<td class="px-5 py-4">
								<div class="flex items-center gap-3">
									{#if user.avatarUrl}
										<img src={user.avatarUrl} alt="" class="h-8 w-8 rounded-full object-cover" />
									{:else}
										<div class="flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 bg-gray-50 text-xs font-semibold text-gray-500 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300">
											{initial(user.email)}
										</div>
									{/if}
									<div>
										<p class="text-sm font-medium text-gray-950 dark:text-white">{user.githubUsername ?? 'Not logged in yet'}</p>
										<p class="text-xs text-gray-500 dark:text-gray-400">{user.email}</p>
									</div>
								</div>
							</td>
							<td class="px-5 py-4">
								<span class="inline-flex rounded-md border px-2 py-1 text-xs font-medium capitalize
									{user.role === 'owner'
										? 'border-brand-500/30 bg-brand-50 text-brand-900 dark:border-brand-500/40 dark:bg-brand-500/10 dark:text-brand-100'
										: 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300'}"
								>
									{user.role}
								</span>
							</td>
							<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
								{formatDate(user.lastLoginAt)}
							</td>
							<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
								{formatDate(user.createdAt)}
							</td>
							<td class="px-5 py-4 text-right">
								<div class="flex justify-end gap-2">
									{#if confirmRemoveUserId === user.id}
										<ActionButton variant="ghost" size="xs" on:click={() => (confirmRemoveUserId = '')}>
											Cancel
										</ActionButton>
										<ActionButton
											variant="danger"
											size="xs"
											on:click={() => handleRemove(user.id, user.email)}
											disabled={removingUserId !== '' && removingUserId !== user.id}
											loading={removingUserId === user.id}
											loadingLabel="Removing..."
										>
											Confirm
										</ActionButton>
									{:else}
										<ActionButton
											variant="ghostDanger"
											size="xs"
											on:click={() => requestRemove(user.id)}
											disabled={removingUserId !== ''}
										>
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
