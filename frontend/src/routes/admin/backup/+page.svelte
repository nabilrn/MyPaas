<script lang="ts">
	import { Download, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';

	let loading = true;
	let savingS3 = false;
	let s3Configured = false;
	let configuringS3 = false;

	let s3Config = {
		endpoint: '',
		bucket: '',
		region: 'auto',
		access_key: '',
		secret_key: ''
	};

	$: canSaveS3 = Boolean(
		s3Config.endpoint.trim()
		&& s3Config.bucket.trim()
		&& s3Config.region.trim()
		&& s3Config.access_key.trim()
		&& s3Config.secret_key.trim()
	);

	onMount(() => {
		void loadConfig();
	});

	async function loadConfig() {
		loading = true;
		try {
			const data = await api.admin.getSettings();
			s3Configured = Boolean((data as any).s3_configured);
		} catch (error) {
			toast.error('Failed to load backup configuration');
			console.error(error);
		} finally {
			loading = false;
		}
	}

	async function saveS3Config() {
		if (savingS3 || !canSaveS3) return;
		savingS3 = true;
		try {
			const response = await fetch('/api/admin/settings/s3', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					s3_endpoint: s3Config.endpoint.trim(),
					s3_bucket: s3Config.bucket.trim(),
					s3_region: s3Config.region.trim(),
					s3_access_key: s3Config.access_key.trim(),
					s3_secret_key: s3Config.secret_key
				})
			});
			const body = await response.json().catch(() => ({}));
			if (!response.ok) throw new Error(body.error?.message || 'Failed to save S3 configuration');
			s3Configured = true;
			configuringS3 = false;
			s3Config = { endpoint: '', bucket: '', region: 'auto', access_key: '', secret_key: '' };
			toast.success('Backup storage saved');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save S3 configuration');
		} finally {
			savingS3 = false;
		}
	}

	function closeS3Config() {
		if (savingS3) return;
		configuringS3 = false;
		s3Config = { endpoint: '', bucket: '', region: 'auto', access_key: '', secret_key: '' };
	}

	function downloadBackup() {
		window.location.href = '/api/admin/backup/download';
	}
</script>

<svelte:head>
	<title>Backup · MyPaaS</title>
</svelte:head>

<div class="page-shell">
	{#if loading}
		<div class="flex min-h-48 items-center justify-center"><LoadingIndicator label="Loading backup settings" /></div>
	{:else}
		<section>
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Automatic backup</h2>
			<div class="mt-3 divide-y divide-gray-100 border-y border-gray-100 dark:divide-neutral-800 dark:border-neutral-800">
				<div class="grid gap-3 py-3 sm:grid-cols-[10rem_minmax(0,1fr)_auto] sm:items-center">
					<p class="text-sm text-gray-500 dark:text-gray-400">Status</p>
					<p class="inline-flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white"><span class={`status-dot ${s3Configured ? 'bg-emerald-500' : 'bg-gray-400 dark:bg-gray-500'}`}></span>{s3Configured ? 'Configured' : 'Not configured'}</p>
					<ActionButton variant="secondary" size="sm" on:click={() => (configuringS3 = true)}>{s3Configured ? 'Change storage' : 'Configure storage'}</ActionButton>
				</div>
				<div class="grid gap-3 py-3 sm:grid-cols-[10rem_minmax(0,1fr)] sm:items-center">
					<p class="text-sm text-gray-500 dark:text-gray-400">Storage</p>
					<p class="text-sm text-gray-950 dark:text-white">S3-compatible</p>
				</div>
			</div>
		</section>

		<section>
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Manual backup</h2>
			<div class="mt-3 grid gap-3 border-y border-gray-100 py-3 sm:grid-cols-[10rem_minmax(0,1fr)_auto] sm:items-center dark:border-neutral-800">
				<p class="text-sm text-gray-500 dark:text-gray-400">Snapshot</p>
				<p class="text-sm text-gray-950 dark:text-white">Database and platform configuration <span class="text-gray-500 dark:text-gray-400">· project volumes excluded</span></p>
				<ActionButton variant="secondary" size="sm" on:click={downloadBackup}><Download slot="icon" class="h-4 w-4" />Download</ActionButton>
			</div>
		</section>
	{/if}
</div>

{#if configuringS3}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close backup storage setup" on:click={closeS3Config}></button>
		<div class="overlay relative max-h-[90vh] w-full max-w-xl overflow-hidden" role="dialog" aria-modal="true" aria-labelledby="backup-storage-title">
			<div class="panel-header flex items-start justify-between gap-3">
				<h2 id="backup-storage-title" class="panel-title">Backup storage</h2>
				<ActionButton variant="ghost" size="xs" on:click={closeS3Config} disabled={savingS3}><X slot="icon" class="h-4 w-4" />Close</ActionButton>
			</div>
			<div class="max-h-[calc(90vh-5rem)] space-y-4 overflow-y-auto p-4">
				<div><label class="field-label" for="endpoint">S3 endpoint</label><input id="endpoint" type="text" bind:value={s3Config.endpoint} class="field w-full font-mono text-sm" placeholder="https://<account-id>.r2.cloudflarestorage.com" /></div>
				<div class="grid gap-4 sm:grid-cols-2">
					<div><label class="field-label" for="bucket">Bucket</label><input id="bucket" type="text" bind:value={s3Config.bucket} class="field w-full font-mono text-sm" placeholder="mypaas-backups" /></div>
					<div><label class="field-label" for="region">Region</label><input id="region" type="text" bind:value={s3Config.region} class="field w-full font-mono text-sm" placeholder="auto" /></div>
				</div>
				<div><label class="field-label" for="access-key">Access key</label><input id="access-key" type="text" autocomplete="off" bind:value={s3Config.access_key} class="field w-full font-mono text-sm" /></div>
				<div><label class="field-label" for="secret-key">Secret key</label><input id="secret-key" type="password" autocomplete="new-password" bind:value={s3Config.secret_key} class="field w-full font-mono text-sm" /></div>
				<details class="border-y border-gray-100 py-3 dark:border-neutral-800">
					<summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">Cloudflare R2</summary>
					<ol class="mt-3 space-y-1 text-sm text-gray-500 dark:text-gray-400">
						<li>1. Create a bucket.</li>
						<li>2. Create an Object Read &amp; Write token.</li>
						<li>3. Copy the S3 endpoint and credentials here.</li>
					</ol>
				</details>
				<div class="flex justify-end gap-2"><ActionButton variant="ghost" on:click={closeS3Config} disabled={savingS3}>Cancel</ActionButton><ActionButton variant="primary" disabled={!canSaveS3} loading={savingS3} loadingLabel="Saving" on:click={saveS3Config}>Save</ActionButton></div>
			</div>
		</div>
	</div>
{/if}
