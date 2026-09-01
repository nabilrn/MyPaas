<script lang="ts">
	import { Plus, Trash2 } from '@lucide/svelte';
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

<div class="page-shell">
	<TableShell
		title="Application bindings"
		description={localBinding ? undefined : 'Warning: the container bind address is not local.'}
		{loading}
		{error}
		empty={(overview?.allocations.length ?? 0) === 0}
		emptyTitle="No allocated project ports."
		emptyDescription="A runtime port appears after a container-backed project is deployed."
		on:retry={load}
	>
		<table class="data-table table-fixed min-w-[52rem]">
			<colgroup>
				<col class="w-[18%]" />
				<col class="w-[22%]" />
				<col class="w-[19%]" />
				<col class="w-[10%]" />
				<col class="w-[13%]" />
				<col class="w-[18%]" />
			</colgroup>
			<thead><tr><th>Host binding</th><th>Project</th><th>Service</th><th>App port</th><th>Runtime</th><th>Exposure</th></tr></thead>
			<tbody>
				{#each overview?.allocations ?? [] as item}
					<tr>
						<td class="whitespace-nowrap font-mono text-[13px]"><span class="block truncate" title={`${overview?.bindHost}:${item.port}`}>{overview?.bindHost}:{item.port}</span></td>
						<td><a class="block truncate font-medium text-gray-950 hover:underline dark:text-white" title={item.projectName} href={`/projects/${item.projectId}`}>{item.projectName}</a></td>
						<td><span class="block truncate font-mono text-[13px]" title={item.service}>{item.service}</span></td>
						<td class="whitespace-nowrap font-mono text-[13px] tabular-nums">{item.appPort}</td>
						<td class="whitespace-nowrap text-sm capitalize">{item.deployMode}</td>
						<td><span class="block truncate text-[13px] text-gray-500 dark:text-gray-400" title={localBinding ? 'Local only · Caddy' : 'Host bound'}>{localBinding ? 'Local only · Caddy' : 'Host bound'}</span></td>
					</tr>
				{/each}
			</tbody>
		</table>
	</TableShell>

	<SectionPanel title="Reserved ports" padded={false}>
		<div class="overflow-x-auto">
			<table class="data-table table-fixed">
				<colgroup><col class="w-[22%]" /><col class="w-[54%]" /><col class="w-[24%]" /></colgroup>
				<thead><tr><th>Port</th><th>Used by</th><th>Status</th></tr></thead>
				<tbody>
					<tr><td class="whitespace-nowrap font-mono text-[13px]">22/tcp</td><td>SSH</td><td><span class="text-[13px] text-gray-500 dark:text-gray-400">Protected</span></td></tr>
					<tr><td class="whitespace-nowrap font-mono text-[13px]">80/tcp</td><td>Caddy HTTP</td><td><span class="text-[13px] text-gray-500 dark:text-gray-400">Protected</span></td></tr>
					<tr><td class="whitespace-nowrap font-mono text-[13px]">443/tcp</td><td>Caddy HTTPS</td><td><span class="text-[13px] text-gray-500 dark:text-gray-400">Protected</span></td></tr>
				</tbody>
			</table>
		</div>
	</SectionPanel>

	<SectionPanel title="Firewall rules" contentClass="p-0">
		{#if error && !overview}
			<div class="px-5 py-4 text-sm text-red-600 dark:text-red-300">{error}</div>
		{:else if overview && !overview.firewall.available}
			<div class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">
				Firewall helper unavailable. Project bindings remain visible, but host firewall changes are disabled.
			</div>
		{:else if overview}
			<div class="flex items-center justify-between gap-3 border-b border-gray-100/70 px-5 py-3 dark:border-neutral-900">
				<p class="text-sm font-medium text-gray-950 dark:text-white">UFW</p>
				<span class="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium" class:border-emerald-200={overview.firewall.active} class:bg-emerald-50={overview.firewall.active} class:text-emerald-700={overview.firewall.active} class:border-gray-200={!overview.firewall.active} class:text-gray-500={!overview.firewall.active}>{overview.firewall.active ? 'Active' : 'Inactive'}</span>
			</div>

			<form class="grid gap-3 border-b border-gray-100/70 px-5 py-4 dark:border-neutral-900 sm:grid-cols-[minmax(0,1fr)_10rem_auto] sm:items-end" on:submit|preventDefault={openPort}>
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
				<p class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">No MyPaaS-managed firewall rules.</p>
			{:else}
				<div class="overflow-x-auto">
					<table class="data-table table-fixed">
						<colgroup><col class="w-[18%]" /><col class="w-[18%]" /><col class="w-[40%]" /><col class="w-[24%]" /></colgroup>
						<thead><tr><th>Port</th><th>Protocol</th><th>Owner</th><th class="text-right">Action</th></tr></thead>
						<tbody>
							{#each overview.firewall.rules as rule}
								<tr>
									<td class="whitespace-nowrap font-mono text-[13px] tabular-nums">{rule.port}</td>
									<td class="whitespace-nowrap uppercase text-[13px]">{rule.protocol}</td>
									<td class="text-[13px] text-gray-500 dark:text-gray-400">MyPaaS managed</td>
									<td class="whitespace-nowrap text-right"><ActionButton variant="ghostDanger" size="xs" loading={deleting === `${rule.port}/${rule.protocol}`} loadingLabel="Closing" disabled={Boolean(deleting) && deleting !== `${rule.port}/${rule.protocol}`} on:click={() => closePort(rule.port, rule.protocol)}><Trash2 slot="icon" class="h-3.5 w-3.5" />Close</ActionButton></td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		{/if}
	</SectionPanel>
</div>
