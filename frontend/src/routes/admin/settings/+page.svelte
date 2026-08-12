<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Check, Copy, Download, Loader2, AlertTriangle, Package, RefreshCw } from '@lucide/svelte';
	import { api, type MigrationStatus } from '$api';
	import { toast } from '$stores/toast';
	import PageHeader from '$components/PageHeader.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import ActionButton from '$components/ActionButton.svelte';

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
		{ key: 'user_ram_quota_gb', label: 'User RAM Quota (GB)' },
		{ key: 'user_cpu_quota', label: 'User CPU Quota (cores)' },
		{ key: 'max_projects', label: 'Max Projects' },
		{ key: 'max_concurrent_deploys', label: 'Max Concurrent Deploys' },
		{ key: 'project_default_ram_mb', label: 'Default Project RAM (MB)' },
		{ key: 'project_default_cpu', label: 'Default Project CPU (cores)' },
		{ key: 'build_timeout_minutes', label: 'Build Timeout (minutes)' }
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
			if (migration.status === 'preparing') {
				startPolling();
			}
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
					if (status.status === 'failed') {
						toast.error(status.error || 'Migration preparation failed');
					} else if (status.status === 'ready') {
						toast.success('Migration package is ready');
					}
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

	function formatBytes(bytes?: number) {
		if (!bytes) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
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

<div class="page-shell py-6">
	<PageHeader title="Platform Settings" description="Manage platform configurations and migrations" />

	<SectionPanel title="Resource Configuration" description="Configure default platform limits and resource quotas." className="mb-8">
		{#if loadingSettings}
			<div class="flex h-32 items-center justify-center">
				<Loader2 class="h-6 w-6 animate-spin text-gray-950 dark:text-white" />
			</div>
		{:else}
			<div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
				{#each settingsConfig as { key, label }}
					<div class="space-y-1.5">
						<label for={key} class="block text-sm font-medium text-gray-700 dark:text-gray-300">
							{label}
						</label>
						<input
							type="number"
							id={key}
							bind:value={settings[key]}
							class="field block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-gray-950 focus:outline-none focus:ring-1 focus:ring-gray-950 dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:focus:border-white dark:focus:ring-white"
						/>
					</div>
				{/each}
			</div>
		{/if}

		<svelte:fragment slot="actions">
			<ActionButton variant="primary" loading={savingSettings} on:click={saveSettings} disabled={loadingSettings}>
				Save Changes
			</ActionButton>
		</svelte:fragment>
	</SectionPanel>

	<SectionPanel title="AI Agent Integration (MCP)" description="Connect an AI Agent to MyPaas using the Model Context Protocol." className="mb-8">
		<div class="space-y-4">
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Use this API Token to authenticate your MCP server with the MyPaas backend. The MCP server can deploy projects, view statuses, and manage your platform automatically.
			</p>
			
			<div class="space-y-1.5">
				<label for="mcp_token" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
					MCP API Token
				</label>
				<div class="flex max-w-lg items-center gap-2">
					<input
						type="text"
						id="mcp_token"
						readonly
						value={mcpToken || 'Not configured in .env'}
						class="field block w-full rounded-md border border-gray-300 bg-gray-50 px-3 py-2 text-sm text-gray-500 focus:border-gray-950 focus:outline-none focus:ring-1 focus:ring-gray-950 dark:border-gray-700 dark:bg-gray-800/50 dark:text-gray-400 dark:focus:border-white dark:focus:ring-white"
					/>
					{#if mcpToken}
						<ActionButton variant="secondary" on:click={() => copyToClipboard(mcpToken, 'mcp_token')}>
							{#if copiedText === 'mcp_token'}
								<Check class="h-4 w-4" />
							{:else}
								<Copy class="h-4 w-4" />
							{/if}
						</ActionButton>
					{/if}
				</div>
				<p class="mt-2 text-xs text-gray-500">
					This token is saved automatically to the <code>MYPAAS_API_TOKEN</code> environment variable.
				</p>
				
				<div class="mt-4 border-t border-gray-100 pt-4 dark:border-gray-800">
					{#if confirmRegenerateToken}
						<div class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
							<p>Regenerating the token will disconnect any AI agents using the old token.</p>
							<div class="mt-3 flex flex-wrap gap-2">
								<ActionButton variant="ghost" size="xs" on:click={() => (confirmRegenerateToken = false)}>
									Cancel
								</ActionButton>
								<ActionButton variant="danger" size="xs" on:click={regenerateToken} loading={regeneratingToken} loadingLabel="Regenerating...">
									Regenerate now
								</ActionButton>
							</div>
						</div>
					{:else}
						<ActionButton on:click={requestRegenerateToken}>
							Regenerate token
						</ActionButton>
					{/if}
				</div>

				<details class="mt-4 rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900/50">
					<summary class="cursor-pointer select-none px-4 py-3 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800/50">
						How to connect AI Agents (Cursor, Cline, Claude Desktop)
					</summary>
					<div class="border-t border-gray-200 p-4 text-sm text-gray-600 dark:border-gray-800 dark:text-gray-400">
						<p class="mb-3">Since MyPaas runs remotely, you must run the MCP server on your local machine to securely bridge your AI agent with this server.</p>
						<ol class="mb-4 list-decimal pl-5 space-y-1">
							<li>Ensure you have <a href="https://go.dev/dl/" target="_blank" class="font-medium text-gray-950 hover:underline dark:text-white">Go</a> installed locally.</li>
							<li>Clone the MyPaas repository to your machine if you haven't already.</li>
							<li>Add the configuration below to your agent's MCP settings (e.g. <code>cline_mcp.json</code> or Cursor settings). Ensure you adjust the absolute path to your local clone!</li>
						</ol>
						
						<div class="relative rounded-md bg-gray-900 p-4 text-left">
							<pre class="overflow-x-auto text-xs text-gray-300"><code>{`{
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
					</div>
				</details>
			</div>
		</div>
	</SectionPanel>

	<SectionPanel title="VM Migration" description="Migrate your entire MyPaas installation to a new server.">
		{#if !migration || (migration.status === 'failed' && !preparingMigration)}
			<div class="space-y-6">
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Generate a migration package containing your complete MyPaas state — database, project volumes, configurations, and encrypted secrets. Transfer this package to your new VM to restore everything.
				</p>

				<div class="rounded-md border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-900/50 dark:bg-yellow-900/20">
					<div class="flex gap-3">
						<AlertTriangle class="h-5 w-5 shrink-0 text-yellow-600 dark:text-yellow-500" />
						<div class="text-sm text-yellow-800 dark:text-yellow-200">
							<strong>Warning:</strong> All running project containers will be temporarily stopped during export. They will restart automatically after the package is created.
						</div>
					</div>
				</div>

				<ActionButton variant="primary" on:click={startMigration}>
					Prepare Migration Package
				</ActionButton>
			</div>
		{:else if migration.status === 'preparing' || preparingMigration}
			<div class="flex flex-col items-center justify-center space-y-4 rounded-lg border border-gray-200 py-12 dark:border-gray-800">
				<div class="relative flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 dark:bg-neutral-900">
					<div class="absolute inset-0 animate-ping rounded-full bg-gray-400 opacity-20 dark:bg-gray-600"></div>
					<Loader2 class="h-8 w-8 animate-spin text-gray-950 dark:text-white" />
				</div>
				<p class="text-center text-sm font-medium text-gray-900 dark:text-gray-100">
					Creating migration package...
				</p>
				<p class="text-center text-sm text-gray-500 dark:text-gray-400">
					This may take a few minutes depending on data size.
				</p>
			</div>
		{:else if migration.status === 'ready'}
			<div class="space-y-8">
				<div class="rounded-md border border-green-200 bg-green-50 p-5 dark:border-green-900/30 dark:bg-green-900/10">
					<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
						<div>
							<h3 class="text-sm font-medium text-green-800 dark:text-green-400">Migration package ready</h3>
							<p class="mt-1 text-sm text-green-700 dark:text-green-500">
								This package expires in {formatHoursLeft(migration.expiresAt)} hours
							</p>
						</div>
						<a
							href={downloadUrl}
							class="inline-flex items-center justify-center gap-2 rounded-md border border-gray-300 bg-transparent px-4 py-2 text-sm font-medium text-gray-700 shadow-sm transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-950 focus:ring-offset-2 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800 dark:focus:ring-white dark:focus:ring-offset-gray-900"
						>
							<Download class="h-4 w-4" />
							Download Package (Manual Backup)
						</a>
					</div>
				</div>

				<div class="rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-950">
					<h3 class="text-lg font-medium text-gray-900 dark:text-white">One-Step Automated Migration</h3>
					<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
						SSH into your brand new VM and paste this single command. It will install MyPaas, securely download your migration package, restore your database & volumes, and automatically restart your projects.
					</p>
					
					<div class="group relative mt-6 rounded-md bg-gray-900 p-4 pr-12 text-sm text-gray-300">
						<code class="block overflow-x-auto whitespace-pre-wrap font-mono leading-relaxed">{migrationCommand}</code>
						<button
							on:click={() => copyToClipboard(migrationCommand, 'automated_cmd')}
							class="absolute right-3 top-3 rounded p-2 text-gray-400 hover:bg-gray-800 hover:text-white focus:outline-none"
							aria-label="Copy code"
						>
							{#if copiedText === 'automated_cmd'}
								<Check class="h-5 w-5 text-green-400" />
							{:else}
								<Copy class="h-5 w-5" />
							{/if}
						</button>
					</div>
				</div>
			</div>
		{/if}
	</SectionPanel>
</div>
