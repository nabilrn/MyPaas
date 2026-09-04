<script lang="ts">
	import { onDestroy } from 'svelte';
	import { AlertTriangle, Check, Copy, Download, LoaderCircle, Package } from '@lucide/svelte';
	import { api, type MigrationStatus } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import ActionLink from '$components/ActionLink.svelte';
	import ConfirmActionDialog from '$components/ConfirmActionDialog.svelte';
	import MigrationTransferIllustration from '$components/MigrationTransferIllustration.svelte';

	type MigrationVisualState = 'idle' | 'preparing' | 'ready' | 'failed' | 'expired';

	let migration: MigrationStatus | null = null;
	let preparingMigration = false;
	let confirmPrepare = false;
	let pollingInterval: ReturnType<typeof setInterval> | undefined;
	let copiedText: string | null = null;

	$: canPrepare = !migration || migration.status === 'failed' || migration.status === 'expired';
	$: visualState = (preparingMigration ? 'preparing' : migration?.status || 'idle') as MigrationVisualState;
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
					else if (status.status === 'ready') toast.success('Migration package ready');
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
			toast.error('Failed to copy');
		}
	}

	function formatHoursLeft(expiresAt?: string) {
		if (!expiresAt) return 0;
		const diff = new Date(expiresAt).getTime() - Date.now();
		return Math.max(0, Math.ceil(diff / (1000 * 60 * 60)));
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

	function statusTitle() {
		if (preparingMigration || migration?.status === 'preparing') return 'Preparing package';
		if (migration?.status === 'ready') return 'Ready to move';
		if (migration?.status === 'failed') return 'Preparation failed';
		if (migration?.status === 'expired') return 'Package expired';
		return 'Not prepared';
	}

	function statusDescription() {
		if (preparingMigration || migration?.status === 'preparing') return 'Capturing databases, configuration, and persistent runtime data.';
		if (migration?.status === 'ready') {
			const size = formatBytes(migration.sizeBytes);
			return `${size ? `${size} · ` : ''}download expires in ${formatHoursLeft(migration.expiresAt)}h.`;
		}
		if (migration?.status === 'failed') return 'Review the error below, then prepare a new package.';
		if (migration?.status === 'expired') return 'Prepare a new package before moving to the destination VM.';
		return 'Create a portable package for a new MyPaaS VM.';
	}
</script>

<svelte:head>
	<title>Migration · MyPaaS</title>
</svelte:head>

<div class="page-shell migration-workspace w-full">
	<section class="border-b border-[color:var(--workspace-divider)]">
		<div class="grid lg:grid-cols-[minmax(0,1.05fr)_minmax(28rem,0.95fr)]">
			<div class="flex min-w-0 flex-col justify-between px-4 py-4 lg:min-h-[20rem] lg:border-r lg:border-[color:var(--workspace-divider)]">
				<div>
					<p class="text-xs font-medium uppercase tracking-[0.14em] text-gray-500 dark:text-gray-400">Migration package</p>
					<h2 class="mt-2 text-base font-semibold text-gray-950 dark:text-white">Move MyPaaS to another VM</h2>
					<p class="mt-1 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">Capture the platform database, shared project databases, configuration, and persistent runtime data into one portable archive.</p>
				</div>

				<div class="mt-6 flex flex-col gap-3 border-t border-[color:var(--workspace-divider)] pt-3 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<p class="inline-flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white">
							{#if preparingMigration || migration?.status === 'preparing'}
								<LoaderCircle class="h-4 w-4 shrink-0 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
							{:else}
								<span class={`status-dot ${migration?.status === 'ready' ? 'bg-emerald-500' : migration?.status === 'failed' ? 'bg-red-500' : migration?.status === 'expired' ? 'bg-amber-500' : 'bg-gray-400 dark:bg-gray-500'}`}></span>
							{/if}
							{statusTitle()}
						</p>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{statusDescription()}</p>
					</div>
					<div class="flex shrink-0 items-center gap-2">
						{#if migration?.status === 'ready'}
							<ActionLink href={downloadUrl} variant="secondary" size="sm"><Download slot="icon" class="h-4 w-4" />Download package</ActionLink>
						{:else if canPrepare && !preparingMigration}
							<ActionButton variant="primary" size="sm" on:click={() => (confirmPrepare = true)}><Package slot="icon" class="h-4 w-4" />Prepare package</ActionButton>
						{:else}
							<ActionButton variant="secondary" size="sm" disabled><LoaderCircle slot="icon" class="h-4 w-4 animate-spin motion-reduce:animate-none" />Preparing</ActionButton>
						{/if}
					</div>
				</div>

				{#if migration?.status === 'failed'}
					<div class="mt-3 alert-danger"><AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" /><p>{migration.error || 'Migration preparation failed.'}</p></div>
				{/if}
			</div>

			<div class="flex min-h-[20rem] items-center justify-center px-6 py-6">
				<div class="w-full max-w-[46rem]">
					<MigrationTransferIllustration state={visualState} />
				</div>
			</div>
		</div>
	</section>

	<section class="border-b border-[color:var(--workspace-divider)]">
		<div class="px-4 py-2.5">
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Captured state</h2>
			<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">The export follows the data MyPaaS actually restores on the destination VM.</p>
		</div>
		<div class="grid border-t border-[color:var(--workspace-divider)] sm:grid-cols-2 xl:grid-cols-4">
			<div class="px-4 py-3 sm:border-r sm:border-[color:var(--workspace-divider)]">
				<p class="text-sm font-medium text-gray-950 dark:text-white">Platform database</p>
				<p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">System database and database roles when available.</p>
			</div>
			<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-t-0 xl:border-r">
				<p class="text-sm font-medium text-gray-950 dark:text-white">Project databases</p>
				<p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">Discovered shared PostgreSQL databases.</p>
			</div>
			<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-r xl:border-t-0">
				<p class="text-sm font-medium text-gray-950 dark:text-white">Persistent data</p>
				<p class="mt-0.5 font-mono text-xs leading-5 text-gray-500 dark:text-gray-400">bind mounts · Compose named volumes · static</p>
			</div>
			<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 xl:border-t-0">
				<p class="text-sm font-medium text-gray-950 dark:text-white">Configuration</p>
				<p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">MyPaaS <span class="font-mono">.env</span> when the source file is readable.</p>
			</div>
		</div>
	</section>

	<section class="border-b border-[color:var(--workspace-divider)]">
		<div class="grid lg:grid-cols-[18rem_minmax(0,1fr)]">
			<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
				<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Runtime safety</h2>
				<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Running project runtimes pause during capture. Compose named volumes are staged only after their consumers stop, then runtimes are started again before the package is marked ready.</p>
			</div>
			<div class="grid sm:grid-cols-3">
				<div class="px-4 py-3 sm:border-r sm:border-[color:var(--workspace-divider)]"><p class="text-xs text-gray-500 dark:text-gray-400">1 · Preflight</p><p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">Inspect storage</p></div>
				<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-t-0 sm:border-r"><p class="text-xs text-gray-500 dark:text-gray-400">2 · Capture</p><p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">Pause and export</p></div>
				<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:border-t-0"><p class="text-xs text-gray-500 dark:text-gray-400">3 · Resume</p><p class="mt-1 text-sm font-medium text-gray-950 dark:text-white">Restore runtime state</p></div>
			</div>
		</div>
	</section>

	{#if migration?.status === 'ready'}
		<section class="border-b border-[color:var(--workspace-divider)]">
			<div class="grid lg:grid-cols-[18rem_minmax(0,1fr)]">
				<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Restore on the new server</h2>
					<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Run the generated command on the destination VM. The download token is embedded in the migration URL.</p>
				</div>
				<div class="min-w-0 px-4 py-3">
					<pre class="console-surface max-h-52 overflow-auto p-3"><code class="whitespace-pre-wrap">{migrationCommand}</code></pre>
					<ActionButton variant="secondary" size="sm" className="mt-2" on:click={() => copyToClipboard(migrationCommand, 'command')}>{#if copiedText === 'command'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}{copiedText === 'command' ? 'Copied' : 'Copy command'}</ActionButton>
				</div>
			</div>
		</section>
	{/if}
</div>

<ConfirmActionDialog
	open={confirmPrepare}
	title="Prepare migration package?"
	description="MyPaaS will briefly pause running project runtimes while it creates a consistent export."
	confirmLabel="Prepare package"
	busy={preparingMigration}
	busyLabel="Preparing"
	on:confirm={startMigration}
	on:cancel={() => (confirmPrepare = false)}
>
	<p>The export captures platform and shared databases, bind-mounted persistent data, Compose named volumes, and readable platform configuration. Project runtimes are resumed before the package becomes downloadable.</p>
</ConfirmActionDialog>
