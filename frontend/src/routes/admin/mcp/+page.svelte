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

	const capabilities = [
		{ label: 'Projects', detail: 'List, inspect, create, and update projects.' },
		{ label: 'Deployments', detail: 'Deploy, start, stop, restart, and roll back.' },
		{ label: 'Observability', detail: 'Read deployment history, logs, metrics, quota, and host stats.' },
		{ label: 'Environment', detail: 'List, set, and delete environment variables.' }
	] as const;

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
			toast.error(error instanceof Error ? error.message : 'Failed to load MCP');
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
		<div class="flex min-h-48 items-center justify-center"><LoadingIndicator label="Loading MCP" /></div>
	{:else}
		<div class="max-w-5xl">
			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5"><h2 class="text-sm font-semibold text-gray-950 dark:text-white">Access</h2></div>
				<div class="border-t border-[color:var(--workspace-divider)]">
					<div class="grid gap-3 px-4 py-3 sm:grid-cols-[9rem_minmax(0,1fr)_auto] sm:items-center">
						<p class="text-sm text-gray-500 dark:text-gray-400">API token</p>
						<div class="min-w-0">
							<p class="break-all font-mono text-sm text-gray-950 dark:text-white">{mcpToken ? (showToken ? mcpToken : '••••••••••••••••') : 'Not configured'}</p>
							{#if confirmRegenerateToken}<p class="mt-0.5 text-xs text-amber-700 dark:text-amber-300">Existing clients will need the new token.</p>{/if}
						</div>
						<div class="flex flex-wrap items-center justify-end gap-1">
							{#if mcpToken}
								<IconButton label={showToken ? 'Hide API token' : 'Reveal API token'} variant="ghost" on:click={() => (showToken = !showToken)}>{#if showToken}<EyeOff class="h-4 w-4" />{:else}<Eye class="h-4 w-4" />{/if}</IconButton>
								<IconButton label={copiedText === 'token' ? 'API token copied' : 'Copy API token'} variant="ghost" on:click={() => copyToClipboard(mcpToken, 'token')}>{#if copiedText === 'token'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
							{/if}
							{#if confirmRegenerateToken}
								<ActionButton variant="ghost" size="sm" on:click={() => (confirmRegenerateToken = false)} disabled={regeneratingToken}><X slot="icon" class="h-4 w-4" />Cancel</ActionButton>
								<ActionButton variant="danger" size="sm" on:click={regenerateToken} loading={regeneratingToken} loadingLabel="Regenerating"><RefreshCw slot="icon" class="h-4 w-4" />Confirm</ActionButton>
							{:else}
								<ActionButton variant="secondary" size="sm" on:click={() => (confirmRegenerateToken = true)}><RefreshCw slot="icon" class="h-4 w-4" />Regenerate</ActionButton>
							{/if}
						</div>
					</div>
					<div class="grid gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:grid-cols-[9rem_minmax(0,1fr)] sm:items-center">
						<p class="text-sm text-gray-500 dark:text-gray-400">Supported clients</p>
						<AgentBadgeStack />
					</div>
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Agent capabilities</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Actions available through the MyPaaS MCP bridge.</p>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] sm:grid-cols-2">
					{#each capabilities as capability, index}
						<div class={`px-4 py-3 ${index > 0 ? 'border-t border-[color:var(--workspace-divider)] sm:border-t-0' : ''} ${index % 2 === 1 ? 'sm:border-l sm:border-[color:var(--workspace-divider)]' : ''} ${index >= 2 ? 'sm:border-t sm:border-[color:var(--workspace-divider)]' : ''}`}>
							<p class="text-sm font-medium text-gray-950 dark:text-white">{capability.label}</p>
							<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{capability.detail}</p>
						</div>
					{/each}
				</div>
			</section>

			<details class="border-b border-[color:var(--workspace-divider)] px-4 py-3">
				<summary class="app-focus cursor-pointer select-none text-sm font-medium text-gray-700 dark:text-gray-300">Agent setup</summary>
				<div class="mt-3">
					<pre class="console-surface max-h-80 overflow-auto whitespace-pre-wrap p-3"><code>{setupPrompt}</code></pre>
					<ActionButton variant="secondary" size="sm" className="mt-2" disabled={!mcpToken} on:click={() => copyToClipboard(setupPrompt, 'prompt')}>{#if copiedText === 'prompt'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}{copiedText === 'prompt' ? 'Copied' : 'Copy setup'}</ActionButton>
				</div>
			</details>
		</div>
	{/if}
</div>
