<script lang="ts">
	import { Plus, RefreshCw, Save, Upload, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import EmptyState from '$components/EmptyState.svelte';
	import ErrorState from '$components/ErrorState.svelte';
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
			const revealed = await api.env.reveal($page.params.id ?? '', row.key);
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
			await api.env.delete($page.params.id ?? '', key);
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
			await api.env.bulkUpdate($page.params.id ?? '', {
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
			await api.env.bulkUpdate($page.params.id ?? '', { vars: [{ key: normalizeEnvKey(newKey), value: newValue }] });
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
		importing = true;
		adding = false;
	}

	function clearImport() {
		importText = '';
		importFileName = '';
		confirmOverwrite = false;
		if (importFileInput) importFileInput.value = '';
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
			await api.env.bulkUpdate($page.params.id ?? '', {
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
		if (!background) loading = true;
		error = '';
		try {
			const rows = await api.env.list($page.params.id ?? '');
			vars = rows.map((v) => ({ ...v, value: '', revealed: false, dirty: false, revealing: false }));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load environment variables';
			if (background) toast.error(error);
		} finally {
			if (!background) loading = false;
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

	function importStatusDotClass(status: ImportStatus) {
		return {
			new: 'bg-emerald-500',
			overwrite: 'bg-amber-500',
			duplicate: 'bg-gray-400 dark:bg-gray-500',
			invalid: 'bg-red-500'
		}[status];
	}

	function importStatusTextClass(status: ImportStatus) {
		if (status === 'invalid') return 'text-red-700 dark:text-red-300';
		if (status === 'overwrite') return 'text-amber-700 dark:text-amber-200';
		return 'text-gray-600 dark:text-gray-300';
	}

	function importedValueLabel(row: ImportRow) {
		if (row.importStatus === 'invalid') return 'Not saved';
		return row.value === '' ? 'Empty value' : 'Value parsed';
	}
</script>

<svelte:head>
	<title>Environment · MyPaas</title>
</svelte:head>

<SectionPanel title="Environment variables" description="Encrypted at rest. Reveal only when you need to inspect a stored value." contentClass="p-0">
	<svelte:fragment slot="actions">
		<div class="flex flex-wrap items-center gap-2">
			{#if hasDirty}
				<span class="inline-flex items-center gap-2 text-sm text-amber-700 dark:text-amber-200">
					<span class="status-dot bg-amber-500"></span>
					{dirtyCount} unsaved
				</span>
				<ActionButton variant="primary" on:click={handleSave} loading={savingChanges} loadingLabel="Saving">
					<Save slot="icon" class="h-4 w-4" />
					Save changes
				</ActionButton>
			{/if}
			{#if !importing}
				<ActionButton variant="secondary" on:click={openImport} disabled={loading || Boolean(error)}>
					<Upload slot="icon" class="h-4 w-4" />
					Import .env
				</ActionButton>
			{/if}
			{#if !adding}
				<ActionButton variant="primary" on:click={() => (adding = true)}>
					<Plus slot="icon" class="h-4 w-4" />
					Add variable
				</ActionButton>
			{/if}
		</div>
	</svelte:fragment>

	{#if importing && !loading && !error}
		<div class="border-b border-gray-100 bg-gray-50/50 dark:border-neutral-800 dark:bg-neutral-900/40">
			<div class="grid xl:grid-cols-[minmax(0,1fr)_18rem]">
				<div class="min-w-0 p-4">
					<div class="mb-3 flex flex-wrap items-start justify-between gap-3">
						<div class="min-w-0">
							<h3 class="text-[0.9375rem] font-semibold text-gray-950 dark:text-white">Import .env</h3>
							<p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">Paste environment content or load a local file for review before saving.</p>
							{#if importFileName}<p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{importFileName}</p>{/if}
						</div>
						<div class="flex flex-wrap gap-2">
							<input bind:this={importFileInput} type="file" accept=".env,.env.example,.env.sample,.env.template,.txt,text/plain" class="sr-only" on:change={handleImportFile} />
							<ActionButton variant="secondary" size="sm" on:click={() => importFileInput?.click()} disabled={savingImport}>
								<Upload slot="icon" class="h-4 w-4" />
								Upload file
							</ActionButton>
							<ActionButton variant="ghost" size="sm" on:click={clearImport} disabled={savingImport || !importText}>
								<RefreshCw slot="icon" class="h-4 w-4" />
								Clear
							</ActionButton>
							<ActionButton variant="ghost" size="sm" on:click={closeImport} disabled={savingImport}>
								<X slot="icon" class="h-4 w-4" />
								Close
							</ActionButton>
						</div>
					</div>

					<textarea
						bind:value={importText}
						rows="8"
						placeholder={'KEY=value\nSECRET="quoted value"\nEMPTY='}
						class="field min-h-44 w-full resize-y font-mono text-sm leading-5"
						disabled={savingImport}
					></textarea>
				</div>

				<div class="border-t border-gray-100 p-4 dark:border-neutral-800 xl:border-l xl:border-t-0">
					<div class="grid grid-cols-2 divide-x divide-y divide-gray-100 dark:divide-neutral-800">
						<div class="p-3">
							<p class="metric-label">New</p>
							<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{importCounts.newCount}</p>
						</div>
						<div class="p-3">
							<p class="metric-label">Overwrite</p>
							<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{importCounts.overwrite}</p>
						</div>
						<div class="p-3">
							<p class="metric-label">Duplicate</p>
							<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{importCounts.duplicate}</p>
						</div>
						<div class="p-3">
							<p class="metric-label">Invalid</p>
							<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{importCounts.invalid}</p>
						</div>
					</div>

					{#if importCounts.overwrite > 0}
						<label class="mt-4 flex items-start gap-2 text-sm text-amber-800 dark:text-amber-200">
							<input type="checkbox" bind:checked={confirmOverwrite} class="mt-0.5 h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-neutral-700 dark:text-white dark:focus:ring-white" disabled={savingImport} />
							<span>Allow {importCounts.overwrite} existing variable{importCounts.overwrite === 1 ? '' : 's'} to be overwritten</span>
						</label>
					{/if}

					{#if hasDirty}
						<div class="alert-warning mt-4 py-2.5 text-sm">Save or discard existing drafts before importing.</div>
					{/if}

					<ActionButton variant="primary" className="mt-4" on:click={handleImportSave} loading={savingImport} loadingLabel="Saving" disabled={!canSaveImport} full>
						<Save slot="icon" class="h-4 w-4" />
						Save {importReadyRows.length} variable{importReadyRows.length === 1 ? '' : 's'}
					</ActionButton>
				</div>
			</div>

			{#if importRows.length > 0}
				<div class="overflow-x-auto border-t border-gray-100 bg-white dark:border-neutral-800 dark:bg-neutral-950/50">
					<table class="data-table min-w-[38rem]">
						<thead>
							<tr>
								<th class="w-16">Line</th>
								<th>Key</th>
								<th class="w-32">Value</th>
								<th class="w-40">Status</th>
							</tr>
						</thead>
						<tbody>
							{#each importRows as row}
								<tr>
									<td class="font-mono text-xs">{row.line}</td>
									<td>
										<p class="truncate font-mono text-sm text-gray-900 dark:text-gray-100">{row.importStatus === 'invalid' ? 'Invalid line' : row.key}</p>
										{#if row.importStatus === 'invalid'}<p class="mt-0.5 truncate text-xs text-red-600 dark:text-red-300">{row.error}</p>{/if}
									</td>
									<td class="text-sm">{importedValueLabel(row)}</td>
									<td>
										<span class={`inline-flex items-center gap-2 text-sm ${importStatusTextClass(row.importStatus)}`}>
											<span class={`status-dot ${importStatusDotClass(row.importStatus)}`}></span>
											<span class="truncate">{importStatusLabel(row)}</span>
										</span>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	{/if}

	{#if loading}
		<div class="space-y-2 p-4">
			{#each [1, 2, 3] as _}<div class="h-12 animate-pulse rounded-md bg-gray-100 dark:bg-neutral-800"></div>{/each}
		</div>
	{:else if error}
		<ErrorState title="Could not load environment variables" message={error} on:retry={() => void load()} />
	{:else if vars.length === 0}
		<EmptyState title="No environment variables yet." description="Add variables when the app needs runtime configuration or secrets." compact />
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

	{#if adding}
		<div class="grid gap-3 border-t border-gray-100 bg-gray-50/50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40 lg:grid-cols-[14rem_minmax(0,1fr)_auto] lg:items-end">
			<div>
				<label class="field-label" for="new-env-key">Key</label>
				<input id="new-env-key" type="text" value={newKey} on:input={(event) => (newKey = normalizeEnvKey((event.currentTarget as HTMLInputElement).value))} placeholder="KEY" class="field w-full font-mono uppercase" />
			</div>
			<div>
				<label class="field-label" for="new-env-value">Value</label>
				<input id="new-env-value" type="text" bind:value={newValue} placeholder="value" class="field w-full font-mono" />
			</div>
			<div class="flex gap-2">
				<ActionButton variant="primary" on:click={handleAdd} loading={savingNewVar} loadingLabel="Adding" disabled={!canAdd}>
					<Plus slot="icon" class="h-4 w-4" />
					Add
				</ActionButton>
				<ActionButton variant="ghost" on:click={() => (adding = false)} disabled={savingNewVar}>
					<X slot="icon" class="h-4 w-4" />
					Cancel
				</ActionButton>
			</div>
		</div>
	{/if}
</SectionPanel>
