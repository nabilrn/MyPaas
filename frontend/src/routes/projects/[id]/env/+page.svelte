<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { EnvVar } from '$types';

	let vars: Array<EnvVar & { value: string; revealed: boolean; dirty: boolean }> = [];
	let loading = true;

	let newKey   = '';
	let newValue = '';
	let adding   = false;
	let savingChanges = false;
	let savingNewVar = false;
	let deletingKeys = new Set<string>();

	$: dirtyCount = vars.filter((v) => v.dirty).length;
	$: hasDirty = dirtyCount > 0;

	function toggleReveal(id: string) {
		vars = vars.map((v) => (v.id === id ? { ...v, revealed: !v.revealed } : v));
	}

	function markDirty(id: string, value: string) {
		vars = vars.map((v) => (v.id === id ? { ...v, value, dirty: true } : v));
	}

	async function handleDelete(id: string, key: string) {
		if (deletingKeys.has(key)) return;
		deletingKeys = new Set(deletingKeys).add(key);
		try {
			await api.env.delete($page.params.id, key);
			vars = vars.filter((v) => v.id !== id);
			toast.success(`Deleted ${key}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete variable');
		} finally {
			const next = new Set(deletingKeys);
			next.delete(key);
			deletingKeys = next;
		}
	}

	async function handleSave() {
		if (savingChanges) return;
		savingChanges = true;
		try {
			await api.env.bulkUpdate($page.params.id, {
				vars: vars.filter((v) => v.dirty).map((v) => ({ key: v.key, value: v.value }))
			});
			toast.success('Environment variables saved');
			await load(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save environment variables');
		} finally {
			savingChanges = false;
		}
	}

	async function handleAdd() {
		if (!newKey.trim() || savingNewVar) return;
		savingNewVar = true;
		try {
			await api.env.bulkUpdate($page.params.id, { vars: [{ key: newKey.trim(), value: newValue }] });
			newKey   = '';
			newValue = '';
			adding   = false;
			toast.success('Variable added');
			await load(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to add variable');
		} finally {
			savingNewVar = false;
		}
	}

	onMount(load);

	async function load(background = false) {
		if (!background) {
			loading = true;
		}
		try {
			const rows = await api.env.list($page.params.id);
			vars = rows.map((v) => ({ ...v, value: '', revealed: false, dirty: false }));
		} finally {
			if (!background) {
				loading = false;
			}
		}
	}
</script>

<svelte:head>
	<title>Environment · MyPaas</title>
</svelte:head>

<section class="surface overflow-hidden">
	<div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-gray-800 sm:flex-row sm:items-center sm:justify-between">
		<div>
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Environment variables</h2>
			<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
				Encrypted at rest. Values stay masked until edited.
			</p>
		</div>
		<div class="flex items-center gap-2">
			{#if hasDirty}
				<span class="rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
					{dirtyCount} unsaved
				</span>
				<ActionButton
					variant="primary"
					on:click={handleSave}
					loading={savingChanges}
					loadingLabel="Saving..."
				>
					Save
				</ActionButton>
			{/if}
			{#if !adding}
				<ActionButton variant="secondary" on:click={() => (adding = true)}>
					Add variable
				</ActionButton>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="space-y-3 p-5">
			{#each [1, 2, 3] as _}
				<div class="h-11 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
			{/each}
		</div>
	{:else}
		<div class="divide-y divide-gray-100 dark:divide-gray-800">
			{#each vars as v}
				<div class="grid gap-3 px-5 py-3 lg:grid-cols-[14rem_minmax(0,1fr)_6rem] lg:items-center">
					<div class="min-w-0">
						<p class="truncate font-mono text-sm font-semibold text-gray-950 dark:text-white">
							{v.key}
							{#if v.dirty}
								<span class="ml-1 text-amber-500">●</span>
							{/if}
						</p>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Updated {new Date(v.updatedAt).toLocaleDateString()}</p>
					</div>
					<input
						type={v.revealed ? 'text' : 'password'}
						value={v.revealed ? v.value || '••••••••' : ''}
						placeholder="••••••••"
						on:input={(e) => markDirty(v.id, (e.currentTarget as HTMLInputElement).value)}
						class="field w-full font-mono"
					/>
					<div class="flex items-center gap-1 lg:justify-end">
						<button
							on:click={() => toggleReveal(v.id)}
							class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white"
							aria-label="Toggle visibility"
						>
							{#if v.revealed}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M3 3l18 18M10.6 10.6A2 2 0 0013.4 13.4M9.9 4.2A10.8 10.8 0 0112 4c4.5 0 8.3 2.9 9.5 7a10.9 10.9 0 01-3 4.7M6.1 6.1A10.8 10.8 0 002.5 11c1.2 4.1 5 7 9.5 7 1.3 0 2.5-.2 3.6-.7" />
								</svg>
							{:else}
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
								</svg>
							{/if}
						</button>
						<ActionButton
							variant="ghostDanger"
							size="xs"
							on:click={() => handleDelete(v.id, v.key)}
							className="px-2"
							loading={deletingKeys.has(v.key)}
							ariaLabel={`Delete ${v.key}`}
						>
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
							</svg>
						</ActionButton>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	{#if adding}
		<div class="grid gap-3 border-t border-gray-100 bg-gray-50/70 px-5 py-4 dark:border-gray-800 dark:bg-gray-900/60 lg:grid-cols-[14rem_minmax(0,1fr)_auto]">
			<input type="text" bind:value={newKey} placeholder="KEY" class="field w-full font-mono" />
			<input type="text" bind:value={newValue} placeholder="value" class="field w-full font-mono" />
			<div class="flex gap-2">
				<ActionButton
					variant="primary"
					on:click={handleAdd}
					loading={savingNewVar}
					loadingLabel="Adding..."
				>
					Add
				</ActionButton>
				<ActionButton variant="ghost" on:click={() => (adding = false)}>
					Cancel
				</ActionButton>
			</div>
		</div>
	{/if}
</section>
