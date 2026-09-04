<script context="module" lang="ts">
	import { api, type WebhookStatus } from '$api';

	type WebhookConnectionState = WebhookStatus['status'];
	type CachedStatus = { status: WebhookConnectionState; fetchedAt: number };

	const webhookStatusCache = new Map<string, CachedStatus>();
	const webhookStatusRequests = new Map<string, Promise<WebhookConnectionState>>();
	const webhookStatusTtlMs = 60_000;

	async function cachedWebhookStatus(projectId: string): Promise<WebhookConnectionState> {
		const cached = webhookStatusCache.get(projectId);
		if (cached && Date.now() - cached.fetchedAt < webhookStatusTtlMs) return cached.status;

		const existing = webhookStatusRequests.get(projectId);
		if (existing) return existing;

		const request = api.projects.webhookStatus(projectId)
			.then((result) => {
				webhookStatusCache.set(projectId, { status: result.status, fetchedAt: Date.now() });
				return result.status;
			})
			.finally(() => webhookStatusRequests.delete(projectId));
		webhookStatusRequests.set(projectId, request);
		return request;
	}
</script>

<script lang="ts">
	import { onMount } from 'svelte';

	export let projectId: string;
	export let applicable = true;
	export let compact = false;

	let status: WebhookConnectionState | null = null;
	let failed = false;

	$: label = status === 'connected' ? 'Connected' : status === 'issue' ? 'Issue' : status === 'unverified' ? 'Unverified' : 'Checking';
	$: dotClass = status === 'connected'
		? 'bg-emerald-500'
		: status === 'issue'
			? 'bg-amber-500'
			: 'bg-gray-300 dark:bg-neutral-700';

	onMount(() => {
		if (!applicable) return;
		void cachedWebhookStatus(projectId)
			.then((value) => {
				status = value;
			})
			.catch(() => {
				failed = true;
			});
	});
</script>

{#if !applicable}
	<span class="text-gray-400 dark:text-gray-500">—</span>
{:else if failed}
	<a href={`/projects/${projectId}/settings/webhook`} class="inline-flex items-center gap-1.5 whitespace-nowrap text-xs text-gray-500 hover:text-gray-950 hover:underline dark:text-gray-400 dark:hover:text-white" title="Open webhook settings">
		<span class="h-1.5 w-1.5 rounded-full bg-gray-300 dark:bg-neutral-700" aria-hidden="true"></span>
		<span>{compact ? 'Unknown' : 'Unavailable'}</span>
	</a>
{:else}
	<a href={`/projects/${projectId}/settings/webhook`} class="inline-flex items-center gap-1.5 whitespace-nowrap text-xs text-gray-600 hover:text-gray-950 hover:underline dark:text-gray-300 dark:hover:text-white" title="Open webhook settings" aria-label={`Webhook ${label}`}>
		<span class={`h-1.5 w-1.5 rounded-full ${dotClass}`} aria-hidden="true"></span>
		<span>{compact && status === null ? '…' : label}</span>
	</a>
{/if}
