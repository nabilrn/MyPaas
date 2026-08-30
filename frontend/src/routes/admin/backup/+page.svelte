<script lang="ts">
	import { onMount } from 'svelte';
	import { Cloud, Download, Database, ShieldCheck } from '@lucide/svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	let loading = true;
	let savingS3 = false;

	let s3Config = {
		endpoint: '',
		bucket: '',
		region: '',
		access_key: '',
		secret_key: ''
	};

	onMount(() => {
		void loadConfig();
	});

	async function loadConfig() {
		loading = true;
		try {
			const data = await api.admin.getSettings();
			s3Config = {
				endpoint: ((data as any).s3_endpoint as string) || '',
				bucket: ((data as any).s3_bucket as string) || '',
				region: ((data as any).s3_region as string) || '',
				access_key: ((data as any).s3_access_key as string) || '',
				secret_key: ((data as any).s3_secret_key as string) || ''
			};
		} catch (error) {
			toast.error('Failed to load backup configuration');
			console.error(error);
		} finally {
			loading = false;
		}
	}

	async function saveS3Config() {
		savingS3 = true;
		try {
			await api.admin.updateS3Config(s3Config);
			toast.success('S3 configuration saved');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save S3 configuration');
		} finally {
			savingS3 = false;
		}
	}

	function downloadBackup() {
		window.location.href = '/api/admin/backup/download';
	}
</script>

<svelte:head>
	<title>Backup · MyPaas</title>
</svelte:head>

<div class="page-shell space-y-4 py-6">
	<p class="px-5 text-sm text-gray-500 dark:text-gray-400">Manage automated off-site backups and on-demand manual archives of PostgreSQL and platform configuration.</p>

	<SectionPanel title="How backup works" description="MyPaaS automatically creates and manages backups to ensure your platform state is safe." contentClass="p-0">
		<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 lg:grid-cols-3 lg:divide-x lg:divide-y-0">
			<div class="flex gap-3 p-4">
				<Database class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
				<div>
					<p class="text-sm font-medium text-gray-950 dark:text-white">1. Database Snapshot</p>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">The platform performs a consistent pg_dump of the entire PostgreSQL database, capturing users, projects, and deployment state.</p>
				</div>
			</div>
			<div class="flex gap-3 p-4">
				<ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
				<div>
					<p class="text-sm font-medium text-gray-950 dark:text-white">2. Platform Config</p>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Core platform files and configuration are archived alongside the database to ensure a seamless restoration process.</p>
				</div>
			</div>
			<div class="flex gap-3 p-4">
				<Cloud class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
				<div>
					<p class="text-sm font-medium text-gray-950 dark:text-white">3. Off-site Sync</p>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">When configured, backups are automatically compressed, encrypted (if using HTTPS), and synced to an S3-compatible object storage provider daily.</p>
				</div>
			</div>
		</div>
	</SectionPanel>

	{#if loading}
		<div class="surface flex h-40 items-center justify-center">
			<LoadingIndicator label="Loading backup configuration" />
		</div>
	{:else}
		<SectionPanel title="S3 Automated Backup" description="Configure S3-compatible storage for automated daily backups.">
			<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.72fr)]">
				<div class="h-max rounded-md border border-gray-200 p-4 text-sm text-gray-600 dark:border-neutral-800 dark:text-gray-300">
					<p class="font-medium text-gray-950 dark:text-white">Cloudflare R2 Setup Guide</p>
					<div class="mt-2 space-y-2 leading-6">
						<p>1. Go to Cloudflare Dashboard &rarr; <strong>R2 Object Storage</strong> and create a bucket.</p>
						<p>2. Click <strong>Manage R2 API Tokens</strong> and create a token with <strong>Object Read &amp; Write</strong> permissions.</p>
						<p>3. Copy the <strong>S3 Endpoint</strong> from the bucket settings.</p>
						<p>4. Use Region <code>auto</code> unless you specified a specific jurisdiction.</p>
					</div>
				</div>
				<div class="space-y-4">
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="endpoint">S3 Endpoint</label>
						<input id="endpoint" type="text" bind:value={s3Config.endpoint} class="field w-full font-mono text-sm" placeholder="https://<account-id>.r2.cloudflarestorage.com" />
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="bucket">Bucket</label>
						<input id="bucket" type="text" bind:value={s3Config.bucket} class="field w-full font-mono text-sm" placeholder="mypaas-backups" />
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="region">Region</label>
						<input id="region" type="text" bind:value={s3Config.region} class="field w-full font-mono text-sm" placeholder="auto" />
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="access-key">Access Key</label>
						<input id="access-key" type="text" bind:value={s3Config.access_key} class="field w-full font-mono text-sm" />
					</div>
					<div>
						<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="secret-key">Secret Key</label>
						<input id="secret-key" type="password" bind:value={s3Config.secret_key} class="field w-full font-mono text-sm" />
					</div>
					<div class="pt-2">
						<ActionButton variant="primary" loading={savingS3} on:click={saveS3Config}>Save S3 Config</ActionButton>
					</div>
				</div>
			</div>
		</SectionPanel>

		<SectionPanel title="Manual Backup" description="Download a complete snapshot of the database and platform configuration immediately.">
			<div class="flex items-center gap-4">
				<ActionButton variant="secondary" size="sm" on:click={downloadBackup}>
					<Download slot="icon" class="h-4 w-4" />
					Download Backup
				</ActionButton>
				<p class="text-sm text-gray-500 dark:text-gray-400">
					Note: This includes the database dump and core config, but does not include user project volumes.
				</p>
			</div>
		</SectionPanel>
	{/if}
</div>
