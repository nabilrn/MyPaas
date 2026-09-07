<script lang="ts">
	import { Activity, Boxes, Check, Copy, KeyRound, PlugZap, RefreshCw, Rocket, ShieldCheck, Trash2, Variable } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api } from '$api';
	import { remoteMCPApi, type MCPAPIToken } from '$lib/api/mcp';
	import ActionButton from '$components/ActionButton.svelte';
	import AgentClientGrid from '$components/AgentClientGrid.svelte';
	import ConfirmActionDialog from '$components/ConfirmActionDialog.svelte';
	import IconButton from '$components/IconButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import { toast } from '$stores/toast';

	let loading = true;
	let creating = false;
	let tokens: MCPAPIToken[] = [];
	let allowedScopes: string[] = [];
	let selectedScopes: string[] = [];
	let tokenName = 'MCP agent';
	let expiry = 'never';
	let createdToken = '';
	let copiedText = '';
	let revokingId = '';
	let tokenToRevoke: MCPAPIToken | null = null;

	let legacyToken = '';
	let regeneratingLegacy = false;
	let confirmRegenerateLegacy = false;

	const capabilities = [
		{ label: 'Projects', detail: 'List, inspect, create, and update projects.', icon: Boxes },
		{ label: 'Deployments', detail: 'Deploy, start, stop, restart, and roll back.', icon: Rocket },
		{ label: 'Observability', detail: 'Read deployment history, logs, metrics, quota, and optional host stats.', icon: Activity },
		{ label: 'Environment', detail: 'List, set, and delete environment variables without secret reveal.', icon: Variable }
	] as const;

	$: origin = typeof window !== 'undefined' ? window.location.origin : 'https://<your-domain>';
	$: mcpEndpoint = `${origin}/mcp`;
	$: remoteConfig = `{
  "mcpServers": {
    "mypaas": {
      "url": "${mcpEndpoint}",
      "headers": {
        "Authorization": "Bearer ${createdToken || '<myp-token>'}"
      }
    }
  }
}`;
	$: legacyTarget = `${origin}/api`;
	$: legacyPrompt = `MYPAAS_URL=${legacyTarget}\nMYPAAS_API_TOKEN=${legacyToken || '<legacy-token>'}\ngo run ./backend/cmd/mcp`;

	onMount(loadData);

	async function loadData() {
		loading = true;
		try {
			const [remote, settings] = await Promise.all([remoteMCPApi.listTokens(), api.admin.getSettings()]);
			tokens = remote.tokens;
			allowedScopes = remote.allowedScopes;
			selectedScopes = [...remote.defaultScopes];
			legacyToken = settings.mcp_api_token ?? '';
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to load MCP settings');
		} finally {
			loading = false;
		}
	}

	function toggleScope(scope: string) {
		selectedScopes = selectedScopes.includes(scope)
			? selectedScopes.filter((item) => item !== scope)
			: [...selectedScopes, scope];
	}

	function expiresAt(): string | undefined {
		if (expiry === '30d') return new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString();
		if (expiry === '90d') return new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString();
		return undefined;
	}

	async function createToken() {
		if (creating || selectedScopes.length === 0) return;
		creating = true;
		try {
			const created = await remoteMCPApi.createToken({
				name: tokenName.trim() || 'MCP agent',
				scopes: selectedScopes,
				expiresAt: expiresAt()
			});
			createdToken = created.token;
			tokens = [created, ...tokens];
			toast.success('MCP token created. Copy it now; it will not be shown again.');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to create MCP token');
		} finally {
			creating = false;
		}
	}

	async function revokeSelectedToken() {
		if (!tokenToRevoke || revokingId) return;
		const token = tokenToRevoke;
		revokingId = token.id;
		try {
			await remoteMCPApi.revokeToken(token.id);
			const now = new Date().toISOString();
			tokens = tokens.map((item) => (item.id === token.id ? { ...item, revokedAt: now } : item));
			tokenToRevoke = null;
			toast.success('MCP token revoked');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to revoke MCP token');
		} finally {
			revokingId = '';
		}
	}

	async function regenerateLegacyToken() {
		if (regeneratingLegacy) return;
		regeneratingLegacy = true;
		try {
			const data = await api.admin.regenerateMCPToken();
			legacyToken = data.mcp_api_token ?? '';
			confirmRegenerateLegacy = false;
			toast.success('Legacy STDIO token regenerated');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to regenerate legacy token');
		} finally {
			regeneratingLegacy = false;
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
			}, 1800);
		} catch {
			toast.error('Failed to copy');
		}
	}

	function formatTime(value?: string) {
		if (!value) return 'Never';
		return new Date(value).toLocaleString();
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
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Remote MCP</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Connect an agent directly to this MyPaaS instance over HTTPS.</p>
				</div>
				<div class="grid border-t border-[color:var(--workspace-divider)] xl:grid-cols-2">
					<div class="min-w-0 px-4 py-3 xl:border-r xl:border-[color:var(--workspace-divider)]">
						<div class="flex items-start gap-3">
							<span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-gray-200 text-gray-600 dark:border-neutral-700 dark:text-gray-300"><PlugZap class="h-4 w-4" /></span>
							<div class="min-w-0 flex-1">
								<p class="text-xs text-gray-500 dark:text-gray-400">Endpoint</p>
								<p class="mt-1 break-all font-mono text-sm font-medium text-gray-950 dark:text-white">{mcpEndpoint}</p>
								<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Derived from this installation domain; no MyPaaS public domain is hardcoded.</p>
							</div>
							<IconButton label="Copy MCP endpoint" variant="ghost" on:click={() => copyToClipboard(mcpEndpoint, 'endpoint')}>{#if copiedText === 'endpoint'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
						</div>
					</div>
					<div class="min-w-0 border-t border-[color:var(--workspace-divider)] px-4 py-3 xl:border-t-0">
						<div class="flex items-start gap-3">
							<span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-gray-200 text-gray-600 dark:border-neutral-700 dark:text-gray-300"><ShieldCheck class="h-4 w-4" /></span>
							<div>
								<p class="text-sm font-medium text-gray-950 dark:text-white">Scoped bearer authentication</p>
								<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">Keys are stored as hashes, can expire or be revoked, and cannot access shell, DB Studio, secret reveal, raw SQL, project deletion, routing, backup, or system update.</p>
							</div>
						</div>
					</div>
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Create access key</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">The full token is shown only once after creation.</p>
				</div>
				<div class="grid gap-4 border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:grid-cols-[16rem_12rem_minmax(0,1fr)]">
					<label class="text-xs text-gray-500 dark:text-gray-400">Name<input bind:value={tokenName} maxlength="100" class="mt-1 w-full rounded-md border border-gray-200 bg-transparent px-2.5 py-2 text-sm text-gray-950 outline-none focus:border-gray-400 dark:border-neutral-700 dark:text-white" /></label>
					<label class="text-xs text-gray-500 dark:text-gray-400">Expiration<select bind:value={expiry} class="mt-1 w-full rounded-md border border-gray-200 bg-transparent px-2.5 py-2 text-sm text-gray-950 outline-none dark:border-neutral-700 dark:text-white"><option value="never">Never</option><option value="30d">30 days</option><option value="90d">90 days</option></select></label>
					<div>
						<p class="text-xs text-gray-500 dark:text-gray-400">Scopes</p>
						<div class="mt-1.5 flex flex-wrap gap-2">
							{#each allowedScopes as scope}
								<label class="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-gray-200 px-2 py-1.5 text-xs text-gray-700 dark:border-neutral-700 dark:text-gray-300"><input type="checkbox" checked={selectedScopes.includes(scope)} on:change={() => toggleScope(scope)} />{scope}</label>
							{/each}
						</div>
					</div>
				</div>
				<div class="flex items-center justify-between gap-3 border-t border-[color:var(--workspace-divider)] px-4 py-2.5">
					<p class="text-xs text-gray-500 dark:text-gray-400">Default scopes cover project/deployment/env operations and observability; <code>admin:read</code> is opt-in.</p>
					<ActionButton size="sm" disabled={creating || selectedScopes.length === 0} on:click={createToken}><KeyRound slot="icon" class="h-4 w-4" />{creating ? 'Creating' : 'Create key'}</ActionButton>
				</div>
				{#if createdToken}
					<div class="border-t border-[color:var(--workspace-divider)] bg-gray-50/60 px-4 py-3 dark:bg-neutral-900/30">
						<p class="text-xs font-medium text-gray-950 dark:text-white">Copy this token now</p>
						<div class="mt-1.5 flex items-start gap-2"><code class="min-w-0 flex-1 break-all rounded-md border border-gray-200 bg-white px-2.5 py-2 text-xs text-gray-900 dark:border-neutral-700 dark:bg-neutral-950 dark:text-gray-100">{createdToken}</code><IconButton label="Copy new token" variant="ghost" on:click={() => copyToClipboard(createdToken, 'new-token')}>{#if copiedText === 'new-token'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton></div>
						<p class="mt-1.5 text-xs text-amber-700 dark:text-amber-400">It cannot be recovered after this page state is lost. Create a replacement if needed.</p>
					</div>
				{/if}
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5"><h2 class="text-sm font-semibold text-gray-950 dark:text-white">Access keys</h2><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Prefixes identify keys without exposing their secrets.</p></div>
				<div class="overflow-x-auto border-t border-[color:var(--workspace-divider)]">
					<table class="w-full min-w-[760px] text-left text-xs">
						<thead class="text-gray-500 dark:text-gray-400"><tr><th class="px-4 py-2 font-medium">Name</th><th class="px-4 py-2 font-medium">Prefix</th><th class="px-4 py-2 font-medium">Scopes</th><th class="px-4 py-2 font-medium">Last used</th><th class="px-4 py-2 font-medium">Expires</th><th class="px-4 py-2 text-right font-medium">State</th></tr></thead>
						<tbody class="divide-y divide-[color:var(--workspace-divider)]">
							{#each tokens as token}
								<tr class="text-gray-700 dark:text-gray-300"><td class="px-4 py-2.5 font-medium text-gray-950 dark:text-white">{token.name}</td><td class="px-4 py-2.5 font-mono">{token.prefix}…</td><td class="max-w-sm px-4 py-2.5">{token.scopes.join(', ')}</td><td class="px-4 py-2.5">{token.lastUsedAt ? formatTime(token.lastUsedAt) : 'Never'}</td><td class="px-4 py-2.5">{formatTime(token.expiresAt)}</td><td class="px-4 py-2.5 text-right">{#if token.revokedAt}<span class="text-gray-400">Revoked</span>{:else}<button class="inline-flex items-center gap-1 text-red-600 hover:underline disabled:opacity-50 dark:text-red-400" disabled={revokingId === token.id} on:click={() => (tokenToRevoke = token)}><Trash2 class="h-3.5 w-3.5" />Revoke</button>{/if}</td></tr>
							{:else}
								<tr><td colspan="6" class="px-4 py-6 text-center text-gray-500 dark:text-gray-400">No scoped MCP keys yet.</td></tr>
							{/each}
						</tbody>
					</table>
				</div>
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Agent friendly</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Bring your preferred MCP-compatible coding agent to the same scoped remote endpoint.</p>
				</div>
				<AgentClientGrid />
			</section>

			<section class="border-b border-[color:var(--workspace-divider)]">
				<div class="px-4 py-2.5">
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Agent capabilities</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">The remote surface reuses the existing MyPaaS actions while REST scopes remain authoritative.</p>
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
				<div class="px-4 py-2.5"><h2 class="text-sm font-semibold text-gray-950 dark:text-white">Connect an agent</h2><p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Use the installation endpoint and one scoped token in any Streamable HTTP compatible MCP client.</p></div>
				<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3"><pre class="console-surface max-h-80 overflow-auto whitespace-pre-wrap p-3"><code>{remoteConfig}</code></pre><ActionButton variant="secondary" size="sm" className="mt-2" on:click={() => copyToClipboard(remoteConfig, 'config')}>{#if copiedText === 'config'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}{copiedText === 'config' ? 'Copied' : 'Copy config'}</ActionButton></div>
			</section>

			<details class="border-b border-[color:var(--workspace-divider)]">
				<summary class="cursor-pointer px-4 py-2.5 text-sm font-semibold text-gray-950 dark:text-white">Local STDIO compatibility</summary>
				<div class="grid border-t border-[color:var(--workspace-divider)] lg:grid-cols-[minmax(0,1fr)_auto]">
					<div class="min-w-0 px-4 py-3"><p class="text-xs text-gray-500 dark:text-gray-400">The existing local bridge remains supported for clients that spawn MCP over STDIO.</p><pre class="console-surface mt-2 overflow-auto whitespace-pre-wrap p-3"><code>{legacyPrompt}</code></pre></div>
					<div class="flex items-start gap-2 border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:border-l lg:border-t-0"><ActionButton variant="secondary" size="sm" disabled={!legacyToken} on:click={() => copyToClipboard(legacyPrompt, 'legacy')}>{#if copiedText === 'legacy'}<Check slot="icon" class="h-4 w-4" />{:else}<Copy slot="icon" class="h-4 w-4" />{/if}Copy setup</ActionButton><ActionButton variant="secondary" size="sm" disabled={regeneratingLegacy} on:click={() => (confirmRegenerateLegacy = true)}><RefreshCw slot="icon" class="h-4 w-4" />Regenerate legacy key</ActionButton></div>
				</div>
			</details>
		</div>
	{/if}
</div>

<ConfirmActionDialog
	open={tokenToRevoke !== null}
	title="Revoke MCP token?"
	description="The connected agent using this key will stop authenticating immediately."
	confirmLabel="Revoke token"
	busyLabel="Revoking"
	variant="danger"
	busy={Boolean(tokenToRevoke && revokingId === tokenToRevoke.id)}
	on:cancel={() => (tokenToRevoke = null)}
	on:confirm={revokeSelectedToken}
>
	<p>{tokenToRevoke ? `${tokenToRevoke.name} (${tokenToRevoke.prefix}…)` : 'This token'} cannot be restored after revocation.</p>
</ConfirmActionDialog>

<ConfirmActionDialog
	open={confirmRegenerateLegacy}
	title="Regenerate legacy STDIO token?"
	description="Existing local STDIO clients using the legacy token will stop authenticating until they are updated."
	confirmLabel="Regenerate token"
	busyLabel="Regenerating"
	variant="danger"
	busy={regeneratingLegacy}
	on:cancel={() => (confirmRegenerateLegacy = false)}
	on:confirm={regenerateLegacyToken}
>
	<p>The scoped remote MCP keys are not affected.</p>
</ConfirmActionDialog>
