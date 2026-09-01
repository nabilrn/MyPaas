<script lang="ts">
	import { onMount } from 'svelte';
	import { Download } from '@lucide/svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
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

<div class="page-shell">
	{#if !loading}
		<SectionPanel title="S3 automated backup" description="Configure S3-compatible storage for automated daily backups." contentClass="p-0">
			<div class="grid lg:grid-cols-[minmax(18rem,0.72fr)_minmax(0,1fr)]">
				<div class="border-b border-gray-100/70 p-5 text-sm text-gray-600 dark:border-neutral-900 dark:text-gray-300 lg:border-b-0 lg:border-r">
					<p class="font-medium text-gray-950 dark:text-white">Cloudflare R2 setup</p>
					<div class="mt-3 space-y-2 leading-6">
						<p>1. Create a bucket in R2 Object Storage.</p>
						<p>2. Create an API token with Object Read &amp; Write permissions.</p>
						<p>3. Copy the S3 endpoint from the bucket settings.</p>
						<p>4. Use region <code>auto</code> unless a jurisdiction requires another value.</p>
					</div>
				</div>
				<div class="space-y-4 p-5">
					<div>
						<label class="field-label" for="endpoint">S3 endpoint</label>
						<input id="endpoint" type="text" bind:value={s3Config.endpoint} class="field w-full font-mono text-sm" placeholder="https://<account-id>.r2.cloudflarestorage.com" />
					</div>
					<div>
						<label class="field-label" for="bucket">Bucket</label>
						<input id="bucket" type="text" bind:value={s3Config.bucket} class="field w-full font-mono text-sm" placeholder="mypaas-backups" />
					</div>
					<div>
						<label class="field-label" for="region">Region</label>
						<input id="region" type="text" bind:value={s3Config.region} class="field w-full font-mono text-sm" placeholder="auto" />
					</div>
					<div>
						<label class="field-label" for="access-key">Access key</label>
						<input id="access-key" type="text" bind:value={s3Config.access_key} class="field w-full font-mono text-sm" />
					</div>
					<div>
						<label class="field-label" for="secret-key">Secret key</label>
						<input id="secret-key" type="password" bind:value={s3Config.secret_key} class="field w-full font-mono text-sm" />
					</div>
					<ActionButton variant="primary" loading={savingS3} on:click={saveS3Config}>Save S3 config</ActionButton>
				</div>
			</div>
		</SectionPanel>

		<SectionPanel title="Manual backup" description="Download a current database and platform-configuration snapshot.">
			<div class="flex flex-wrap items-center gap-4">
				<ActionButton variant="secondary" size="sm" on:click={downloadBackup}>
					<Download slot="icon" class="h-4 w-4" />
					Download backup
				</ActionButton>
				<p class="text-sm text-gray-500 dark:text-gray-400">Project volumes are not included.</p>
			</div>
		</SectionPanel>
	{/if}
</div>
