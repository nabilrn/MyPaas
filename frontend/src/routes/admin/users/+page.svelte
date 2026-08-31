<script lang="ts">
	import { Plus, RefreshCw } from '@lucide/svelte';
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
	let adding = false;
	let savingUser = false;

	$: pageStart = currentPage * pageSize;
	$: visibleUsers = users.slice(pageStart, pageStart + pageSize);
	$: hasNext = pageStart + pageSize < users.length;
	$: canAdd = Boolean(addEmail.trim() && !savingUser);
	$: addDisabledReason = addEmail.trim() ? '' : 'Email is required before adding an owner.';

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

<div class="page-shell py-6">
	{#if adding}
		<SectionPanel title="Add owner" description="Whitelist an owner for GitHub OAuth and control-plane access.">
			<form class="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-start" on:submit|preventDefault={handleAdd}>
				<div>
					<label class="field-label" for="user-email">Email</label>
					<input id="user-email" type="email" bind:value={addEmail} placeholder="user@example.com" class="field w-full" />
					{#if addDisabledReason}<p class="field-hint">{addDisabledReason}</p>{/if}
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
		title="Owners"
		description="Whitelisted access to this MyPaaS control plane."
		{loading}
		loadingRows={3}
		{error}
		empty={users.length === 0}
		emptyTitle="No owners are whitelisted yet."
		emptyDescription="Add an owner to allow GitHub OAuth sign-in."
		on:retry={load}
	>
		<svelte:fragment slot="actions">
			<div class="flex items-center gap-2">
				<ActionButton variant="secondary" size="xs" loading={loading} loadingLabel="Refreshing" on:click={load}>
					<RefreshCw slot="icon" class="h-3.5 w-3.5" />
					Refresh
				</ActionButton>
				<ActionButton variant="primary" size="xs" disabled={adding} on:click={() => (adding = true)}>
					<Plus slot="icon" class="h-3.5 w-3.5" />
					Add owner
				</ActionButton>
			</div>
		</svelte:fragment>

		<table class="data-table table-fixed min-w-[48rem]">
			<colgroup>
				<col class="w-[42%]" />
				<col class="w-[14%]" />
				<col class="w-[14%]" />
				<col class="w-[14%]" />
				<col class="w-[16%]" />
			</colgroup>
			<thead>
				<tr>
					<th>User</th>
					<th>Role</th>
					<th>Last login</th>
					<th>Added</th>
					<th>Access</th>
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
						<td class="whitespace-nowrap text-center text-xs text-gray-500 dark:text-gray-400">Protected</td>
					</tr>
				{/each}
			</tbody>
		</table>
		<svelte:fragment slot="footer">
			<Pagination bind:page={currentPage} {pageSize} totalShown={visibleUsers.length} {hasNext} {loading} label="Owners" />
		</svelte:fragment>
	</TableShell>
</div>
