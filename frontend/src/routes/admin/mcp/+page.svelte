<script lang="ts">
	import { onMount } from 'svelte';
	import { Activity, Bot, Check, Copy, GitBranch, KeyRound, LoaderCircle, RefreshCw, Rocket, ShieldCheck, SquareTerminal, X } from '@lucide/svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	let mcpToken = '';
	let loading = true;
	let regeneratingToken = false;
	let confirmRegenerateToken = false;
	let copiedText: string | null = null;

	$: origin = typeof window !== 'undefined' ? window.location.origin : 'https://<your-domain>';
	$: mcpConfig = `{
  "mcpServers": {
    "mypaas": {
      "command": "go",
      "args": [
        "run",
        "/absolute/path/to/MyPaas/backend/cmd/mcp/main.go"
      ],
      "env": {
        "MYPAAS_URL": "${origin}/api",
        "MYPAAS_API_TOKEN": "${mcpToken || '<your-token>'}"
      }
    }
  }
}`;

	onMount(loadToken);

	async function loadToken() {
		loading = true;
		try {
			const data = await api.admin.getSettings();
			mcpToken = data.mcp_api_token ?? '';
		} catch (error) {
			toast.error('Failed to load MCP configuration');
			console.error(error);
		} finally {
			loading = false;
		}
	}

	async function copyToClipboard(text: string, id: string) {
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

	async function regenerateToken() {
		if (regeneratingToken) return;
		regeneratingToken = true;
		try {
			const data = await api.admin.regenerateMCPToken();
			mcpToken = data.mcp_api_token ?? '';
			confirmRegenerateToken = false;
			toast.success('MCP token regenerated');
		} catch (error) {
			toast.error('Failed to regenerate MCP token');
			console.error(error);
		} finally {
			regeneratingToken = false;
		}
	}
</script>

<svelte:head>
	<title>MCP · MyPaas</title>
</svelte:head>

<div class="page-shell space-y-4 py-6">
	<p class="px-5 text-sm text-gray-500 dark:text-gray-400">Connect an MCP-compatible AI agent to this MyPaaS instance through the authenticated local bridge included in the repository.</p>

	<SectionPanel title="What MCP enables" description="The MyPaaS MCP server translates agent tool calls into authenticated MyPaaS API operations." contentClass="p-0">
		<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 lg:grid-cols-2 lg:divide-x lg:divide-y-0">
			<div class="divide-y divide-gray-100 dark:divide-neutral-800">
				<div class="flex gap-3 p-4">
					<GitBranch class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<div>
						<p class="text-sm font-medium text-gray-950 dark:text-white">Projects and repository inspection</p>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">List and inspect projects, inspect Git repositories, detect Compose configuration, create projects, and update mutable project settings.</p>
					</div>
				</div>
				<div class="flex gap-3 p-4">
					<Rocket class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<div>
						<p class="text-sm font-medium text-gray-950 dark:text-white">Deployments and runtime control</p>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Deploy, start, stop, and restart projects; inspect deployment history; and roll back to a successful deployment.</p>
					</div>
				</div>
			</div>
			<div class="divide-y divide-gray-100 dark:divide-neutral-800">
				<div class="flex gap-3 p-4">
					<Activity class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<div>
						<p class="text-sm font-medium text-gray-950 dark:text-white">Logs, metrics, and environment</p>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Read recent logs and metrics snapshots, list environment keys, set environment variables, and delete a variable with explicit key confirmation. Environment values are not revealed by the list operation.</p>
					</div>
				</div>
				<div class="flex gap-3 p-4">
					<ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
					<div>
						<p class="text-sm font-medium text-gray-950 dark:text-white">Capacity and admin context</p>
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Inspect quota usage and, with owner/admin access, host resource statistics. The bridge does not bypass the authorization enforced by the MyPaaS API.</p>
					</div>
				</div>
			</div>
		</div>
	</SectionPanel>

	<SectionPanel title="Access token" description="Credential used by the local MCP bridge to authenticate against this MyPaaS instance." contentClass="p-0">
		<div class="p-4">
			<div class="alert-neutral">
				<KeyRound class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<p>Treat this token like an administrative credential. Any MCP client that receives it can perform the MyPaaS operations allowed to that token.</p>
			</div>

			{#if loading}
				<div class="flex h-24 items-center justify-center">
					<LoaderCircle class="h-5 w-5 animate-spin motion-reduce:animate-none text-gray-500" aria-hidden="true" />
				</div>
			{:else}
				<label class="mt-4 block max-w-3xl" for="mcp_token">
					<span class="field-label">MCP API token</span>
					<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
						<input type="text" id="mcp_token" readonly value={mcpToken || 'Not configured'} class="field min-w-0 flex-1 bg-gray-50 font-mono text-sm text-gray-500 dark:bg-neutral-900 dark:text-gray-400" />
						{#if mcpToken}
							<ActionButton variant="secondary" size="sm" on:click={() => copyToClipboard(mcpToken, 'token')}>
								{#if copiedText === 'token'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}
								{copiedText === 'token' ? 'Copied' : 'Copy token'}
							</ActionButton>
						{/if}
					</div>
					<p class="field-hint">Use this value as <code class="font-mono">MYPAAS_API_TOKEN</code> in the MCP client configuration below.</p>
				</label>
			{/if}
		</div>

		<div class="border-t border-gray-100 p-4 dark:border-neutral-800">
			{#if confirmRegenerateToken}
				<div class="alert-warning flex-wrap items-center justify-between">
					<div class="min-w-0 flex-1">
						<p class="font-medium">Regenerate MCP token?</p>
						<p class="mt-0.5">Clients still using the old token will lose access immediately and must be reconfigured.</p>
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
				<ActionButton variant="secondary" size="sm" on:click={() => (confirmRegenerateToken = true)} disabled={loading}>
					<RefreshCw slot="icon" class="h-4 w-4" />
					Regenerate token
				</ActionButton>
			{/if}
		</div>
	</SectionPanel>

	<SectionPanel title="Connect an agent" description="Run the MCP bridge locally over stdio and point it at this remote MyPaaS API." contentClass="p-0">
		<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 lg:grid-cols-[20rem_minmax(0,1fr)] lg:divide-x lg:divide-y-0">
			<div class="p-4">
				<div class="flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white">
					<Bot class="h-4 w-4" aria-hidden="true" />
					Setup
				</div>
				<ol class="mt-3 list-decimal space-y-2 pl-5 text-sm text-gray-600 dark:text-gray-300">
					<li>Install Go on the machine where your MCP-compatible agent runs.</li>
					<li>Clone the MyPaaS repository locally.</li>
					<li>Set the absolute path to <code class="font-mono text-xs">backend/cmd/mcp/main.go</code> in your MCP client.</li>
					<li>Provide this instance URL and MCP token through the two environment variables shown.</li>
				</ol>
			</div>
			<div class="min-w-0 p-4">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<div class="flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white">
						<SquareTerminal class="h-4 w-4" aria-hidden="true" />
						MCP client configuration
					</div>
					<ActionButton variant="secondary" size="xs" on:click={() => copyToClipboard(mcpConfig, 'config')}>
						{#if copiedText === 'config'}<Check slot="icon" class="h-3.5 w-3.5" />{:else}<Copy slot="icon" class="h-3.5 w-3.5" />{/if}
						{copiedText === 'config' ? 'Copied' : 'Copy config'}
					</ActionButton>
				</div>
				<pre class="console-surface mt-3 overflow-x-auto p-4"><code>{mcpConfig}</code></pre>
			</div>
		</div>
	</SectionPanel>
</div>