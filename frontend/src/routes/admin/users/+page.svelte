<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { User } from '$types';

	let users: User[] = [];
	let loading = true;

	let addEmail = '';
	let addRole: 'owner' | 'collaborator' = 'collaborator';
	let adding = false;

	async function handleAdd() {
		if (!addEmail.trim()) return;
		try {
			await api.admin.addUser({ email: addEmail.trim(), role: addRole });
			toast.success('User added');
			adding = false;
			addEmail = '';
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to add user');
		}
	}

	async function handleRemove(id: string, email: string) {
		try {
			await api.admin.removeUser(id);
			users = users.filter((u) => u.id !== id);
			toast.success(`Removed ${email}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove user');
		}
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

<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6">
	<div class="mb-6 flex items-center justify-between">
		<div>
			<h1 class="text-xl font-bold text-gray-900 dark:text-white">User whitelist</h1>
			<p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
				Only listed users can sign in via GitHub OAuth.
			</p>
		</div>
		<button
			on:click={() => (adding = true)}
			class="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
				<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
			</svg>
			Add user
		</button>
	</div>

	<!-- Add user form -->
	{#if adding}
		<div class="mb-4 rounded-xl border border-brand-200 bg-brand-50 p-4 dark:border-brand-900/50 dark:bg-brand-900/10">
			<h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">Add user to whitelist</h3>
			<div class="flex gap-3">
				<input
					type="email"
					bind:value={addEmail}
					placeholder="user@example.com"
					class="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
						   dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
				/>
				<select
					bind:value={addRole}
					class="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800 dark:text-white"
				>
					<option value="collaborator">Collaborator</option>
					<option value="owner">Owner</option>
				</select>
				<button
					on:click={handleAdd}
					class="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
				>
					Add
				</button>
				<button
					on:click={() => (adding = false)}
					class="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
				>
					Cancel
				</button>
			</div>
		</div>
	{/if}

	<!-- Users table -->
	<div class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
		{#if loading}
			<p class="p-5 text-sm text-gray-500 dark:text-gray-400">Loading users...</p>
		{:else}
		<table class="min-w-full divide-y divide-gray-100 dark:divide-gray-800">
			<thead>
				<tr class="bg-gray-50 dark:bg-gray-800/50">
					<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">User</th>
					<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Role</th>
					<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Last login</th>
					<th class="px-5 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">Added</th>
					<th class="px-5 py-3"></th>
				</tr>
			</thead>
			<tbody class="divide-y divide-gray-50 dark:divide-gray-800">
				{#each users as user}
					<tr class="hover:bg-gray-50 dark:hover:bg-gray-800/30">
						<td class="px-5 py-4">
							<div class="flex items-center gap-3">
								<img src={user.avatarUrl} alt="" class="h-8 w-8 rounded-full" />
								<div>
									<p class="text-sm font-medium text-gray-900 dark:text-white">{user.githubUsername ?? 'Not logged in yet'}</p>
									<p class="text-xs text-gray-500 dark:text-gray-400">{user.email}</p>
								</div>
							</div>
						</td>
						<td class="px-5 py-4">
							<span class="inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium
								{user.role === 'owner'
									? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
									: 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'}">
								{user.role}
							</span>
						</td>
						<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
							{user.lastLoginAt ? new Date(user.lastLoginAt).toLocaleDateString() : '—'}
						</td>
						<td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">
							{new Date(user.createdAt).toLocaleDateString()}
						</td>
						<td class="px-5 py-4 text-right">
							<button
								on:click={() => handleRemove(user.id, user.email)}
								class="text-sm text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
							>
								Remove
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
		{/if}
	</div>
</div>
