<script lang="ts">
	import { Check, Copy, KeyRound, RefreshCw, X } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api } from '$api';
	import ActionButton from '$components/ActionButton.svelte';
	import AgentBadgeStack from '$components/AgentBadgeStack.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { toast } from '$stores/toast';

	let mcpToken = '';
	let loading = true;
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
	<SectionPanel title="MCP access" contentClass="p-0">
		<svelte:fragment slot="actions"><AgentBadgeStack /></svelte:fragment>
		<div class="p-4 lg:p-5">
			<div class="alert-neutral">
				<KeyRound class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<p>The token grants owner-level MyPaaS API access. Store it only in the local MCP client.</p>
			</div>

			{#if loading}
				<p class="mt-4 text-sm text-gray-500 dark:text-gray-400">Loading token…</p>
			{:else}
				<label class="mt-4 block max-w-3xl" for="mcp_token">
					<span class="field-label">API token</span>
					<div class="flex flex-col gap-2 sm:flex-row sm:items-center">
						<input type="password" id="mcp_token" readonly value={mcpToken || 'Not configured'} class="field min-w-0 flex-1 bg-gray-50 font-mono text-sm text-gray-500 dark:bg-neutral-900 dark:text-gray-400" />
						{#if mcpToken}
							<ActionButton variant="secondary" size="sm" on:click={() => copyToClipboard(mcpToken, 'token')}>
								{#if copiedText === 'token'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}
								{copiedText === 'token' ? 'Copied' : 'Copy token'}
							</ActionButton>
						{/if}
					</div>
				</label>
			{/if}
		</div>

		<div class="border-t border-gray-100/70 p-4 dark:border-neutral-900 lg:px-5">
			{#if confirmRegenerateToken}
				<div class="alert-warning flex-wrap items-center justify-between">
					<p class="min-w-0 flex-1">Regenerating disconnects clients using the current token.</p>
					<div class="flex gap-2">
						<ActionButton variant="ghost" size="xs" on:click={() => (confirmRegenerateToken = false)} disabled={regeneratingToken}><X slot="icon" class="h-3.5 w-3.5" />Cancel</ActionButton>
						<ActionButton variant="danger" size="xs" on:click={regenerateToken} loading={regeneratingToken} loadingLabel="Regenerating"><RefreshCw slot="icon" class="h-3.5 w-3.5" />Regenerate</ActionButton>
					</div>
				</div>
			{:else}
				<ActionButton variant="secondary" size="sm" on:click={() => (confirmRegenerateToken = true)} disabled={loading}><RefreshCw slot="icon" class="h-4 w-4" />Regenerate token</ActionButton>
			{/if}
		</div>
	</SectionPanel>

	<SectionPanel title="Agent setup prompt" contentClass="p-0">
		<div class="p-4 lg:p-5">
			<pre class="console-surface max-h-96 overflow-auto whitespace-pre-wrap p-4"><code>{setupPrompt}</code></pre>
			<ActionButton variant="primary" size="sm" className="mt-3" disabled={!mcpToken} on:click={() => copyToClipboard(setupPrompt, 'prompt')}>
				{#if copiedText === 'prompt'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}
				{copiedText === 'prompt' ? 'Copied' : 'Copy setup prompt'}
			</ActionButton>
		</div>
	</SectionPanel>
</div>
