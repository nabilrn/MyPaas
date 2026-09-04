<script lang="ts">
	import { Activity, Boxes, Check, Copy, Eye, EyeOff, KeyRound, PlugZap, RefreshCw, Rocket, Variable } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api } from '$api';
	import ActionButton from '$components/ActionButton.svelte';
	import IconButton from '$components/IconButton.svelte';
	import AgentClientGrid from '$components/AgentClientGrid.svelte';
	import ConfirmActionDialog from '$components/ConfirmActionDialog.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import { toast } from '$stores/toast';

	let mcpToken = '';
	let loading = true;
	let showToken = false;
	let regeneratingToken = false;
	let confirmRegenerateToken = false;
	let copiedText = '';

	const capabilities = [
		{ label: 'Projects', detail: 'List, inspect, create, and update projects.', icon: Boxes },
		{ label: 'Deployments', detail: 'Deploy, start, stop, restart, and roll back.', icon: Rocket },
		{ label: 'Observability', detail: 'Read deployment history, logs, metrics, quota, and host stats.', icon: Activity },
		{ label: 'Environment', detail: 'List, set, and delete environment variables.', icon: Variable }
	] as const;

	$: origin = typeof window !== 'undefined' ? window.location.origin : 'https://<your-domain>';
	$: apiTarget = `${origin}/api`;
	$: setupPrompt = `Configure this agent to use the local MyPaaS MCP bridge.\n\n1. Clone https://github.com/nabilrn/MyPaas on the machine running the agent.\n2. Run backend/cmd/mcp/main.go with Go over stdio.\n3. Set MYPAAS_URL=${apiTarget}\n4. Set MYPAAS_API_TOKEN=${mcpToken || '<your-token>'}\n5. Verify the connection by listing MyPaaS projects.\n\nKeep the token secret. Do not deploy, restart, delete, or change project configuration unless I explicitly ask.`;

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
		if (!text) return;
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
		<div class="admin-mcp-workspace w-full">
			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Access</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Credentials and bridge target used by connected agents.</p>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] xl:grid-cols-2">
					<div class="min-w-0 px-4 py-3 xl:border-r xl:border-[color:var(--workspace-divider)]">
						<div class="flex items-start justify-between gap-3">
							<div class="flex min-w-0 items-start gap-3">
								<span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-gray-200 text-gray-600 dark:border-neutral-700 dark:text-gray-300"><KeyRound class="h-4.5 w-4.5" aria-hidden="true" /></span>
								<div class="min-w-0">
									<p class="text-xs text-gray-500 dark:text-gray-400">API token</p>
									<p class="mt-1 break-all font-mono text-sm font-medium text-gray-950 dark:text-white">{mcpToken ? (showToken ? mcpToken : '••••••••••••••••••••••••') : 'Not configured'}</p>
								</div>
							</div>
							<div class="flex shrink-0 items-center gap-1">
								{#if mcpToken}
									<IconButton label={showToken ? 'Hide API token' : 'Reveal API token'} variant="ghost" on:click={() => (showToken = !showToken)}>{#if showToken}<EyeOff class="h-4 w-4" />{:else}<Eye class="h-4 w-4" />{/if}</IconButton>
									<IconButton label={copiedText === 'token' ? 'API token copied' : 'Copy API token'} variant="ghost" on:click={() => copyToClipboard(mcpToken, 'token')}>{#if copiedText === 'token'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
								{/if}
								<ActionButton variant="secondary" size="sm" on:click={() => (confirmRegenerateToken = true)}><RefreshCw slot="icon" class="h-4 w-4" />Regenerate</ActionButton>
							</div>
						</div>
					</div>

					<div class="min-w-0 border-t border-[color:var(--workspace-divider)] px-4 py-3 xl:border-t-0">
						<div class="flex items-start gap-3">
							<span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-gray-200 text-gray-600 dark:border-neutral-700 dark:text-gray-300"><PlugZap class="h-4.5 w-4.5" aria-hidden="true" /></span>
							<div class="min-w-0">
								<p class="text-xs text-gray-500 dark:text-gray-400">Bridge target</p>
								<p class="mt-1 break-all font-mono text-sm font-medium text-gray-950 dark:text-white">{apiTarget}</p>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Local stdio bridge authenticates to this MyPaaS API.</p>
							</div>
						</div>
					</div>
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Supported clients</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Popular coding agents that can use an MCP-compatible local bridge.</p>
				</div>
				<AgentClientGrid />
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Agent capabilities</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Actions exposed by the current MyPaaS MCP tool surface.</p>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] md:grid-cols-2 xl:grid-cols-4">
					{#each capabilities as capability, index}
						<div class={`min-w-0 px-4 py-3 ${index > 0 ? 'border-t border-[color:var(--workspace-divider)] md:border-t-0' : ''} ${index % 2 === 1 ? 'md:border-l md:border-[color:var(--workspace-divider)]' : ''} ${index >= 2 ? 'md:border-t md:border-[color:var(--workspace-divider)] xl:border-t-0' : ''} ${index > 0 ? 'xl:border-l xl:border-[color:var(--workspace-divider)]' : ''}`}>
							<div class="flex items-start gap-3">
								<span class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-gray-200 text-gray-600 dark:border-neutral-700 dark:text-gray-300"><svelte:component this={capability.icon} class="h-4 w-4" aria-hidden="true" /></span>
								<div class="min-w-0"><p class="text-sm font-medium text-gray-950 dark:text-white">{capability.label}</p><p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{capability.detail}</p></div>
							</div>
						</div>
					{/each}
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Connect a client</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Run the bridge on the same machine as your coding agent.</p>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] lg:grid-cols-[18rem_minmax(0,1fr)]">
					<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
						<ol class="space-y-3 text-sm text-gray-600 dark:text-gray-300">
							<li><span class="font-mono text-xs text-gray-400 dark:text-gray-500">01</span><p class="mt-0.5 font-medium text-gray-950 dark:text-white">Clone MyPaaS</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">The MCP bridge lives in the repository.</p></li>
							<li><span class="font-mono text-xs text-gray-400 dark:text-gray-500">02</span><p class="mt-0.5 font-medium text-gray-950 dark:text-white">Start the stdio bridge</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Provide the API target and token as environment variables.</p></li>
							<li><span class="font-mono text-xs text-gray-400 dark:text-gray-500">03</span><p class="mt-0.5 font-medium text-gray-950 dark:text-white">Verify with a read action</p><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">List projects before allowing write operations.</p></li>
						</ol>
					</div>
					<div class="min-w-0 border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:border-t-0">
						<pre class="console-surface max-h-80 overflow-auto whitespace-pre-wrap p-3"><code>{setupPrompt}</code></pre>
						<ActionButton variant="secondary" size="sm" className="mt-2" disabled={!mcpToken} on:click={() => copyToClipboard(setupPrompt, 'prompt')}>{#if copiedText === 'prompt'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}{copiedText === 'prompt' ? 'Copied' : 'Copy setup'}</ActionButton>
					</div>
				</div>
			</section>
		</div>
	{/if}
</div>

<ConfirmActionDialog
	open={confirmRegenerateToken}
	title="Regenerate MCP token?"
	description="Existing connected clients will stop authenticating until they are updated with the new token."
	confirmLabel="Regenerate token"
	busyLabel="Regenerating"
	variant="danger"
	busy={regeneratingToken}
	on:cancel={() => (confirmRegenerateToken = false)}
	on:confirm={regenerateToken}
>
	<p>After regeneration, copy the new token into every client that uses this bridge.</p>
</ConfirmActionDialog>
