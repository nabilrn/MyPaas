<script lang="ts">
	import { Check, CircleAlert, CircleCheck, CircleDashed, Copy, ExternalLink, Eye, EyeOff, RefreshCw } from '@lucide/svelte';
	import { onDestroy, onMount } from 'svelte';
	import { page } from '$app/stores';
	import ActionButton from '$components/ActionButton.svelte';
	import ConfirmActionDialog from '$components/ConfirmActionDialog.svelte';
	import ErrorState from '$components/ErrorState.svelte';
	import IconButton from '$components/IconButton.svelte';
	import LoadingIndicator from '$components/LoadingIndicator.svelte';
	import { api, type WebhookStatus } from '$api';
	import { toast } from '$stores/toast';
	import { webhookURL } from '$lib/utils/urls';
	import type { Project } from '$types';

	const githubWebhookDocs = 'https://docs.github.com/en/webhooks/using-webhooks/creating-webhooks';

	let project: Project | null = null;
	let webhookStatus: WebhookStatus | null = null;
	let loading = true;
	let loadError = '';
	let statusError = '';
	let refreshingStatus = false;
	let regeneratingSecret = false;
	let confirmRegenerateSecret = false;
	let showWebhookSecret = false;
	let copiedTarget: 'url' | 'secret' | '' = '';
	let copiedResetTimer: ReturnType<typeof setTimeout> | undefined;

	$: publicWebhookURL = project ? webhookURL(project.id, $page.url.origin) : '';
	$: statusTone = webhookStatus?.status ?? 'unverified';
	$: statusLabel = statusTone === 'connected' ? 'Connected' : statusTone === 'issue' ? 'Delivery issue' : 'Not verified';
	$: statusDescription = statusError
		? statusError
		: statusTone === 'connected'
			? webhookStatus?.lastDelivery?.processed
				? 'A signed GitHub delivery reached MyPaaS and triggered a deployment.'
				: 'A signed GitHub delivery reached MyPaaS. The latest event did not trigger a deployment.'
			: statusTone === 'issue'
				? 'The latest recorded delivery did not pass signature validation.'
				: 'No GitHub delivery has been verified for this project yet.';

	onMount(() => {
		void load();
	});

	onDestroy(() => {
		if (copiedResetTimer) clearTimeout(copiedResetTimer);
	});

	async function load() {
		loading = true;
		loadError = '';
		statusError = '';
		try {
			project = await api.projects.get($page.params.id ?? '');
			if (project.sourceType === 'git') await refreshStatus();
		} catch (error) {
			loadError = error instanceof Error ? error.message : 'Failed to load webhook settings';
		} finally {
			loading = false;
		}
	}

	async function refreshStatus() {
		if (!project || project.sourceType !== 'git' || refreshingStatus) return;
		refreshingStatus = true;
		statusError = '';
		try {
			webhookStatus = await api.projects.webhookStatus(project.id);
		} catch (error) {
			statusError = error instanceof Error ? error.message : 'Could not verify webhook delivery status';
		} finally {
			refreshingStatus = false;
		}
	}

	async function regenerateSecret() {
		if (!project || regeneratingSecret) return;
		regeneratingSecret = true;
		try {
			const result = await api.projects.regenerateWebhookSecret(project.id);
			project = { ...project, webhookSecret: result.webhookSecret };
			showWebhookSecret = true;
			confirmRegenerateSecret = false;
			toast.success('Webhook secret regenerated');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to regenerate webhook secret');
		} finally {
			regeneratingSecret = false;
		}
	}

	function copyText(value: string, successMessage: string, target: 'url' | 'secret') {
		navigator.clipboard?.writeText(value)
			.then(() => {
				copiedTarget = target;
				if (copiedResetTimer) clearTimeout(copiedResetTimer);
				copiedResetTimer = setTimeout(() => {
					copiedTarget = '';
					copiedResetTimer = undefined;
				}, 1800);
				toast.success(successMessage);
			})
			.catch(() => toast.error('Failed to copy'));
	}

	function formatDeliveryTime(value: string | undefined) {
		if (!value) return 'No delivery yet';
		const date = new Date(value);
		return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
	}
</script>

<svelte:head><title>Webhook · MyPaas</title></svelte:head>

{#if loading}
	<div class="flex min-h-64 items-center justify-center"><LoadingIndicator label="Loading webhook" /></div>
{:else if loadError || !project}
	<ErrorState title="Could not load webhook" message={loadError || 'Project not found'} on:retry={() => void load()} />
{:else if project}
	<div class="w-full">
		<div class="border-b border-[color:var(--workspace-divider)] px-5 pb-3 pt-4">
			<div class="flex items-center gap-2">
				<svg data-github-platform-icon class="h-4 w-4 shrink-0 text-gray-700 dark:text-gray-300" viewBox="0 0 24 24" aria-hidden="true">
					<path fill="currentColor" d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.84 1.237 1.84 1.237 1.07 1.835 2.809 1.305 3.495.998.108-.776.418-1.305.762-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
				</svg>
				<h1 class="text-lg font-semibold text-gray-950 dark:text-white">Webhook</h1>
			</div>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Deploy this project from signed GitHub push events.</p>
		</div>

		{#if project.sourceType !== 'git'}
			<div class="px-4 py-4">
				<div class="alert-neutral">Webhook deployment is available only for projects using a Git repository source. <a href={`/projects/${project.id}/settings`} class="font-medium underline underline-offset-2">Open Settings</a>.</div>
			</div>
		{:else}
			<section class="grid border-b border-[color:var(--workspace-divider)] lg:grid-cols-[1.1fr_0.9fr_0.8fr]">
				<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0">
							<p class="text-xs font-medium uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500">Connection</p>
							<div class="mt-1.5 flex items-center gap-2">
								{#if statusTone === 'connected'}
									<CircleCheck class="h-4 w-4 shrink-0 text-emerald-500" aria-hidden="true" />
								{:else if statusTone === 'issue'}
									<CircleAlert class="h-4 w-4 shrink-0 text-red-500" aria-hidden="true" />
								{:else}
									<CircleDashed class="h-4 w-4 shrink-0 text-gray-400" aria-hidden="true" />
								{/if}
								<p class="text-sm font-semibold text-gray-950 dark:text-white">{statusLabel}</p>
							</div>
							<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{statusDescription}</p>
						</div>
						<IconButton label="Refresh webhook status" variant="ghost" loading={refreshingStatus} on:click={() => void refreshStatus()}><RefreshCw class="h-4 w-4" aria-hidden="true" /></IconButton>
					</div>
				</div>
				<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
					<p class="text-xs font-medium uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500">Last delivery</p>
					<p class="mt-1.5 text-sm font-medium text-gray-950 dark:text-white">{formatDeliveryTime(webhookStatus?.lastDelivery?.receivedAt)}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{webhookStatus?.lastDelivery?.eventType ?? 'Waiting for GitHub'}{webhookStatus?.lastDelivery?.processed ? ' · deployment queued' : ''}</p>
				</div>
				<div class="px-4 py-3">
					<p class="text-xs font-medium uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500">Expected branch</p>
					<p class="mt-1.5 font-mono text-sm font-medium text-gray-950 dark:text-white">{project.branch}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Push events on this branch can trigger deployment.</p>
				</div>
			</section>

			<div class="grid min-w-0 lg:grid-cols-[1.15fr_0.85fr]">
				<section class="min-w-0 px-4 py-4 lg:border-r lg:border-[color:var(--workspace-divider)]">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Connection details</h2>
						<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Copy these values into the repository webhook configuration on GitHub.</p>
					</div>

					<div class="mt-4 space-y-4">
						<div>
							<p class="field-label">Payload URL</p>
							<div class="grid min-w-0 grid-cols-[minmax(0,1fr)_4.75rem] items-center gap-2">
								<p class="min-w-0 break-all font-mono text-sm text-gray-950 dark:text-white">{publicWebhookURL}</p>
								<div class="flex shrink-0 justify-end">
									<IconButton label={copiedTarget === 'url' ? 'Payload URL copied' : 'Copy payload URL'} variant="ghost" on:click={() => copyText(publicWebhookURL, 'Webhook URL copied', 'url')}>{#if copiedTarget === 'url'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
								</div>
							</div>
						</div>

						<div>
							<p class="field-label">Secret</p>
							<div class="grid min-w-0 grid-cols-[minmax(0,1fr)_4.75rem] items-center gap-2">
								<p class="min-w-0 break-all font-mono text-sm text-gray-950 dark:text-white">{showWebhookSecret ? project.webhookSecret : '••••••••••••••••'}</p>
								<div class="flex shrink-0 items-center justify-end gap-1">
									<IconButton label={showWebhookSecret ? 'Hide webhook secret' : 'Reveal webhook secret'} variant="ghost" on:click={() => (showWebhookSecret = !showWebhookSecret)}>{#if showWebhookSecret}<EyeOff class="h-4 w-4" />{:else}<Eye class="h-4 w-4" />{/if}</IconButton>
									<IconButton label={copiedTarget === 'secret' ? 'Webhook secret copied' : 'Copy webhook secret'} variant="ghost" on:click={() => copyText(project?.webhookSecret ?? '', 'Webhook secret copied', 'secret')}>{#if copiedTarget === 'secret'}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}</IconButton>
								</div>
							</div>
						</div>
					</div>

					<div class="mt-5">
						<ActionButton variant="ghost" size="sm" on:click={() => (confirmRegenerateSecret = true)}><RefreshCw slot="icon" class="h-3.5 w-3.5" />Regenerate secret</ActionButton>
					</div>
				</section>

				<section class="border-t border-[color:var(--workspace-divider)] px-4 py-4 lg:border-t-0">
					<div class="flex items-center gap-2">
						<svg data-github-platform-icon class="h-4 w-4 shrink-0 text-gray-700 dark:text-gray-300" viewBox="0 0 24 24" aria-hidden="true">
							<path fill="currentColor" d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.84 1.237 1.84 1.237 1.07 1.835 2.809 1.305 3.495.998.108-.776.418-1.305.762-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
						</svg>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">GitHub setup</h2>
					</div>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Create a repository webhook with these delivery rules.</p>

					<dl class="mt-4 grid grid-cols-[8rem_minmax(0,1fr)] gap-x-3 gap-y-3 text-sm">
						<dt class="text-gray-500 dark:text-gray-400">Content type</dt><dd class="font-mono text-gray-950 dark:text-white">application/json</dd>
						<dt class="text-gray-500 dark:text-gray-400">Events</dt><dd class="text-gray-950 dark:text-white">Just the push event</dd>
						<dt class="text-gray-500 dark:text-gray-400">Branch</dt><dd class="font-mono text-gray-950 dark:text-white">{project.branch}</dd>
						<dt class="text-gray-500 dark:text-gray-400">Verification</dt><dd class="text-gray-950 dark:text-white">HMAC SHA-256 secret</dd>
					</dl>

					<a href={githubWebhookDocs} target="_blank" rel="noreferrer" class="app-focus mt-5 inline-flex items-center gap-1.5 text-sm font-medium text-gray-700 hover:text-gray-950 dark:text-gray-300 dark:hover:text-white">
						GitHub webhook documentation <ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
					</a>
				</section>
			</div>
		{/if}
	</div>
{/if}

<ConfirmActionDialog
	open={confirmRegenerateSecret}
	title="Regenerate webhook secret?"
	description="GitHub deliveries using the current secret will fail until the repository webhook is updated."
	confirmLabel="Regenerate secret"
	busyLabel="Regenerating"
	variant="danger"
	busy={regeneratingSecret}
	on:cancel={() => (confirmRegenerateSecret = false)}
	on:confirm={regenerateSecret}
>
	<p>After regeneration, copy the new secret into the GitHub repository webhook configuration.</p>
</ConfirmActionDialog>
