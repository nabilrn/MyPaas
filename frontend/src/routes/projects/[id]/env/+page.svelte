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
	import { parseEnvContent, type ParsedEnvEntry } from '$lib/utils/envParser';
	import type { EnvVar } from '$types';

	type EnvRow = EnvVar & { value: string; revealed: boolean; dirty: boolean; revealing: boolean };
	type ImportStatus = 'new' | 'overwrite' | 'duplicate' | 'invalid';
	type ImportRow = ParsedEnvEntry & { importStatus: ImportStatus };
	type ImportCounts = {
		total: number;
		newCount: number;
		overwrite: number;
		duplicate: number;
		invalid: number;
	};

	const MAX_ENV_IMPORT_BYTES = 128 * 1024;

	let vars: EnvRow[] = [];
	let loading = true;
	let error = '';

	let newKey   = '';
	let newValue = '';
	let adding   = false;
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
	$: canSaveImport = importReadyRows.length > 0 && !savingImport && !hasDirty;

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

	function openImport() {
		importing = true;
		adding = false;
	}

	function clearImport() {
		importText = '';
		importFileName = '';
		confirmOverwrite = false;
		if (importFileInput) {
			importFileInput.value = '';
		}
	}

	function closeImport() {
		if (savingImport) return;
		importing = false;
		clearImport();
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
			input.value = '';
		}
	}

	async function handleImportSave() {
		if (!importReadyRows.length || savingImport) return;
		if (hasDirty) {
			toast.warning('Save or discard existing drafts before importing');
			return;
		}

		savingImport = true;
		try {
			await api.env.bulkUpdate($page.params.id, {
				vars: importReadyRows.map((row) => ({ key: row.key, value: row.value }))
			});
			toast.success(`Imported ${importReadyRows.length} environment variables`);
			clearImport();
			importing = false;
			await load(true);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to import environment variables');
		} finally {
			savingImport = false;
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
		return rows.reduce<ImportCounts>(
			(counts, row) => {
				counts.total += 1;
				if (row.importStatus === 'new') counts.newCount += 1;
				if (row.importStatus === 'overwrite') counts.overwrite += 1;
				if (row.importStatus === 'duplicate') counts.duplicate += 1;
				if (row.importStatus === 'invalid') counts.invalid += 1;
				return counts;
			},
			{ total: 0, newCount: 0, overwrite: 0, duplicate: 0, invalid: 0 }
		);
	}

	function importStatusLabel(row: ImportRow) {
		if (row.importStatus === 'invalid') return row.error;
		if (row.importStatus === 'duplicate') return 'Duplicate';
		if (row.importStatus === 'overwrite') return confirmOverwrite ? 'Overwrite' : 'Confirm';
		return 'New';
	}

	function importStatusClass(status: ImportStatus) {
		return {
			new: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-200',
			overwrite: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200',
			duplicate: 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300',
			invalid: 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200'
		}[status];
	}

	function importedValueLabel(row: ImportRow) {
		if (row.importStatus === 'invalid') return 'Not saved';
		return row.value === '' ? 'Empty value' : 'Value parsed';
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
			{#if !importing}
				<ActionButton variant="secondary" on:click={openImport} disabled={loading || Boolean(error)}>
					Import .env
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

	{#if importing && !loading && !error}
		<div class="border-b border-gray-100 bg-gray-50/70 p-5 dark:border-gray-800 dark:bg-gray-900/60">
			<div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_20rem]">
				<div class="min-w-0 space-y-3">
					<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
						<div class="min-w-0">
							<h3 class="text-sm font-semibold text-gray-950 dark:text-white">Import .env</h3>
							{#if importFileName}
								<p class="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">{importFileName}</p>
							{/if}
						</div>
						<div class="flex flex-wrap gap-2">
							<input
								bind:this={importFileInput}
								type="file"
								accept=".env,.env.example,.env.sample,.env.template,.txt,text/plain"
								class="sr-only"
								on:change={handleImportFile}
							/>
							<ActionButton variant="secondary" on:click={() => importFileInput?.click()} disabled={savingImport}>
								Upload file
							</ActionButton>
							<ActionButton variant="ghost" on:click={clearImport} disabled={savingImport || !importText}>
								Clear
							</ActionButton>
							<ActionButton variant="ghost" on:click={closeImport} disabled={savingImport}>
								Close
							</ActionButton>
						</div>
					</div>

					<textarea
						bind:value={importText}
						rows="8"
						placeholder={'KEY=value\nSECRET="quoted value"\nEMPTY='}
						class="field min-h-44 w-full resize-y font-mono text-xs leading-5"
						disabled={savingImport}
					></textarea>
				</div>

				<div class="soft-panel flex min-w-0 flex-col gap-4 p-4">
					<div class="grid grid-cols-2 gap-2 text-xs">
						<div class="rounded-md border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950/70">
							<div class="metric-label">New</div>
							<div class="mt-1 text-lg font-semibold text-gray-950 dark:text-white">{importCounts.newCount}</div>
						</div>
						<div class="rounded-md border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950/70">
							<div class="metric-label">Overwrite</div>
							<div class="mt-1 text-lg font-semibold text-gray-950 dark:text-white">{importCounts.overwrite}</div>
						</div>
						<div class="rounded-md border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950/70">
							<div class="metric-label">Duplicate</div>
							<div class="mt-1 text-lg font-semibold text-gray-950 dark:text-white">{importCounts.duplicate}</div>
						</div>
						<div class="rounded-md border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950/70">
							<div class="metric-label">Invalid</div>
							<div class="mt-1 text-lg font-semibold text-gray-950 dark:text-white">{importCounts.invalid}</div>
						</div>
					</div>

					{#if importCounts.overwrite > 0}
						<label class="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
							<input
								type="checkbox"
								bind:checked={confirmOverwrite}
								class="mt-0.5 h-4 w-4 rounded border-amber-300 text-brand-700 focus:ring-brand-600"
								disabled={savingImport}
							/>
							<span>Allow {importCounts.overwrite} existing variable{importCounts.overwrite === 1 ? '' : 's'} to be overwritten</span>
						</label>
					{/if}

					{#if hasDirty}
						<p class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
							Save or discard existing drafts before importing.
						</p>
					{/if}

					<ActionButton
						variant="primary"
						on:click={handleImportSave}
						loading={savingImport}
						loadingLabel="Saving..."
						disabled={!canSaveImport}
						full
					>
						Save {importReadyRows.length} variable{importReadyRows.length === 1 ? '' : 's'}
					</ActionButton>
				</div>
			</div>

			{#if importRows.length > 0}
				<div class="mt-4 overflow-x-auto rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950/70">
					<div class="min-w-[34rem]">
						<div class="grid grid-cols-[4rem_minmax(0,1fr)_8rem_9rem] gap-3 border-b border-gray-100 px-4 py-2 text-xs font-medium text-gray-500 dark:border-gray-800 dark:text-gray-400">
							<span>Line</span>
							<span>Key</span>
							<span>Value</span>
							<span>Status</span>
						</div>
						<div class="max-h-72 overflow-y-auto">
							{#each importRows as row}
								<div class="grid grid-cols-[4rem_minmax(0,1fr)_8rem_9rem] gap-3 border-b border-gray-100 px-4 py-3 text-xs last:border-b-0 dark:border-gray-800">
									<span class="font-mono text-gray-500 dark:text-gray-400">{row.line}</span>
									<div class="min-w-0">
										<div class="truncate font-mono text-gray-900 dark:text-gray-100">
											{row.importStatus === 'invalid' ? 'Invalid line' : row.key}
										</div>
										{#if row.importStatus === 'invalid'}
											<p class="mt-0.5 truncate text-red-600 dark:text-red-300">{row.error}</p>
										{/if}
									</div>
									<span class="truncate text-gray-500 dark:text-gray-400">{importedValueLabel(row)}</span>
									<span class={`inline-flex min-w-0 items-center justify-center rounded-md border px-2 py-1 font-medium ${importStatusClass(row.importStatus)}`}>
										<span class="truncate">{importStatusLabel(row)}</span>
									</span>
								</div>
							{/each}
						</div>
					</div>
				</div>
			{/if}
		</div>
	{/if}

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
