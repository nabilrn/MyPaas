<script lang="ts">
	import { Check, Copy, Eye, EyeOff, RefreshCw, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api } from '$api';
	import ActionButton from '$components/ActionButton.svelte';
	import IconButton from '$components/IconButton.svelte';
	import AgentBadgeStack from '$components/AgentBadgeStack.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import { toast } from '$stores/toast';

	let mcpToken = '';
	let loading = true;
	let showToken = false;
	let regeneratingToken = false;
	let confirmRegenerateToken = false;
	let copiedText = '';

	$: origin = typeof window !== 'undefined' ? window.location.origin : 'https://<your-domain>';
	$: setupPrompt = `Configure this agent to use the local MyPaaS MCP bridge.

1. Clone https://github.com/nabilrn/MyPaas on the machine running the agent.
2. Run backend/cmd/mcp/main.go with Go over stdio.
3. Set MYPAAS_URL=${origin}/api
4. Set MYPAAS_API_TOKEN=${mcpToken || '<your-token>'}
5. Verify the connection by listing MyPaaS projects.

Keep the token secret. Do not deploy, restart, delete, or change project configuration unless I explicitly ask.`;

	onMount(loadToken);

	async function loadToken() {
		loading = true;
		try {
			const data = await api.admin.getSettings();
			mcpToken = data.mcp_api_token ?? '';
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to load MCP access');
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
				if (copiedText === id) copiedText = '';
			}, 2000);
		} catch {
			toast.error('Failed to copy');
		}
	}

	async function regenerateToken() {
		if (regeneratingToken) return;
		regeneratingToken = true;
		try {
			const data = await api.admin.regenerateMCPToken();
			mcpToken = data.mcp_api_token ?? '';
			showToken = true;
			confirmRegenerateToken = false;
			toast.success('MCP token regenerated');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to regenerate MCP token');
		} finally {
			regeneratingToken = false;
		}
	}
</script>

<svelte:head>
	<title>MCP · MyPaas</title>
</svelte:head>

<div class="page-shell">
	{#if loading}
		<div class="flex min-h-48 items-center justify-center"><LoadingIndicator label="Loading MCP access" /></div>
	{:else}
		<section>
			<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Access</h2>
			<div class="mt-3 divide-y divide-gray-100 border-y border-gray-100 dark:divide-neutral-800 dark:border-neutral-800">
				<div class="grid gap-3 py-3 sm:grid-cols-[10rem_minmax(0,1fr)_auto] sm:items-center">
					<p class="text-sm text-gray-500 dark:text-gray-400">API token</p>
					<p class="min-w-0 break-all font-mono text-sm text-gray-950 dark:text-white">{mcpToken ? (showToken ? mcpToken : '••••••••••••••••') : 'Not configured'}</p>
					{#if mcpToken}
						<div class="flex items-center gap-1">
							<IconButton label={showToken ? 'Hide API token' : 'Reveal API token'} variant="ghost" on:click={() => (showToken = !showToken)}>{#if showToken}<EyeOff class="h-4 w-4" />{:else}<Eye class="h-4 w-4" />{/if}</IconButton>
							<IconButton label={copiedText === 'token' ? 'API token copied' : 'Copy API token'} variant="ghost" on:click={() => copyToClipboard(mcpToken, 'token')}>{#if copiedText === 'token'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
						</div>
					{/if}
				</div>
				<div class="grid gap-3 py-3 sm:grid-cols-[10rem_minmax(0,1fr)] sm:items-center">
					<p class="text-sm text-gray-500 dark:text-gray-400">Clients</p>
					<AgentBadgeStack />
				</div>
			</div>
			<div class="mt-3 flex flex-wrap items-center gap-2">
				{#if confirmRegenerateToken}
					<span class="text-sm text-gray-500 dark:text-gray-400">Existing clients will disconnect.</span>
					<ActionButton variant="ghost" size="sm" on:click={() => (confirmRegenerateToken = false)} disabled={regeneratingToken}><X slot="icon" class="h-4 w-4" />Cancel</ActionButton>
					<ActionButton variant="danger" size="sm" on:click={regenerateToken} loading={regeneratingToken} loadingLabel="Regenerating"><RefreshCw slot="icon" class="h-4 w-4" />Regenerate</ActionButton>
				{:else}
					<ActionButton variant="secondary" size="sm" on:click={() => (confirmRegenerateToken = true)}><RefreshCw slot="icon" class="h-4 w-4" />Regenerate token</ActionButton>
				{/if}
			</div>
		</section>

		<details class="border-y border-gray-100 py-3 dark:border-neutral-800">
			<summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">Agent setup</summary>
			<div class="mt-3">
				<pre class="console-surface max-h-96 overflow-auto whitespace-pre-wrap p-4"><code>{setupPrompt}</code></pre>
				<ActionButton variant="secondary" size="sm" className="mt-2" disabled={!mcpToken} on:click={() => copyToClipboard(setupPrompt, 'prompt')}>{#if copiedText === 'prompt'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}{copiedText === 'prompt' ? 'Copied' : 'Copy setup'}</ActionButton>
			</div>
		</details>
	{/if}
</div>
