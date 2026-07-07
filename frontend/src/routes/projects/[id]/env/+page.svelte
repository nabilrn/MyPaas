<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import SecretField from '$components/SecretField.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import type { EnvVar } from '$types';

	type EnvRow = EnvVar & { value: string; revealed: boolean; dirty: boolean; revealing: boolean };

	let vars: EnvRow[] = [];
	let loading = true;
	let error = '';

	let newKey   = '';
	let newValue = '';
	let adding   = false;
	let savingChanges = false;
	let savingNewVar = false;
	let deletingKeys = new Set<string>();

	$: dirtyCount = vars.filter((v) => v.dirty).length;
	$: hasDirty = dirtyCount > 0;
	$: canAdd = Boolean(newKey.trim() && !savingNewVar);

	async function toggleReveal(id: string) {
		const row = vars.find((v) => v.id === id);
		if (!row || row.revealing) return;
		if (row.revealed) {
			vars = vars.map((v) => (v.id === id ? { ...v, revealed: false } : v));
			return;
		}
		if (row.dirty) {
			toast.warning('Save or discard the draft before revealing the stored value');
			return;
		}

		vars = vars.map((v) => (v.id === id ? { ...v, revealing: true } : v));
		try {
			const revealed = await api.env.reveal($page.params.id, row.key);
			vars = vars.map((v) => (v.id === id ? { ...v, value: revealed.value, revealed: true, revealing: false } : v));
		} catch (err) {
			vars = vars.map((v) => (v.id === id ? { ...v, revealing: false } : v));
			toast.error(err instanceof Error ? err.message : 'Failed to reveal variable');
		}
	}

	function markDirty(id: string, value: string) {
		vars = vars.map((v) => (v.id === id ? { ...v, value, dirty: true } : v));
	}

	function discardDraft(id: string) {
		vars = vars.map((v) => (v.id === id ? { ...v, value: '', dirty: false, revealed: false } : v));
	}

	function copyValue(key: string, value: string) {
		if (!value) return;
		void navigator.clipboard.writeText(value)
			.then(() => toast.success(`${key} copied`))
			.catch(() => toast.error('Failed to copy variable'));
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
			await api.env.bulkUpdate($page.params.id, { vars: [{ key: normalizeEnvKey(newKey), value: newValue }] });
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
		error = '';
		try {
			const rows = await api.env.list($page.params.id);
			vars = rows.map((v) => ({ ...v, value: '', revealed: false, dirty: false, revealing: false }));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load environment variables';
			if (background) {
				toast.error(error);
			}
		} finally {
			if (!background) {
				loading = false;
			}
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

<svelte:head>
	<title>Environment · MyPaas</title>
</svelte:head>

<SectionPanel
	title="Environment variables"
	description="Encrypted at rest. Reveal only when you need to inspect a stored value."
	contentClass="p-0"
>
	<svelte:fragment slot="actions">
		<div class="flex flex-wrap items-center gap-2">
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
				<IconButton label="Add variable" variant="primary" on:click={() => (adding = true)}>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
					</svg>
				</IconButton>
			{/if}
		</div>
	</svelte:fragment>

	{#if loading}
		<div class="space-y-3 p-5">
			{#each [1, 2, 3] as _}
				<div class="h-11 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>
			{/each}
		</div>
	{:else if error}
		<ErrorState title="Could not load environment variables" message={error} on:retry={() => void load()} />
	{:else if vars.length === 0}
		<EmptyState
			title="No environment variables yet."
			description="Add variables when the app needs runtime configuration or secrets."
			compact
		/>
	{:else}
		<div class="divide-y divide-gray-100 dark:divide-gray-800">
			{#each vars as v}
				<SecretField
					keyName={v.key}
					value={v.value}
					revealed={v.revealed}
					dirty={v.dirty}
					revealing={v.revealing}
					deleting={deletingKeys.has(v.key)}
					stateLabel={envState(v)}
					on:change={(event) => markDirty(v.id, event.detail)}
					on:copy={() => copyValue(v.key, v.value)}
					on:discard={() => discardDraft(v.id)}
					on:reveal={() => toggleReveal(v.id)}
					on:remove={() => handleDelete(v.id, v.key)}
				/>
			{/each}
		</div>
	{/if}

	{#if adding}
		<div class="grid gap-3 border-t border-gray-100 bg-gray-50/70 px-5 py-4 dark:border-gray-800 dark:bg-gray-900/60 lg:grid-cols-[14rem_minmax(0,1fr)_auto]">
			<input
				type="text"
				value={newKey}
				on:input={(event) => (newKey = normalizeEnvKey((event.currentTarget as HTMLInputElement).value))}
				placeholder="KEY"
				class="field w-full font-mono uppercase"
			/>
			<input type="text" bind:value={newValue} placeholder="value" class="field w-full font-mono" />
			<div class="flex gap-2">
				<ActionButton
					variant="primary"
					on:click={handleAdd}
					loading={savingNewVar}
					loadingLabel="Adding..."
					disabled={!canAdd}
				>
					Add
				</ActionButton>
				<ActionButton variant="ghost" on:click={() => (adding = false)} disabled={savingNewVar}>
					Cancel
				</ActionButton>
			</div>
		</div>
	{/if}
</SectionPanel>
