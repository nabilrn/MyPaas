<script lang="ts">
	import { onDestroy } from 'svelte';
	import { AlertTriangle, Check, Copy, Download, LoaderCircle, Package } from '@lucide/svelte';
	import { api, type MigrationStatus } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import ActionLink from '$components/ActionLink.svelte';

	let migration: MigrationStatus | null = null;
	let preparingMigration = false;
	let confirmPrepare = false;
	let pollingInterval: ReturnType<typeof setInterval> | undefined;
	let copiedText: string | null = null;

	$: canPrepare = !migration || migration.status === 'failed' || migration.status === 'expired';
	$: downloadUrl = migration?.downloadToken ? `/api/admin/migrate/${migration.id}/download?token=${migration.downloadToken}` : '#';
	$: migrationCommand = migration?.downloadToken && typeof window !== 'undefined'
		? `git clone https://github.com/nabilrn/MyPaas.git mypaas && cd mypaas && bash scripts/install-vm.sh --migrate-url "${window.location.origin}/api/admin/migrate/${migration.id}/download?token=${migration.downloadToken}"`
		: '';

	onDestroy(() => {
		if (pollingInterval) clearInterval(pollingInterval);
	});

	async function startMigration() {
		if (preparingMigration) return;
		confirmPrepare = false;
		preparingMigration = true;
		migration = null;
		try {
			migration = await api.admin.prepareMigration();
			if (migration.status === 'preparing') startPolling();
			else preparingMigration = false;
		} catch (error) {
			toast.error('Failed to prepare migration');
			console.error(error);
			preparingMigration = false;
		}
	}

	function startPolling() {
		if (pollingInterval) clearInterval(pollingInterval);
		pollingInterval = setInterval(async () => {
			if (!migration?.id) return;
			try {
				const status = await api.admin.migrationStatus(migration.id);
				migration = status;
				if (status.status !== 'preparing') {
					if (pollingInterval) clearInterval(pollingInterval);
					pollingInterval = undefined;
					preparingMigration = false;
					if (status.status === 'failed') toast.error(status.error || 'Migration preparation failed');
					else if (status.status === 'ready') toast.success('Migration package is ready');
				}
			} catch (error) {
				console.error('Error polling migration status:', error);
			}
		}, 3000);
	}

	async function copyToClipboard(text: string, id: string) {
		if (!text) return;
		try {
			await navigator.clipboard.writeText(text);
			copiedText = id;
			toast.success('Copied');
			setTimeout(() => {
				if (copiedText === id) copiedText = null;
			}, 2000);
		} catch {
			toast.error('Failed to copy text');
		}
	}

	function formatHoursLeft(expiresAt?: string) {
		if (!expiresAt) return 0;
		const diff = new Date(expiresAt).getTime() - Date.now();
		return Math.max(0, Math.floor(diff / (1000 * 60 * 60)));
	}

	function formatBytes(value?: number) {
		if (!value || value <= 0) return '';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		let amount = value;
		let unit = 0;
		while (amount >= 1024 && unit < units.length - 1) {
			amount /= 1024;
			unit += 1;
		}
		return `${amount >= 10 ? amount.toFixed(1) : amount.toFixed(2)} ${units[unit]}`;
	}
</script>

<svelte:head>
	<title>Migration · MyPaaS</title>
</svelte:head>

<div class="page-shell">
	<section>
		<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Migration package</h2>
		<div class="mt-3 border-y border-gray-100 dark:border-neutral-800">
			{#if canPrepare && !preparingMigration}
				<div class="grid gap-3 py-3 sm:grid-cols-[10rem_minmax(0,1fr)_auto] sm:items-center">
					<p class="text-sm text-gray-500 dark:text-gray-400">Status</p>
					<div>
						<p class="text-sm font-medium text-gray-950 dark:text-white">{migration?.status === 'failed' ? 'Failed' : migration?.status === 'expired' ? 'Expired' : 'Not prepared'}</p>
						{#if confirmPrepare}<p class="mt-0.5 text-xs text-amber-700 dark:text-amber-300">Running container projects pause briefly while the package is created.</p>{/if}
					</div>
					{#if confirmPrepare}
						<div class="flex items-center gap-2"><ActionButton variant="ghost" size="sm" on:click={() => (confirmPrepare = false)}>Cancel</ActionButton><ActionButton variant="primary" size="sm" on:click={startMigration}><Package slot="icon" class="h-4 w-4" />Prepare package</ActionButton></div>
					{:else}
						<ActionButton variant="primary" size="sm" on:click={() => (confirmPrepare = true)}><Package slot="icon" class="h-4 w-4" />Prepare package</ActionButton>
					{/if}
				</div>
				{#if migration?.status === 'failed'}
					<div class="border-t border-gray-100 py-3 dark:border-neutral-800"><div class="alert-danger"><AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" /><p>{migration.error || 'Migration preparation failed.'}</p></div></div>
				{/if}
			{:else if migration?.status === 'preparing' || preparingMigration}
				<div class="flex items-center gap-3 py-5"><LoaderCircle class="h-5 w-5 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" /><div><p class="text-sm font-medium text-gray-950 dark:text-white">Preparing package</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">This can take a few minutes.</p></div></div>
			{:else if migration?.status === 'ready'}
				<div class="grid gap-3 py-3 sm:grid-cols-[10rem_minmax(0,1fr)_auto] sm:items-center">
					<p class="text-sm text-gray-500 dark:text-gray-400">Status</p>
					<div><p class="inline-flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-emerald-500"></span>Ready</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Expires in {formatHoursLeft(migration.expiresAt)}h{migration.sizeBytes ? ` · ${formatBytes(migration.sizeBytes)}` : ''}</p></div>
					<ActionLink href={downloadUrl} variant="secondary" size="sm"><Download slot="icon" class="h-4 w-4" />Download</ActionLink>
				</div>
			{/if}
		</div>
	</section>

	{#if migration?.status === 'ready'}
		<section>
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Restore</h2>
			<div class="mt-3 border-y border-gray-100 py-3 dark:border-neutral-800">
				<pre class="console-surface overflow-x-auto p-4"><code class="whitespace-pre-wrap">{migrationCommand}</code></pre>
				<ActionButton variant="secondary" size="sm" className="mt-2" on:click={() => copyToClipboard(migrationCommand, 'command')}>{#if copiedText === 'command'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}{copiedText === 'command' ? 'Copied' : 'Copy command'}</ActionButton>
			</div>
		</section>
	{/if}

	<details class="border-y border-gray-100 py-3 dark:border-neutral-800">
		<summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">What is included?</summary>
		<div class="mt-3 space-y-1 text-sm text-gray-500 dark:text-gray-400"><p>Control-plane state and supported host-managed data.</p><p>Compose volumes must be moved separately when reported by preflight.</p></div>
	</details>
</div>
