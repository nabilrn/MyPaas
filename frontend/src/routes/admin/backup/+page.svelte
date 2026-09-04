<script lang="ts">
	import { CheckCircle2, Download, ExternalLink, ShieldCheck, TriangleAlert } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';

	type ConnectionState = 'idle' | 'valid' | 'error';

	let loading = true;
	let testingConnection = false;
	let savingS3 = false;
	let s3Configured = false;
	let connectionState: ConnectionState = 'idle';
	let connectionMessage = '';
	let savedEndpoint = '';
	let savedBucket = '';
	let savedRegion = 'auto';

	let s3Config = {
		endpoint: '',
		bucket: '',
		region: 'auto',
		access_key: '',
		secret_key: ''
	};

	$: canTestConnection = Boolean(
		s3Config.endpoint.trim()
		&& s3Config.bucket.trim()
		&& s3Config.access_key.trim()
		&& s3Config.secret_key.trim()
	);
	$: canSaveS3 = canTestConnection && connectionState === 'valid';
	$: endpointLooksLikeR2 = /\.r2\.cloudflarestorage\.com\/?$/i.test(s3Config.endpoint.trim());

	onMount(() => {
		void loadConfig();
	});

	async function loadConfig() {
		loading = true;
		try {
			const data = await api.admin.getSettings();
			s3Configured = Boolean((data as any).s3_configured);
			savedEndpoint = String((data as any).s3_endpoint || '');
			savedBucket = String((data as any).s3_bucket || '');
			savedRegion = String((data as any).s3_region || 'auto');
			s3Config = {
				endpoint: savedEndpoint,
				bucket: savedBucket,
				region: savedRegion || 'auto',
				access_key: '',
				secret_key: ''
			};
		} catch (error) {
			toast.error('Failed to load backup configuration');
			console.error(error);
		} finally {
			loading = false;
		}
	}

	function invalidateConnection() {
		connectionState = 'idle';
		connectionMessage = '';
	}

	function requestBody() {
		return {
			s3_endpoint: s3Config.endpoint.trim(),
			s3_bucket: s3Config.bucket.trim(),
			s3_region: s3Config.region.trim() || 'auto',
			s3_access_key: s3Config.access_key.trim(),
			s3_secret_key: s3Config.secret_key
		};
	}

	async function testConnection() {
		if (testingConnection || savingS3 || !canTestConnection) return;
		testingConnection = true;
		connectionState = 'idle';
		connectionMessage = '';
		try {
			const response = await fetch('/api/admin/settings/s3?validate=1', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(requestBody())
			});
			const body = await response.json().catch(() => ({}));
			if (!response.ok) throw new Error(body.error?.message || 'Cloudflare R2 connection failed');
			connectionState = 'valid';
			connectionMessage = 'Connection verified. MyPaaS can access, write to, and clean up a probe object in this bucket.';
		} catch (error) {
			connectionState = 'error';
			connectionMessage = error instanceof Error ? error.message : 'Cloudflare R2 connection failed';
		} finally {
			testingConnection = false;
		}
	}

	async function saveS3Config() {
		if (savingS3 || testingConnection || !canSaveS3) return;
		savingS3 = true;
		try {
			const response = await fetch('/api/admin/settings/s3', {
				method: 'POST',
				credentials: 'include',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(requestBody())
			});
			const body = await response.json().catch(() => ({}));
			if (!response.ok) throw new Error(body.error?.message || 'Failed to save backup storage');

			s3Configured = true;
			savedEndpoint = s3Config.endpoint.trim();
			savedBucket = s3Config.bucket.trim();
			savedRegion = s3Config.region.trim() || 'auto';
			s3Config = {
				endpoint: savedEndpoint,
				bucket: savedBucket,
				region: savedRegion,
				access_key: '',
				secret_key: ''
			};
			connectionState = 'idle';
			connectionMessage = '';
			toast.success('Cloudflare R2 backup storage saved');
		} catch (error) {
			connectionState = 'error';
			connectionMessage = error instanceof Error ? error.message : 'Failed to save backup storage';
		} finally {
			savingS3 = false;
		}
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
		<div class="admin-backup-workspace w-full">
			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="flex flex-wrap items-start justify-between gap-3 px-4 py-3">
					<div class="flex min-w-0 items-start gap-3">
						<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-[color:var(--workspace-divider)] bg-white">
							<svg viewBox="0 0 24 24" class="h-5 w-5 text-[#F38020]" role="img" aria-label="Cloudflare">
								<path fill="currentColor" d="M16.5088 16.8447c.1475-.5068.0908-.9707-.1553-1.3154-.2246-.3164-.6045-.499-1.0615-.5205l-8.6592-.1123a.1559.1559 0 0 1-.1333-.0713c-.0283-.042-.0351-.0986-.021-.1553.0278-.084.1123-.1484.2036-.1562l8.7359-.1123c1.0351-.0489 2.1601-.8868 2.5537-1.9136l.499-1.3013c.0215-.0561.0293-.1128.0147-.168-.5625-2.5463-2.835-4.4453-5.5499-4.4453-2.5039 0-4.6284 1.6177-5.3876 3.8614-.4927-.3658-1.1187-.5625-1.794-.499-1.2026.119-2.1665 1.083-2.2861 2.2856-.0283.31-.0069.6128.0635.894C1.5683 13.171 0 14.7754 0 16.752c0 .1748.0142.3515.0352.5273.0141.083.0844.1475.1689.1475h15.9814c.0909 0 .1758-.0645.2032-.1553l.12-.4268zm2.7568-5.5634c-.0771 0-.1611 0-.2383.0112-.0566 0-.1054.0415-.127.0976l-.3378 1.1744c-.1475.5068-.0918.9707.1543 1.3164.2256.3164.6055.498 1.0625.5195l1.8437.1133c.0557 0 .1055.0263.1329.0703.0283.043.0351.1074.0214.1562-.0283.084-.1132.1485-.204.1553l-1.921.1123c-1.041.0488-2.1582.8867-2.5527 1.914l-.1406.3585c-.0283.0713.0215.1416.0986.1416h6.5977c.0771 0 .1474-.0489.169-.126.1122-.4082.1757-.837.1757-1.2803 0-2.6025-2.125-4.727-4.7344-4.727" />
							</svg>
						</div>
						<div class="min-w-0">
							<div class="flex flex-wrap items-center gap-2">
								<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Cloudflare R2</h2>
								<span class={`status-dot ${s3Configured ? 'bg-emerald-500' : 'bg-gray-400 dark:bg-gray-500'}`}></span>
								<span class="text-xs text-gray-500 dark:text-gray-400">{s3Configured ? 'Configured' : 'Not configured'}</span>
							</div>
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">S3-compatible object storage for MyPaaS backup archives.</p>
						</div>
					</div>
					<a
						href="https://developers.cloudflare.com/r2/get-started/s3/"
						target="_blank"
						rel="noreferrer"
						class="app-focus inline-flex h-9 items-center gap-1.5 rounded-md px-2.5 text-sm font-medium text-gray-600 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white"
					>
						Cloudflare R2 docs
						<ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
					</a>
				</div>

				<div class="grid border-t border-[color:var(--workspace-divider)] lg:grid-cols-[minmax(0,1.45fr)_minmax(20rem,0.75fr)]">
					<div class="px-4 py-4 lg:border-r lg:border-[color:var(--workspace-divider)]">
						<div class="grid gap-4 md:grid-cols-2">
							<div class="md:col-span-2">
								<label class="field-label" for="r2-endpoint">R2 S3 endpoint</label>
								<input
									id="r2-endpoint"
									type="url"
									bind:value={s3Config.endpoint}
									on:input={invalidateConnection}
									class="field w-full max-w-3xl font-mono text-sm"
									placeholder="https://<account-id>.r2.cloudflarestorage.com"
									autocomplete="off"
								/>
								<p class={`mt-1 text-xs ${s3Config.endpoint && !endpointLooksLikeR2 ? 'text-amber-600 dark:text-amber-300' : 'text-gray-500 dark:text-gray-400'}`}>
									{s3Config.endpoint && !endpointLooksLikeR2 ? 'This does not look like a standard Cloudflare R2 endpoint. MyPaaS will still validate it as S3-compatible storage.' : 'Copy the S3 API endpoint from your Cloudflare R2 account.'}
								</p>
							</div>
							<div>
								<label class="field-label" for="r2-bucket">Bucket</label>
								<input id="r2-bucket" type="text" bind:value={s3Config.bucket} on:input={invalidateConnection} class="field w-full font-mono text-sm" placeholder="mypaas-backups" autocomplete="off" />
							</div>
							<div>
								<label class="field-label" for="r2-region">Region</label>
								<input id="r2-region" type="text" bind:value={s3Config.region} on:input={invalidateConnection} class="field w-full max-w-xs font-mono text-sm" placeholder="auto" autocomplete="off" />
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Cloudflare R2 uses <span class="font-mono">auto</span>.</p>
							</div>
							<div>
								<label class="field-label" for="r2-access-key">Access Key ID</label>
								<input id="r2-access-key" type="text" bind:value={s3Config.access_key} on:input={invalidateConnection} class="field w-full font-mono text-sm" autocomplete="off" spellcheck="false" />
							</div>
							<div>
								<label class="field-label" for="r2-secret-key">Secret Access Key</label>
								<input id="r2-secret-key" type="password" bind:value={s3Config.secret_key} on:input={invalidateConnection} class="field w-full font-mono text-sm" autocomplete="new-password" />
							</div>
						</div>
						<p class="mt-4 text-xs text-gray-500 dark:text-gray-400">Use an R2 token with Object Read &amp; Write access to the selected bucket. Credentials are validated by the MyPaaS API, not from the browser directly.</p>
					</div>

					<div class="flex min-w-0 flex-col px-4 py-4">
						<div class="flex items-start gap-2.5">
							<ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
							<div class="min-w-0">
								<p class="text-sm font-medium text-gray-950 dark:text-white">Connection validation</p>
								<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">MyPaaS checks the bucket, writes a small probe object, then removes it. Saving repeats the same server-side validation.</p>
							</div>
						</div>

						{#if connectionState === 'valid'}
							<div class="mt-4 flex items-start gap-2 border-y border-emerald-200 py-3 text-emerald-700 dark:border-emerald-900 dark:text-emerald-300">
								<CheckCircle2 class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
								<p class="text-xs leading-5">{connectionMessage}</p>
							</div>
						{:else if connectionState === 'error'}
							<div class="mt-4 flex items-start gap-2 border-y border-red-200 py-3 text-red-700 dark:border-red-900 dark:text-red-300">
								<TriangleAlert class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
								<p class="text-xs leading-5">{connectionMessage}</p>
							</div>
						{:else if s3Configured}
							<div class="mt-4 border-y border-[color:var(--workspace-divider)] py-3">
								<p class="text-xs font-medium text-gray-950 dark:text-white">Current destination</p>
								<p class="mt-1 break-all font-mono text-xs text-gray-500 dark:text-gray-400">{savedBucket || 'Bucket configured'}</p>
								{#if savedEndpoint}<p class="mt-1 break-all font-mono text-xs text-gray-500 dark:text-gray-400">{savedEndpoint}</p>{/if}
							</div>
						{/if}

						<div class="mt-auto flex flex-wrap justify-end gap-2 pt-5">
							<ActionButton variant="secondary" size="sm" disabled={!canTestConnection || savingS3} loading={testingConnection} loadingLabel="Testing" on:click={testConnection}>Test connection</ActionButton>
							<ActionButton variant="primary" size="sm" disabled={!canSaveS3 || testingConnection} loading={savingS3} loadingLabel="Saving" on:click={saveS3Config}>Save storage</ActionButton>
						</div>
					</div>
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-3">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Manual backup</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Generate a fresh archive on demand.</p>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
					<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
						<p class="text-xs text-gray-500 dark:text-gray-400">Included</p>
						<p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">PostgreSQL database + MyPaaS environment configuration</p>
					</div>
					<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:border-t-0 lg:border-r">
						<p class="text-xs text-gray-500 dark:text-gray-400">Excluded</p>
						<p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">Project volumes and container filesystems</p>
					</div>
					<div class="flex items-center justify-end border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:border-t-0">
						<ActionButton variant="secondary" size="sm" on:click={downloadBackup}><Download slot="icon" class="h-4 w-4" />Download backup</ActionButton>
					</div>
				</div>
			</section>
		</div>
	{/if}
</div>
