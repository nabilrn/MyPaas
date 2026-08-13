<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { ExternalLink } from '@lucide/svelte';
	import { api } from '$api';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	let token = '';
	let zoneId = '';
	let loading = false;
	let error = '';
	const dispatch = createEventDispatcher();

	async function submit() {
		if (!token.trim() || !zoneId.trim()) {
			error = 'Token and Zone ID are required';
			return;
		}
		loading = true;
		error = '';
		try {
			await api.admin.updateCloudflareConfig(token.trim(), zoneId.trim());
			dispatch('success');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save configuration';
		} finally {
			loading = false;
		}
	}
</script>

<SectionPanel title="Cloudflare analytics" description="Connect edge analytics without affecting live runtime telemetry.">
	<form on:submit|preventDefault={submit} class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.72fr)]">
		<div class="rounded-md border border-gray-200 p-4 text-sm text-gray-600 dark:border-neutral-800 dark:text-gray-300">
			<p class="font-medium text-gray-950 dark:text-white">Read-only analytics credentials</p>
			<p class="mt-1 leading-6">Create a Cloudflare API token with <code>Zone → Analytics → Read</code>, then copy the Zone ID from the domain overview.</p>
			<a href="https://dash.cloudflare.com/profile/api-tokens" target="_blank" rel="noopener noreferrer" class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-gray-950 hover:underline dark:text-white">
				Open Cloudflare API Tokens <ExternalLink class="h-3.5 w-3.5" />
			</a>
		</div>
		<div class="space-y-3">
			{#if error}<div class="alert-danger">{error}</div>{/if}
			<div>
				<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="zone-id">Zone ID</label>
				<input id="zone-id" type="text" bind:value={zoneId} required class="field w-full font-mono" />
			</div>
			<div>
				<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="api-token">API Token</label>
				<input id="api-token" type="password" bind:value={token} required class="field w-full font-mono" />
			</div>
			<ActionButton type="submit" variant="primary" {loading}>Save analytics configuration</ActionButton>
		</div>
	</form>
</SectionPanel>
