<script lang="ts">
	import { Plus, Save, Upload, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import SecretField from '$components/SecretField.svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import { parseEnvContent, type ParsedEnvEntry } from '$lib/utils/envParser';
	import type { EnvVar } from '$types';

	export let projectId: string;

	type EnvRow = EnvVar & { value: string; revealed: boolean; dirty: boolean; revealing: boolean };
	type ImportStatus = 'new' | 'overwrite' | 'duplicate' | 'invalid';
	type ImportRow = ParsedEnvEntry & { importStatus: ImportStatus };
	type ImportCounts = { total: number; newCount: number; overwrite: number; duplicate: number; invalid: number };

	const MAX_ENV_IMPORT_BYTES = 128 * 1024;

	let vars: EnvRow[] = [];
	let loading = true;
	let error = '';
	let loadedProjectId = '';
	let newKey = '';
	let newValue = '';
	let adding = false;
	let savingChanges = false;
	let savingNewVar = false;
	let deletingKeys = new Set<string>();
	let importing = false;
	let importText = '';
	let importFileName = '';
	let importFileInput: HTMLInputElement | null = null;
	let confirmOverwrite = false;
	let savingImport = false;

	$: dirtyCount = vars.filter((v) => v.dirty).length;
	$: hasDirty = dirtyCount > 0;
	$: canAdd = Boolean(newKey.trim() && !savingNewVar);
	$: existingKeys = new Set(vars.map((v) => v.key));
	$: importRows = buildImportRows(importText, existingKeys);
	$: importCounts = countImportRows(importRows);
	$: importReadyRows = importRows.filter((row) => row.importStatus === 'new' || (row.importStatus === 'overwrite' && confirmOverwrite));
	$: canSaveImport = importReadyRows.length > 0 && !savingImport && !hasDirty && importCounts.invalid === 0;
	$: if (loadedProjectId && projectId && projectId !== loadedProjectId) void load();

	onMount(load);

	async function load(background = false) {
		if (!projectId) return;
		loadedProjectId = projectId;
		if (!background) loading = true;
		error = '';
		try {
			const rows = await api.env.list(projectId);
			vars = rows.map((v) => ({ ...v, value: '', revealed: false, dirty: false, revealing: false }));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load environment variables';
			if (background) toast.error(error);
		} finally {
			if (!background) loading = false;
		}
	}

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
			const revealed = await api.env.reveal(projectId, row.key);
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
			await api.env.delete(projectId, key);
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
			await api.env.bulkUpdate(projectId, { vars: vars.filter((v) => v.dirty).map((v) => ({ key: v.key, value: v.value })) });
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

	function openImport() {
		if (hasDirty) {
			toast.warning('Save or discard current changes before importing');
			return;
		}
		importing = true;
	}

	function closeImport() {
		if (savingImport) return;
		importing = false;
		importText = '';
		importFileName = '';
		confirmOverwrite = false;
		if (importFileInput) importFileInput.value = '';
	}

	async function handleImportFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		if (file.size > MAX_ENV_IMPORT_BYTES) {
			toast.error('Env file is too large');
			input.value = '';
			return;
		}
		try {
			importText = await file.text();
			importFileName = file.name;
			importing = true;
		} catch {
			toast.error('Failed to read env file');
		}
	}

	async function handleImportSave() {
		if (!canSaveImport) return;
		savingImport = true;
		try {
			await api.env.bulkUpdate(projectId, { vars: importReadyRows.map((row) => ({ key: row.key, value: row.value })) });
			toast.success(`Imported ${importReadyRows.length} environment variables`);
			closeImport();
			await load(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to import environment variables');
		} finally {
			savingImport = false;
		}
	}

	function normalizeEnvKey(value: string) {
		return value.trim().toUpperCase().replace(/[^A-Z0-9_]/g, '_');
	}

	function envState(row: EnvRow) {
		if (row.dirty) return 'Unsaved change';
		if (row.revealed) return 'Value visible';
		return '';
	}

	function buildImportRows(content: string, keys: Set<string>): ImportRow[] {
		if (!content.trim()) return [];
		return parseEnvContent(content).map((entry) => ({
			...entry,
			importStatus: entry.status === 'invalid'
				? 'invalid'
				: entry.status === 'duplicate'
					? 'duplicate'
					: keys.has(entry.key)
						? 'overwrite'
						: 'new'
		}));
	}

	function countImportRows(rows: ImportRow[]): ImportCounts {
		return rows.reduce<ImportCounts>((counts, row) => {
			counts.total += 1;
			if (row.importStatus === 'new') counts.newCount += 1;
			if (row.importStatus === 'overwrite') counts.overwrite += 1;
			if (row.importStatus === 'duplicate') counts.duplicate += 1;
			if (row.importStatus === 'invalid') counts.invalid += 1;
			return counts;
		}, { total: 0, newCount: 0, overwrite: 0, duplicate: 0, invalid: 0 });
	}
</script>

<div class="border-t border-gray-100 dark:border-neutral-800">
	<div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 py-3 dark:border-neutral-800">
		<p class="text-sm text-gray-500 dark:text-gray-400">{vars.length} variable{vars.length === 1 ? '' : 's'}</p>
		<div class="flex flex-wrap items-center gap-2">
			{#if hasDirty}
				<ActionButton variant="primary" on:click={handleSave} loading={savingChanges} loadingLabel="Saving"><Save slot="icon" class="h-4 w-4" />Save {dirtyCount} change{dirtyCount === 1 ? '' : 's'}</ActionButton>
			{/if}
			<ActionButton variant="secondary" on:click={openImport} disabled={loading || Boolean(error)}><Upload slot="icon" class="h-4 w-4" />Import .env</ActionButton>
			<ActionButton variant="primary" on:click={() => (adding = true)} disabled={loading || Boolean(error)}><Plus slot="icon" class="h-4 w-4" />Add variable</ActionButton>
		</div>
	</div>

	{#if loading}
		<div class="flex min-h-40 items-center justify-center py-6"><LoadingIndicator label="Loading environment variables" /></div>
	{:else if error}
		<ErrorState title="Could not load environment variables" message={error} on:retry={() => void load()} />
	{:else if vars.length === 0}
		<EmptyState title="No environment variables yet." description="Add variables when your app needs runtime configuration or secrets." compact />
	{:else}
		<div class="divide-y divide-gray-100 dark:divide-neutral-800">
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
</div>

{#if adding}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close add variable" on:click={() => !savingNewVar && (adding = false)}></button>
		<div class="overlay relative w-full max-w-lg" role="dialog" aria-modal="true" aria-labelledby="add-env-title">
			<div class="panel-header flex items-start justify-between gap-3">
				<div><h2 id="add-env-title" class="panel-title">Add environment variable</h2><p class="panel-description">Add a key and value for this project.</p></div>
				<ActionButton variant="ghost" size="xs" on:click={() => (adding = false)} disabled={savingNewVar}><X slot="icon" class="h-4 w-4" />Close</ActionButton>
			</div>
			<div class="space-y-4 p-4">
				<div><label class="field-label" for="new-env-key">Key</label><input id="new-env-key" type="text" value={newKey} on:input={(event) => (newKey = normalizeEnvKey((event.currentTarget as HTMLInputElement).value))} placeholder="API_KEY" class="field w-full font-mono uppercase" /></div>
				<div><label class="field-label" for="new-env-value">Value</label><input id="new-env-value" type="text" bind:value={newValue} placeholder="value" class="field w-full font-mono" /></div>
				<div class="flex justify-end gap-2"><ActionButton variant="ghost" on:click={() => (adding = false)} disabled={savingNewVar}>Cancel</ActionButton><ActionButton variant="primary" on:click={handleAdd} loading={savingNewVar} loadingLabel="Adding" disabled={!canAdd}>Add variable</ActionButton></div>
			</div>
		</div>
	</div>
{/if}

{#if importing}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close env import" on:click={closeImport}></button>
		<div class="overlay relative max-h-[90vh] w-full max-w-2xl overflow-hidden" role="dialog" aria-modal="true" aria-labelledby="import-env-title">
			<div class="panel-header flex items-start justify-between gap-3">
				<div><h2 id="import-env-title" class="panel-title">Import .env</h2><p class="panel-description">Paste env content or choose a local file.</p></div>
				<ActionButton variant="ghost" size="xs" on:click={closeImport} disabled={savingImport}><X slot="icon" class="h-4 w-4" />Close</ActionButton>
			</div>
			<div class="max-h-[calc(90vh-5rem)] space-y-4 overflow-y-auto p-4">
				<input bind:this={importFileInput} type="file" accept=".env,.env.example,.env.sample,.env.template,.txt,text/plain" class="sr-only" on:change={handleImportFile} />
				<div class="flex flex-wrap items-center justify-between gap-3">
					<p class="text-sm text-gray-500 dark:text-gray-400">{importFileName || 'No file selected'}</p>
					<ActionButton variant="secondary" on:click={() => importFileInput?.click()} disabled={savingImport}><Upload slot="icon" class="h-4 w-4" />Choose file</ActionButton>
				</div>
				<textarea bind:value={importText} rows="9" placeholder={'KEY=value\nSECRET="quoted value"'} class="field min-h-48 w-full resize-y font-mono text-sm leading-5" disabled={savingImport}></textarea>

				{#if importRows.length > 0}
					<div class="grid grid-cols-2 gap-x-6 gap-y-2 border-y border-gray-100 py-3 text-sm dark:border-neutral-800 sm:grid-cols-4">
						<div><span class="text-gray-500 dark:text-gray-400">New</span><span class="ml-2 font-medium text-gray-950 dark:text-white">{importCounts.newCount}</span></div>
						<div><span class="text-gray-500 dark:text-gray-400">Overwrite</span><span class="ml-2 font-medium text-gray-950 dark:text-white">{importCounts.overwrite}</span></div>
						<div><span class="text-gray-500 dark:text-gray-400">Duplicate</span><span class="ml-2 font-medium text-gray-950 dark:text-white">{importCounts.duplicate}</span></div>
						<div><span class="text-gray-500 dark:text-gray-400">Invalid</span><span class="ml-2 font-medium {importCounts.invalid ? 'text-red-600 dark:text-red-300' : 'text-gray-950 dark:text-white'}">{importCounts.invalid}</span></div>
					</div>
				{/if}

				{#if importCounts.overwrite > 0}
					<label class="flex items-start gap-2 text-sm text-gray-700 dark:text-gray-300"><input type="checkbox" bind:checked={confirmOverwrite} class="mt-0.5 h-4 w-4 rounded border-gray-300" disabled={savingImport} /><span>Overwrite {importCounts.overwrite} existing variable{importCounts.overwrite === 1 ? '' : 's'}</span></label>
				{/if}
				{#if importCounts.invalid > 0}<div class="alert-danger">Fix invalid lines before importing.</div>{/if}

				<div class="flex justify-end gap-2"><ActionButton variant="ghost" on:click={closeImport} disabled={savingImport}>Cancel</ActionButton><ActionButton variant="primary" on:click={handleImportSave} loading={savingImport} loadingLabel="Importing" disabled={!canSaveImport}>Import {importReadyRows.length} variable{importReadyRows.length === 1 ? '' : 's'}</ActionButton></div>
			</div>
		</div>
	</div>
{/if}
