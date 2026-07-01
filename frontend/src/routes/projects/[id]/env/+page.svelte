<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { EnvVar } from '$types';

	let vars: Array<EnvVar & { value: string; revealed: boolean; dirty: boolean }> = [];
	let loading = true;

	let newKey   = '';
	let newValue = '';
	let adding   = false;

	function toggleReveal(id: string) {
		vars = vars.map((v) => (v.id === id ? { ...v, revealed: !v.revealed } : v));
	}

	function markDirty(id: string, value: string) {
		vars = vars.map((v) => (v.id === id ? { ...v, value, dirty: true } : v));
	}

	function handleDelete(id: string, key: string) {
		api.env.delete($page.params.id, key)
			.then(() => {
				vars = vars.filter((v) => v.id !== id);
				toast.success(`Deleted ${key}`);
			})
			.catch((err) => toast.error(err instanceof Error ? err.message : 'Failed to delete variable'));
	}

	async function handleSave() {
		try {
			await api.env.bulkUpdate($page.params.id, {
				vars: vars.filter((v) => v.dirty).map((v) => ({ key: v.key, value: v.value }))
			});
			toast.success('Environment variables saved');
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save environment variables');
		}
	}

	async function handleAdd() {
		if (!newKey.trim()) return;
		try {
			await api.env.bulkUpdate($page.params.id, { vars: [{ key: newKey.trim(), value: newValue }] });
			newKey   = '';
			newValue = '';
			adding   = false;
			toast.success('Variable added');
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to add variable');
		}
	}

	$: hasDirty = vars.some((v) => v.dirty);

	onMount(load);

	async function load() {
		loading = true;
		try {
			const rows = await api.env.list($page.params.id);
			vars = rows.map((v) => ({ ...v, value: '', revealed: false, dirty: false }));
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Environment · MyPaas</title>
</svelte:head>

<div class="space-y-4">
	<div class="rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
		<!-- Header -->
		<div class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-gray-800">
			<div>
				<h2 class="font-semibold text-gray-900 dark:text-white">Environment variables</h2>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
					Stored encrypted (AES-256-GCM). Values are masked by default.
				</p>
			</div>
			{#if hasDirty}
				<button
					on:click={handleSave}
					class="rounded-lg bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700"
				>
					Save changes
				</button>
			{/if}
		</div>

		<!-- Variable rows -->
		{#if loading}
			<p class="p-5 text-sm text-gray-500 dark:text-gray-400">Loading environment variables...</p>
		{:else}
		<div class="divide-y divide-gray-50 dark:divide-gray-800/50">
			{#each vars as v}
				<div class="flex items-center gap-3 px-5 py-3">
					<!-- Key -->
					<span class="w-48 shrink-0 font-mono text-sm font-medium text-gray-900 dark:text-white">
						{v.key}
						{#if v.dirty}
							<span class="ml-1 text-xs text-amber-500">●</span>
						{/if}
					</span>

					<!-- Value input -->
					<div class="relative flex-1">
						<input
							type={v.revealed ? 'text' : 'password'}
							value={v.revealed ? v.value || '••••••••' : ''}
							placeholder="••••••••"
							on:input={(e) => markDirty(v.id, (e.currentTarget as HTMLInputElement).value)}
							class="w-full rounded-lg border border-gray-200 px-3 py-1.5 font-mono text-sm
								   focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
								   dark:border-gray-700 dark:bg-gray-800 dark:text-white"
						/>
					</div>

					<!-- Actions -->
					<button
						on:click={() => toggleReveal(v.id)}
						class="shrink-0 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
						aria-label="Toggle visibility"
					>
						{#if v.revealed}
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
							</svg>
						{:else}
							<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
								<path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
							</svg>
						{/if}
					</button>
					<button
						on:click={() => handleDelete(v.id, v.key)}
						class="shrink-0 text-gray-400 hover:text-red-500"
						aria-label="Delete"
					>
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
						</svg>
					</button>
				</div>
			{/each}
		</div>
		{/if}

		<!-- Add new row -->
		{#if adding}
			<div class="flex items-center gap-3 border-t border-gray-100 px-5 py-3 dark:border-gray-800">
				<input
					type="text"
					bind:value={newKey}
					placeholder="KEY"
					class="w-48 shrink-0 rounded-lg border border-gray-300 px-3 py-1.5 font-mono text-sm
						   focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
						   dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
				/>
				<input
					type="text"
					bind:value={newValue}
					placeholder="value"
					class="flex-1 rounded-lg border border-gray-300 px-3 py-1.5 font-mono text-sm
						   focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500
						   dark:border-gray-700 dark:bg-gray-800 dark:text-white dark:placeholder-gray-500"
				/>
				<button
					on:click={handleAdd}
					class="rounded-lg bg-brand-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-brand-700"
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
		{:else}
			<div class="border-t border-gray-100 px-5 py-3 dark:border-gray-800">
				<button
					on:click={() => (adding = true)}
					class="inline-flex items-center gap-1.5 text-sm text-brand-600 hover:text-brand-700 dark:text-brand-400"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
					</svg>
					Add variable
				</button>
			</div>
		{/if}
	</div>
</div>
