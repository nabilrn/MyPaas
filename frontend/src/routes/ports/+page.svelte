<script lang="ts">
	import { Plus, RefreshCw, Trash2 } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import TableShell from '$components/TableShell.svelte';
	import { api, type PortOverview } from '$api';
	import { toast } from '$stores/toast';

	const protectedPorts = new Set([22, 80, 443]);
	let overview: PortOverview | null = null;
	let loading = true;
	let refreshing = false;
	let error = '';
	let portValue = '';
	let protocol: 'tcp' | 'udp' = 'tcp';
	let saving = false;
	let deleting = '';

	$: parsedPort = Number(portValue);
	$: validPort = Number.isInteger(parsedPort) && parsedPort >= 1 && parsedPort <= 65535 && !protectedPorts.has(parsedPort);
	$: localBinding = !overview || ['127.0.0.1', 'localhost', '::1'].includes(overview.bindHost);

	onMount(() => void load());

	async function load() {
		if (refreshing) return;
		refreshing = true;
		error = '';
		try {
			overview = await api.admin.ports();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load ports';
		} finally {
			loading = false;
			refreshing = false;
		}
	}

	async function openPort() {
		if (!validPort || saving || !overview?.firewall.available) return;
		saving = true;
		try {
			await api.admin.openFirewallPort(parsedPort, protocol);
			toast.success(`Opened managed rule ${parsedPort}/${protocol}`);
			portValue = '';
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to open firewall port');
		} finally {
			saving = false;
		}
	}

	async function closePort(port: number, ruleProtocol: 'tcp' | 'udp') {
		const key = `${port}/${ruleProtocol}`;
		if (deleting) return;
		deleting = key;
		try {
			await api.admin.closeFirewallPort(port, ruleProtocol);
			toast.success(`Removed managed rule ${key}`);
			await load();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove firewall rule');
		} finally {
			deleting = '';
		}
	}
</script>

<svelte:head>
	<title>Ports · MyPaas</title>
</svelte:head>

<div class="page-shell py-6">
	<div class="mb-5 flex flex-wrap items-center justify-between gap-3 px-5">
		<div>
			<p class="text-sm text-gray-500 dark:text-gray-400">Inspect MyPaaS runtime bindings and manage only firewall rules created by MyPaaS.</p>
			<p class="mt-1 text-xs text-gray-400 dark:text-gray-500">SSH 22 and Caddy 80/443 are protected. MyPaaS never enables or disables UFW from this page.</p>
		</div>
		<ActionButton variant="secondary" size="sm" loading={refreshing} loadingLabel="Refreshing" on:click={load}>
			<RefreshCw slot="icon" class="h-4 w-4" />
			Refresh
		</ActionButton>
	</div>

	<div class="space-y-5">
		<SectionPanel title="System ports" description="Critical host ports are visible here but intentionally locked." padded={false}>
			<div class="overflow-x-auto">
				<table class="data-table">
					<thead><tr><th>Port</th><th>Purpose</th><th>Control</th></tr></thead>
					<tbody>
						<tr><td class="font-mono text-xs">22/tcp</td><td>SSH access</td><td><span class="text-xs text-gray-500 dark:text-gray-400">Locked</span></td></tr>
						<tr><td class="font-mono text-xs">80/tcp</td><td>Caddy HTTP</td><td><span class="text-xs text-gray-500 dark:text-gray-400">Locked</span></td></tr>
						<tr><td class="font-mono text-xs">443/tcp</td><td>Caddy HTTPS / edge traffic</td><td><span class="text-xs text-gray-500 dark:text-gray-400">Locked</span></td></tr>
					</tbody>
				</table>
			</div>
		</SectionPanel>

		<TableShell
			title="Project bindings"
			description={localBinding ? 'Application host ports are bound locally and public HTTP traffic remains routed through Caddy.' : 'This host is configured with a non-local Docker bind address; review exposure directly on the VM.'}
			{loading}
			{error}
			empty={(overview?.allocations.length ?? 0) === 0}
			emptyTitle="No allocated project ports."
			emptyDescription="A runtime port appears after a container-backed project is deployed."
			on:retry={load}
		>
			<table class="data-table">
				<thead><tr><th>Host binding</th><th>Project</th><th>Service</th><th>App port</th><th>Runtime</th><th>Exposure</th></tr></thead>
				<tbody>
					{#each overview?.allocations ?? [] as item}
						<tr>
							<td class="font-mono text-xs">{overview?.bindHost}:{item.port}</td>
							<td><a class="font-medium text-gray-950 hover:underline dark:text-white" href={`/projects/${item.projectId}`}>{item.projectName}</a></td>
							<td class="font-mono text-xs">{item.service}</td>
							<td class="font-mono text-xs">{item.appPort}</td>
							<td class="text-sm capitalize">{item.deployMode}</td>
							<td class="text-xs text-gray-500 dark:text-gray-400">{localBinding ? 'Local only · Caddy' : 'Host bound'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</TableShell>

		<SectionPanel title="Managed firewall rules" description="Only UFW allow rules tagged as MyPaaS-managed can be removed from this UI.">
			{#if error && !overview}
				<p class="text-sm text-red-600 dark:text-red-300">{error}</p>
			{:else if overview && !overview.firewall.available}
				<div class="rounded-md border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-neutral-800 dark:bg-neutral-900 dark:text-gray-300">
					Firewall helper unavailable. Project bindings remain visible, but host firewall changes are disabled.
				</div>
			{:else if overview}
				<div class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-md border border-gray-200 bg-gray-50 px-4 py-3 dark:border-neutral-800 dark:bg-neutral-900">
					<div>
						<p class="text-sm font-medium text-gray-950 dark:text-white">UFW {overview.firewall.active ? 'active' : 'inactive'}</p>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{overview.firewall.active ? 'Managed rules are enforced by the host firewall.' : 'Rules can be prepared, but MyPaaS will not activate UFW automatically.'}</p>
					</div>
				</div>

				<form class="mb-5 grid gap-3 sm:grid-cols-[minmax(0,1fr)_10rem_auto] sm:items-end" on:submit|preventDefault={openPort}>
					<div>
						<label class="field-label" for="firewall-port">Port</label>
						<input id="firewall-port" class="field w-full" type="number" min="1" max="65535" placeholder="8080" bind:value={portValue} />
						{#if portValue && protectedPorts.has(parsedPort)}<p class="field-hint">Port {parsedPort} is protected by MyPaaS.</p>{/if}
					</div>
					<div>
						<label class="field-label" for="firewall-protocol">Protocol</label>
						<select id="firewall-protocol" class="field w-full" bind:value={protocol}><option value="tcp">TCP</option><option value="udp">UDP</option></select>
					</div>
					<ActionButton type="submit" variant="primary" disabled={!validPort} loading={saving} loadingLabel="Opening">
						<Plus slot="icon" class="h-4 w-4" />Open port
					</ActionButton>
				</form>

				{#if overview.firewall.rules.length === 0}
					<p class="py-3 text-sm text-gray-500 dark:text-gray-400">No MyPaaS-managed firewall rules.</p>
				{:else}
					<div class="overflow-x-auto">
						<table class="data-table">
							<thead><tr><th>Port</th><th>Protocol</th><th>Owner</th><th class="text-right">Action</th></tr></thead>
							<tbody>
								{#each overview.firewall.rules as rule}
									<tr>
										<td class="font-mono text-xs">{rule.port}</td>
										<td class="uppercase text-xs">{rule.protocol}</td>
										<td class="text-xs text-gray-500 dark:text-gray-400">MyPaaS managed</td>
										<td class="text-right"><ActionButton variant="ghostDanger" size="xs" loading={deleting === `${rule.port}/${rule.protocol}`} loadingLabel="Closing" disabled={Boolean(deleting) && deleting !== `${rule.port}/${rule.protocol}`} on:click={() => closePort(rule.port, rule.protocol)}><Trash2 slot="icon" class="h-3.5 w-3.5" />Close</ActionButton></td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			{/if}
		</SectionPanel>
	</div>
</div>
