<script lang="ts">
	import { onDestroy } from 'svelte';
	import { AlertTriangle, Check, Copy, Download, LoaderCircle, Package } from '@lucide/svelte';
	import { api, type MigrationStatus } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import ActionLink from '$components/ActionLink.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	let migration: MigrationStatus | null = null;
	let preparingMigration = false;
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
		} catch (error) {
			console.error(error);
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
	<title>Migration · MyPaas</title>
</svelte:head>

<div class="page-shell">
	<SectionPanel title="Migration safety">
		<div class="alert-warning">
			<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
			<div>
				<p class="font-medium">Container-backed projects pause briefly while the package is created.</p>
				<p class="mt-1">Engine-managed Compose volumes are not included. Move them separately if preflight reports any.</p>
			</div>
		</div>
	</SectionPanel>

	<SectionPanel title="Migration package" description="Prepare a temporary package and restore it on the destination VM." contentClass="p-0">
		{#if canPrepare && !preparingMigration}
			<div class="p-4 lg:p-5">
				{#if migration?.status === 'failed'}
					<div class="alert-danger mb-4">
						<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
						<div>
							<p class="font-medium">Migration preparation failed</p>
							<p class="mt-1">{migration.error || 'The backend rejected or could not complete the export. Review the reported storage/runtime condition and retry.'}</p>
						</div>
					</div>
				{:else if migration?.status === 'expired'}
					<div class="alert-neutral mb-4">
						<p>The previous migration package expired. Prepare a new package to generate a fresh download token.</p>
					</div>
				{/if}

				<p class="max-w-3xl text-sm text-gray-600 dark:text-gray-300">The generated package captures control-plane state plus the host-managed data supported by the migration workflow. Use the generated restore command on a new MyPaaS VM.</p>
				<ActionButton variant="primary" className="mt-4" on:click={startMigration} loading={preparingMigration} loadingLabel="Preparing">
					<Package slot="icon" class="h-4 w-4" />
					Prepare migration package
				</ActionButton>
			</div>
		{:else if migration?.status === 'preparing' || preparingMigration}
			<div class="flex flex-col items-center justify-center gap-3 px-4 py-10 text-center">
				<LoaderCircle class="h-7 w-7 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
				<p class="text-sm font-medium text-gray-950 dark:text-white">Creating migration package…</p>
				<p class="text-sm text-gray-500 dark:text-gray-400">Preflight, runtime quiescing, archive creation, and runtime restoration can take a few minutes depending on the installation.</p>
			</div>
		{:else if migration?.status === 'ready'}
			<div>
				<div class="flex flex-wrap items-center justify-between gap-3 p-4 lg:p-5">
					<div>
						<p class="inline-flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-emerald-500"></span>Migration package ready</p>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
							Expires in {formatHoursLeft(migration.expiresAt)} hours{migration.sizeBytes ? ` · ${formatBytes(migration.sizeBytes)}` : ''}.
						</p>
					</div>
					<ActionLink href={downloadUrl} variant="secondary" size="sm">
						<Download slot="icon" class="h-4 w-4" />
						Download package
					</ActionLink>
				</div>

				<div class="border-t border-gray-100/70 p-4 dark:border-neutral-900 lg:p-5">
					<h3 class="text-[0.9375rem] font-semibold text-gray-950 dark:text-white">Restore on the new VM</h3>
					<p class="mt-1 max-w-3xl text-sm text-gray-600 dark:text-gray-300">SSH into a fresh Linux VM and run the generated command. It clones MyPaaS and passes the temporary migration package URL to the installer for restore.</p>
					<pre class="console-surface mt-4 overflow-x-auto p-4"><code class="whitespace-pre-wrap">{migrationCommand}</code></pre>
					<ActionButton variant="secondary" size="sm" className="mt-2" on:click={() => copyToClipboard(migrationCommand, 'command')}>
						{#if copiedText === 'command'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}
						{copiedText === 'command' ? 'Copied' : 'Copy command'}
					</ActionButton>
				</div>
			</div>
		{/if}
	</SectionPanel>
</div>
