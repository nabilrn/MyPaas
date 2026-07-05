<script lang="ts">
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { User } from '$types';

	let users: User[] = [];
	let loading = true;

	let addEmail = '';
	let addRole: 'owner' | 'collaborator' = 'collaborator';
	let adding = false;
	let savingUser = false;
	let removingUserId = '';

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

	async function handleRemove(id: string, email: string) {
		if (removingUserId || !window.confirm(`Remove ${email} from the whitelist?`)) return;
		removingUserId = id;
		try {
			await api.admin.removeUser(id);
			users = users.filter((u) => u.id !== id);
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

	onMount(load);

	async function load() {
		loading = true;
		try {
			users = await api.admin.listUsers();
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Users · MyPaas Admin</title>
</svelte:head>

<div class="mx-auto max-w-7xl px-4 py-7 sm:px-6">
	<header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<p class="text-xs font-medium uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">Admin</p>
			<h1 class="mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">User whitelist</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Only listed users can sign in via GitHub OAuth.</p>
		</div>
		<ActionButton variant="primary" size="md" on:click={() => (adding = true)}>
			Add user
		</ActionButton>
	</header>

	{#if adding}
		<section class="surface mb-4 p-4">
			<h2 class="mb-3 text-sm font-semibold text-gray-950 dark:text-white">Add user to whitelist</h2>
			<div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_12rem_auto_auto] md:items-center">
				<input type="email" bind:value={addEmail} placeholder="user@example.com" class="field w-full" />
				<select bind:value={addRole} class="field w-full">
					<option value="collaborator">Collaborator</option>
					<option value="owner">Owner</option>
				</select>
				<ActionButton variant="primary" on:click={handleAdd} loading={savingUser} loadingLabel="Adding...">
					Add
				</ActionButton>
				<ActionButton variant="ghost" on:click={() => (adding = false)}>
					Cancel
				</ActionButton>
			</div>
		</section>
	{/if}

	<section class="surface overflow-hidden">
		{#if loading}
			<div class="space-y-3 p-5">
				{#each [1, 2, 3] as _}
					<div class="h-12 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
				{/each}
			</div>
		{:else}
			<table class="min-w-full divide-y divide-gray-100 dark:divide-gray-800">
				<thead>
					<tr class="bg-gray-50/70 dark:bg-gray-900/70">
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">User</th>
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Role</th>
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Last login</th>
						<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Added</th>
						<th class="px-5 py-3"></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-100 dark:divide-gray-800">
					{#each users as user}
						<tr class="hover:bg-gray-50/80 dark:hover:bg-gray-900/70">
							<td class="px-5 py-4">
								<div class="flex items-center gap-3">
									{#if user.avatarUrl}
										<img src={user.avatarUrl} alt="" class="h-8 w-8 rounded-full" />
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
								<span class="inline-flex rounded-md border border-gray-200 px-2 py-1 text-xs font-medium capitalize text-gray-600 dark:border-gray-800 dark:text-gray-300">
									{user.role}
								</span>
							</td>
							<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
								{user.lastLoginAt ? new Date(user.lastLoginAt).toLocaleDateString() : '-'}
							</td>
							<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
								{new Date(user.createdAt).toLocaleDateString()}
							</td>
							<td class="px-5 py-4 text-right">
								<ActionButton
									variant="ghostDanger"
									size="xs"
									on:click={() => handleRemove(user.id, user.email)}
									disabled={removingUserId !== '' && removingUserId !== user.id}
									loading={removingUserId === user.id}
									loadingLabel="Removing..."
								>
									Remove
								</ActionButton>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>
</div>
