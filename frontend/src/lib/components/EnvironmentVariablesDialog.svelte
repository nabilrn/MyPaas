<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';
	import { ExternalLink, Plus, Save, X } from '@lucide/svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import SecretField from '$components/SecretField.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { EnvVar } from '$types';

	type EnvRow = EnvVar & { value: string; revealed: boolean; dirty: boolean; revealing: boolean };

	export let projectId = '';
	export let fullPageHref = '';

	const dispatch = createEventDispatcher<{ close: void; changed: number }>();

	let vars: EnvRow[] = [];
	let loading = true;
	let error = '';
	let newKey = '';
	let newValue = '';
	let adding = false;
	let savingChanges = false;
	let savingNewVar = false;
	let deletingKeys = new Set<string>();

	$: dirtyCount = vars.filter((row) => row.dirty).length;
	$: hasDirty = dirtyCount > 0;
	$: canAdd = Boolean(newKey.trim() && !savingNewVar);

	onMount(() => void load());

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') closeDialog();
	}

	function closeDialog() {
		if (hasDirty) {
			toast.warning('Save or discard environment drafts before closing');
			return;
		}
		dispatch('close');
	}

	async function load(background = false) {
		if (!background) loading = true;
		error = '';
		try {
			const rows = await api.env.list(projectId);
			vars = rows.map((row) => ({ ...row, value: '', revealed: false, dirty: false, revealing: false }));
			dispatch('changed', vars.length);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load environment variables';
			if (background) toast.error(error);
		} finally {
			if (!background) loading = false;
		}
	}

	async function toggleReveal(id: string) {
		const row = vars.find((item) => item.id === id);
		if (!row || row.revealing) return;
		if (row.revealed) {
			vars = vars.map((item) => (item.id === id ? { ...item, value: '', revealed: false } : item));
			return;
		}
		if (row.dirty) {
			toast.warning('Save or discard the draft before revealing the stored value');
			return;
		}

		vars = vars.map((item) => (item.id === id ? { ...item, revealing: true } : item));
		try {
			const revealed = await api.env.reveal(projectId, row.key);
			vars = vars.map((item) => (item.id === id ? { ...item, value: revealed.value, revealed: true, revealing: false } : item));
		} catch (err) {
			vars = vars.map((item) => (item.id === id ? { ...item, revealing: false } : item));
			toast.error(err instanceof Error ? err.message : 'Failed to reveal variable');
		}
	}

	function markDirty(id: string, value: string) {
		vars = vars.map((row) => (row.id === id ? { ...row, value, dirty: true } : row));
	}

	function discardDraft(id: string) {
		vars = vars.map((row) => (row.id === id ? { ...row, value: '', dirty: false, revealed: false } : row));
	}

	function copyValue(key: string, value: string) {
		if (!value) return;
		void navigator.clipboard.writeText(value)
			.then(() => toast.success(`${key} copied`))
			.catch(() => toast.error('Failed to copy variable'));
	}

	async function handleSave() {
		if (!hasDirty || savingChanges) return;
		savingChanges = true;
		try {
			await api.env.bulkUpdate(projectId, {
				vars: vars.filter((row) => row.dirty).map((row) => ({ key: row.key, value: row.value }))
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
			await api.env.bulkUpdate(projectId, { vars: [{ key: normalizeEnvKey(newKey), value: newValue }] });
			newKey = '';
			newValue = '';
			adding = false;
			toast.success('Variable added');
			await load(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to add variable');
		} finally {
			savingNewVar = false;
		}
	}

	async function handleDelete(id: string, key: string) {
		if (deletingKeys.has(key)) return;
		deletingKeys = new Set(deletingKeys).add(key);
		try {
			await api.env.delete(projectId, key);
			vars = vars.filter((row) => row.id !== id);
			dispatch('changed', vars.length);
			toast.success(`Deleted ${key}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete variable');
		} finally {
			const next = new Set(deletingKeys);
			next.delete(key);
			deletingKeys = next;
		}
	}

	function normalizeEnvKey(value: string) {
		return value.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
	}

	function envState(row: EnvRow) {
		if (row.dirty && row.revealed) return 'Unsaved visible change';
		if (row.dirty) return 'Unsaved overwrite draft';
		if (row.revealed) return 'Stored value revealed';
		return `Updated ${new Date(row.updatedAt).toLocaleDateString()}`;
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6">
	<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close environment variables" on:click={closeDialog}></button>
	<div class="overlay relative flex max-h-[min(50rem,calc(100dvh-2rem))] w-full max-w-5xl flex-col overflow-hidden" role="dialog" aria-modal="true" aria-labelledby="environment-dialog-title">
		<div class="flex items-start justify-between gap-4 border-b border-gray-100 px-4 py-3 dark:border-neutral-800 sm:px-5">
			<div class="min-w-0">
				<h2 id="environment-dialog-title" class="text-sm font-semibold text-gray-950 dark:text-white">Environment variables</h2>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Encrypted at rest. Editing is explicit so stored secrets are never mistaken for drafts.</p>
			</div>
			<div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
				{#if hasDirty}
					<span class="inline-flex items-center gap-2 text-xs text-amber-700 dark:text-amber-200">
						<span class="status-dot bg-amber-500"></span>
						{dirtyCount} unsaved
					</span>
					<ActionButton variant="primary" size="xs" on:click={handleSave} loading={savingChanges} loadingLabel="Saving">
						<Save slot="icon" class="h-3.5 w-3.5" />
						Save changes
					</ActionButton>
				{/if}
				{#if !adding}
					<ActionButton variant="secondary" size="xs" on:click={() => (adding = true)} disabled={loading || Boolean(error)}>
						<Plus slot="icon" class="h-3.5 w-3.5" />
						Add variable
					</ActionButton>
				{/if}
				<IconButton label="Close environment variables" variant="ghost" on:click={closeDialog}>
					<X class="h-4 w-4" aria-hidden="true" />
				</IconButton>
			</div>
		</div>

		<div class="min-h-0 flex-1 overflow-y-auto bg-white dark:bg-neutral-950">
			{#if loading}
				<div class="flex min-h-44 items-center justify-center p-5"><LoadingIndicator label="Loading environment variables" /></div>
			{:else if error}
				<ErrorState title="Could not load environment variables" message={error} on:retry={() => void load()} />
			{:else if vars.length === 0}
				<EmptyState title="No environment variables yet." description="Add variables when the app needs runtime configuration or secrets." compact />
			{:else}
				<div class="divide-y divide-gray-100 dark:divide-neutral-800">
					{#each vars as row (row.id)}
						<SecretField
							keyName={row.key}
							value={row.value}
							revealed={row.revealed}
							dirty={row.dirty}
							revealing={row.revealing}
							deleting={deletingKeys.has(row.key)}
							stateLabel={envState(row)}
							on:change={(event) => markDirty(row.id, event.detail)}
							on:copy={() => copyValue(row.key, row.value)}
							on:discard={() => discardDraft(row.id)}
							on:reveal={() => toggleReveal(row.id)}
							on:remove={() => handleDelete(row.id, row.key)}
						/>
					{/each}
				</div>
			{/if}

			{#if adding}
				<div class="grid gap-3 border-t border-gray-100 bg-gray-50/55 p-4 dark:border-neutral-800 dark:bg-neutral-900/40 lg:grid-cols-[14rem_minmax(0,1fr)_auto] lg:items-end">
					<div>
						<label class="field-label" for="dialog-new-env-key">Key</label>
						<input id="dialog-new-env-key" type="text" value={newKey} on:input={(event) => (newKey = normalizeEnvKey((event.currentTarget as HTMLInputElement).value))} placeholder="KEY" class="field w-full font-mono uppercase" />
					</div>
					<div>
						<label class="field-label" for="dialog-new-env-value">Value</label>
						<input id="dialog-new-env-value" type="password" bind:value={newValue} autocomplete="new-password" placeholder="value" class="field w-full font-mono" />
					</div>
					<div class="flex gap-2">
						<ActionButton variant="primary" size="sm" on:click={handleAdd} loading={savingNewVar} loadingLabel="Adding" disabled={!canAdd}>
							<Plus slot="icon" class="h-4 w-4" />
							Add
						</ActionButton>
						<ActionButton variant="ghost" size="sm" on:click={() => { adding = false; newKey = ''; newValue = ''; }} disabled={savingNewVar}>Cancel</ActionButton>
					</div>
				</div>
			{/if}
		</div>

		<div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 px-4 py-2.5 text-xs text-gray-500 dark:border-neutral-800 dark:text-gray-400 sm:px-5">
			<p>Need bulk import or overwrite review? Use the full environment workspace.</p>
			<a href={fullPageHref} class="inline-flex items-center gap-1.5 font-medium text-gray-800 hover:underline dark:text-gray-200">
				Open full workspace <ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
			</a>
		</div>
	</div>
</div>
