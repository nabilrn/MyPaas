<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { AlertTriangle, Check, Copy, Download, LoaderCircle, Package, RefreshCw, Save, X } from '@lucide/svelte';
	import { api, type MigrationStatus } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import ActionLink from '$components/ActionLink.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	let settings: Record<string, number> = {
		user_ram_quota_gb: 0,
		user_cpu_quota: 0,
		max_projects: 0,
		max_concurrent_deploys: 0,
		project_default_ram_mb: 0,
		project_default_cpu: 0,
		build_timeout_minutes: 0
	};
	let mcpToken = '';
	let loadingSettings = true;
	let savingSettings = false;
	let regeneratingToken = false;
	let confirmRegenerateToken = false;

	let migration: MigrationStatus | null = null;
	let preparingMigration = false;
	let pollingInterval: ReturnType<typeof setInterval>;
	let copiedText: string | null = null;

	const settingsConfig = [
		{ key: 'user_ram_quota_gb', label: 'User RAM quota', unit: 'GB' },
		{ key: 'user_cpu_quota', label: 'User CPU quota', unit: 'cores' },
		{ key: 'max_projects', label: 'Maximum projects', unit: 'projects' },
		{ key: 'max_concurrent_deploys', label: 'Concurrent deploys', unit: 'deploys' },
		{ key: 'project_default_ram_mb', label: 'Default project RAM', unit: 'MB' },
		{ key: 'project_default_cpu', label: 'Default project CPU', unit: 'cores' },
		{ key: 'build_timeout_minutes', label: 'Build timeout', unit: 'minutes' }
	];

	onMount(async () => {
		await loadSettings();
	});

	onDestroy(() => {
		if (pollingInterval) clearInterval(pollingInterval);
	});

	async function loadSettings() {
		try {
			const data = await api.admin.getSettings();
			mcpToken = data.mcp_api_token ?? '';
			delete data.mcp_api_token;
			settings = { ...settings, ...(data as Record<string, number>) };
		} catch (error) {
			toast.error('Failed to load settings');
			console.error(error);
		} finally {
			loadingSettings = false;
		}
	}

	async function saveSettings() {
		if (savingSettings) return;
		savingSettings = true;
		try {
			const updated = await api.admin.updateSettings(settings);
			settings = { ...settings, ...(updated as Record<string, number>) };
			toast.success('Settings saved successfully');
		} catch (error) {
			toast.error('Failed to save settings');
			console.error(error);
		} finally {
			savingSettings = false;
		}
	}

	function requestRegenerateToken() {
		confirmRegenerateToken = true;
	}

	async function regenerateToken() {
		if (regeneratingToken) return;
		regeneratingToken = true;
		try {
			const data = await api.admin.regenerateMCPToken();
			mcpToken = data.mcp_api_token ?? '';
			toast.success('MCP Token regenerated successfully');
			confirmRegenerateToken = false;
		} catch (error) {
			toast.error('Failed to regenerate token');
			console.error(error);
		} finally {
			regeneratingToken = false;
		}
	}

	async function startMigration() {
		if (preparingMigration) return;
		preparingMigration = true;
		try {
			migration = await api.admin.prepareMigration();
			if (migration.status === 'preparing') startPolling();
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
					clearInterval(pollingInterval);
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
		try {
			await navigator.clipboard.writeText(text);
			copiedText = id;
			toast.success('Copied!');
			setTimeout(() => {
				if (copiedText === id) copiedText = null;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy', err);
			toast.error('Failed to copy text');
		}
	}

	function formatHoursLeft(expiresAt?: string) {
		if (!expiresAt) return 0;
		const diff = new Date(expiresAt).getTime() - Date.now();
		return Math.max(0, Math.floor(diff / (1000 * 60 * 60)));
	}

	$: downloadUrl = migration?.downloadToken ? `/api/admin/migrate/${migration.id}/download?token=${migration.downloadToken}` : '#';
	$: migrationCommand = migration && typeof window !== 'undefined'
		? `git clone https://github.com/nabilrn/MyPaas.git mypaas && cd mypaas && bash scripts/install-vm.sh --migrate-url "${window.location.origin}/api/admin/migrate/${migration.id}/download?token=${migration.downloadToken}"`
		: '';
</script>

<svelte:head>
	<title>Admin Settings - MyPaas</title>
</svelte:head>

<div class="page-shell space-y-4 py-6">
	<p class="text-sm text-gray-500 dark:text-gray-400">Platform limits, AI-agent access, and migration workflows for this MyPaaS instance.</p>

	<SectionPanel title="Resource configuration" description="Default platform limits and project resource quotas.">
		<svelte:fragment slot="actions">
			<ActionButton variant="primary" size="sm" loading={savingSettings} loadingLabel="Saving" on:click={saveSettings} disabled={loadingSettings}>
				<Save slot="icon" class="h-4 w-4" />
				Save changes
			</ActionButton>
		</svelte:fragment>

		{#if loadingSettings}
			<div class="flex h-28 items-center justify-center">
				<LoaderCircle class="h-6 w-6 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
			</div>
		{:else}
			<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
				{#each settingsConfig as { key, label, unit }}
					<label class="block" for={key}>
						<span class="field-label">{label}</span>
						<div class="flex items-center gap-2">
							<input type="number" id={key} bind:value={settings[key]} class="field min-w-0 flex-1" />
							<span class="w-16 shrink-0 text-xs text-gray-500 dark:text-gray-400">{unit}</span>
						</div>
					</label>
				{/each}
			</div>
		{/if}
	</SectionPanel>

	<SectionPanel title="AI agent integration (MCP)" description="Authenticate a local MCP bridge against the MyPaaS backend." contentClass="p-0">
		<div class="p-4">
			<p class="max-w-3xl text-sm text-gray-600 dark:text-gray-300">The MCP server can deploy projects, inspect statuses, and manage platform resources using this API token.</p>
			<label class="mt-4 block max-w-2xl" for="mcp_token">
				<span class="field-label">MCP API token</span>
				<div class="flex items-center gap-2">
					<input type="text" id="mcp_token" readonly value={mcpToken || 'Not configured in .env'} class="field min-w-0 flex-1 bg-gray-50 font-mono text-sm text-gray-500 dark:bg-neutral-900 dark:text-gray-400" />
					{#if mcpToken}
						<ActionButton variant="secondary" size="sm" on:click={() => copyToClipboard(mcpToken, 'mcp_token')}>
							{#if copiedText === 'mcp_token'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}
							{copiedText === 'mcp_token' ? 'Copied' : 'Copy token'}
						</ActionButton>
					{/if}
				</div>
				<p class="field-hint">Saved automatically to the <code class="font-mono">MYPAAS_API_TOKEN</code> environment variable.</p>
			</label>
		</div>

		<div class="border-t border-gray-100 p-4 dark:border-neutral-800">
			{#if confirmRegenerateToken}
				<div class="alert-warning flex-wrap items-center justify-between">
					<div class="min-w-0 flex-1">
						<p class="font-medium">Regenerate MCP token?</p>
						<p class="mt-0.5">Agents using the old token will be disconnected.</p>
					</div>
					<div class="flex gap-2">
						<ActionButton variant="ghost" size="xs" on:click={() => (confirmRegenerateToken = false)} disabled={regeneratingToken}>
							<X slot="icon" class="h-3.5 w-3.5" />
							Cancel
						</ActionButton>
						<ActionButton variant="danger" size="xs" on:click={regenerateToken} loading={regeneratingToken} loadingLabel="Regenerating">
							<RefreshCw slot="icon" class="h-3.5 w-3.5" />
							Regenerate
						</ActionButton>
					</div>
				</div>
			{:else}
				<ActionButton variant="secondary" size="sm" on:click={requestRegenerateToken}>
					<RefreshCw slot="icon" class="h-4 w-4" />
					Regenerate token
				</ActionButton>
			{/if}
		</div>

		<details class="border-t border-gray-100 dark:border-neutral-800">
			<summary class="app-focus cursor-pointer select-none px-4 py-3 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-neutral-900">How to connect AI agents</summary>
			<div class="border-t border-gray-100 p-4 text-sm text-gray-600 dark:border-neutral-800 dark:text-gray-400">
				<p class="mb-3">Run the MCP server on your local machine to bridge your AI agent with this remote MyPaaS instance.</p>
				<ol class="mb-4 list-decimal space-y-1 pl-5">
					<li>Ensure <a href="https://go.dev/dl/" target="_blank" rel="noopener" class="font-medium text-gray-950 hover:underline dark:text-white">Go</a> is installed locally.</li>
					<li>Clone the MyPaaS repository to your machine.</li>
					<li>Add the configuration below to your agent's MCP settings and adjust the absolute repository path.</li>
				</ol>
				<pre class="console-surface overflow-x-auto p-4"><code>{`{
  "mcpServers": {
    "mypaas": {
      "command": "go",
      "args": [
        "run",
        "/absolute/path/to/MyPaas/backend/cmd/mcp/main.go"
      ],
      "env": {
        "MYPAAS_URL": "${typeof window !== 'undefined' ? window.location.origin : 'https://<your-domain>'}/api",
        "MYPAAS_API_TOKEN": "${mcpToken || '<your-token>'}"
      }
    }
  }
}`}</code></pre>
			</div>
		</details>
	</SectionPanel>

	<SectionPanel title="VM migration" description="Move this MyPaaS installation to a new server." contentClass="p-0">
		{#if !migration || (migration.status === 'failed' && !preparingMigration)}
			<div class="p-4">
				<p class="max-w-3xl text-sm text-gray-600 dark:text-gray-300">Generate a package containing the database, project volumes, configuration, and encrypted secrets for restoration on a new VM.</p>
				<div class="alert-warning mt-4">
					<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
					<p><strong>Running project containers will stop briefly during export</strong> and restart automatically after the package is created.</p>
				</div>
				<ActionButton variant="primary" className="mt-4" on:click={startMigration} loading={preparingMigration} loadingLabel="Preparing">
					<Package slot="icon" class="h-4 w-4" />
					Prepare migration package
				</ActionButton>
			</div>
		{:else if migration.status === 'preparing' || preparingMigration}
			<div class="flex flex-col items-center justify-center gap-3 px-4 py-10 text-center">
				<LoaderCircle class="h-7 w-7 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
				<p class="text-sm font-medium text-gray-950 dark:text-white">Creating migration package…</p>
				<p class="text-sm text-gray-500 dark:text-gray-400">This may take a few minutes depending on data size.</p>
			</div>
		{:else if migration.status === 'ready'}
			<div>
				<div class="flex flex-wrap items-center justify-between gap-3 p-4">
					<div>
						<p class="inline-flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-emerald-500"></span>Migration package ready</p>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Expires in {formatHoursLeft(migration.expiresAt)} hours.</p>
					</div>
					<ActionLink href={downloadUrl} variant="secondary" size="sm">
						<Download slot="icon" class="h-4 w-4" />
						Download package
					</ActionLink>
				</div>

				<div class="border-t border-gray-100 p-4 dark:border-neutral-800">
					<h3 class="text-[0.9375rem] font-semibold text-gray-950 dark:text-white">One-step automated migration</h3>
					<p class="mt-1 max-w-3xl text-sm text-gray-600 dark:text-gray-300">SSH into the new VM and run this command to install MyPaaS, download the package, restore state, and restart projects.</p>
					<div class="mt-4">
						<pre class="console-surface overflow-x-auto p-4 pr-4"><code class="whitespace-pre-wrap">{migrationCommand}</code></pre>
						<ActionButton variant="secondary" size="sm" className="mt-2" on:click={() => copyToClipboard(migrationCommand, 'automated_cmd')}>
							{#if copiedText === 'automated_cmd'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}
							{copiedText === 'automated_cmd' ? 'Copied' : 'Copy command'}
						</ActionButton>
					</div>
				</div>
			</div>
		{/if}
	</SectionPanel>
</div>
