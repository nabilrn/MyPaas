<script lang="ts">
	import { onMount } from 'svelte';
	import {
		AlertTriangle,
		Check,
		Circle,
		ExternalLink,
		LoaderCircle,
		RefreshCw,
		RotateCcw
	} from '@lucide/svelte';
	import { api } from '$api';
	import ActionButton from '$components/ActionButton.svelte';
	import { toast } from '$stores/toast';
	import {
		isUpdateBusy,
		phaseStepIndex,
		updateSteps,
		type UpdateSnapshot
	} from '$lib/system-update';
	import type { PageData } from './$types';

	export let data: PageData;

	let snapshot: UpdateSnapshot | null = data.snapshot ?? null;
	let mounted = false;
	let queueing = false;
	let queuedLocally = false;
	let connectionLost = false;
	let observedThisUpdate = false;
	let reloadScheduled = false;
	let baselineUpdatedAt = snapshot?.status.updatedAt ?? '';
	let pollTimer: ReturnType<typeof setTimeout> | undefined;

	$: busy = isUpdateBusy(snapshot);
	$: status = snapshot?.status ?? null;
	$: release = snapshot?.release ?? null;
	$: activeStepIndex = status ? phaseStepIndex(status.phase) : 0;
	$: terminalFailure = status?.state === 'failed' || status?.state === 'rolled_back' || status?.state === 'blocked';
	$: canUpdate = Boolean(release?.available) && !busy && !queueing && !queuedLocally;
	$: pollingFast = busy || queuedLocally;

	onMount(() => {
		mounted = true;
		if (busy) observedThisUpdate = true;
		schedulePoll(750);
		return () => {
			mounted = false;
			if (pollTimer) clearTimeout(pollTimer);
		};
	});

	function schedulePoll(delay = pollingFast ? 1000 : 30_000) {
		if (!mounted) return;
		if (pollTimer) clearTimeout(pollTimer);
		pollTimer = setTimeout(async () => {
			await refreshSnapshot();
			schedulePoll();
		}, delay);
	}

	async function refreshSnapshot() {
		try {
			const response = await fetch('/internal/system-update', {
				cache: 'no-store',
				credentials: 'include'
			});
			if (!response.ok) throw new Error(`status returned ${response.status}`);
			const next = await response.json() as UpdateSnapshot;
			connectionLost = false;

			if (queuedLocally && (isUpdateBusy(next) || next.status.updatedAt !== baselineUpdatedAt)) {
				queuedLocally = false;
			}
			if (isUpdateBusy(next)) observedThisUpdate = true;
			snapshot = next;

			if (observedThisUpdate && next.status.state === 'succeeded' && !reloadScheduled) {
				reloadScheduled = true;
				setTimeout(() => window.location.reload(), 1200);
			}
		} catch {
			connectionLost = true;
		}
	}

	async function triggerUpdate() {
		if (!canUpdate) return;
		queueing = true;
		baselineUpdatedAt = status?.updatedAt ?? '';
		try {
			await api.admin.triggerUpdate();
			queuedLocally = true;
			observedThisUpdate = true;
			toast.info('Update queued');
			await refreshSnapshot();
			schedulePoll(500);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to queue update');
		} finally {
			queueing = false;
		}
	}

	function shortSha(value: string | undefined) {
		return value ? value.slice(0, 12) : 'Unknown';
	}

	function formatChannel(value: string | undefined) {
		if (value === 'release') return 'Release';
		if (value === 'main') return 'Development';
		return 'Unknown';
	}

	function formatTimestamp(value: string) {
		return value.replace('T', ' ').replace(/Z$/, ' UTC');
	}

	function stepState(index: number) {
		if (queuedLocally && index === 0) return 'active';
		if (!status || status.state === 'idle') return 'pending';
		if (status.state === 'succeeded') return 'done';
		if (index < activeStepIndex) return 'done';
		if (index === activeStepIndex && terminalFailure) return 'error';
		if (index === activeStepIndex && busy) return 'active';
		return 'pending';
	}

	function statusTitle() {
		if (queuedLocally) return 'Update queued';
		if (!status) return 'Updater status unavailable';
		switch (status.state) {
			case 'checking': return 'Preparing update';
			case 'updating': return status.phase === 'verifying' ? 'Verifying update' : status.phase === 'rolling_back' ? 'Restoring previous runtime' : 'Applying update';
			case 'succeeded': return 'Update completed';
			case 'rolled_back': return 'Update rolled back';
			case 'failed': return 'Update failed';
			case 'blocked': return 'Update blocked';
			default: return release?.available ? 'Update available' : 'MyPaaS is up to date';
		}
	}
</script>

<svelte:head>
	<title>System update · MyPaaS</title>
</svelte:head>

<div class="page-shell">
	<div class="admin-update-workspace w-full border-b border-[color:var(--workspace-divider)]">
		<section class="grid lg:grid-cols-3">
			<div class="px-4 py-3 lg:border-r lg:border-[color:var(--workspace-divider)]">
				<p class="text-xs text-gray-500 dark:text-gray-400">Installed revision</p>
				<p class="mt-1 font-mono text-sm font-semibold text-gray-950 dark:text-white">{shortSha(status?.currentSha)}</p>
			</div>
			<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:border-t-0 lg:border-r">
				<p class="text-xs text-gray-500 dark:text-gray-400">Available release</p>
				<div class="mt-1 flex items-center gap-2">
					<p class="text-sm font-semibold text-gray-950 dark:text-white">{release?.tagName ?? 'None'}</p>
					{#if release?.prerelease}<span class="chip px-1.5 py-0.5 text-[10px]">RC</span>{/if}
				</div>
				{#if release?.targetSha}<p class="mt-0.5 font-mono text-[11px] text-gray-400 dark:text-gray-500">{shortSha(release.targetSha)}</p>{/if}
			</div>
			<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 lg:border-t-0">
				<p class="text-xs text-gray-500 dark:text-gray-400">Update channel</p>
				<p class="mt-1 text-sm font-semibold text-gray-950 dark:text-white">{formatChannel(status?.channel)}</p>
			</div>
		</section>

		<section class="border-t border-[color:var(--workspace-divider)]">
			<div class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						{#if busy || queuedLocally}
							<LoaderCircle class="h-4 w-4 shrink-0 animate-spin text-gray-500" aria-hidden="true" />
						{:else if terminalFailure}
							{#if status?.state === 'rolled_back'}<RotateCcw class="h-4 w-4 shrink-0 text-gray-500" aria-hidden="true" />{:else}<AlertTriangle class="h-4 w-4 shrink-0 text-gray-500" aria-hidden="true" />{/if}
						{:else}
							<RefreshCw class="h-4 w-4 shrink-0 text-gray-400" aria-hidden="true" />
						{/if}
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">{statusTitle()}</h2>
					</div>
					<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">
						{queuedLocally ? 'The host updater has been queued and is waiting to start.' : status?.message || 'No updater activity.'}
					</p>
					{#if connectionLost && (busy || queuedLocally)}
						<p class="mt-1 text-xs font-medium text-gray-700 dark:text-gray-300">Dashboard restarted. Reconnecting to updater status…</p>
					{/if}
				</div>

				<div class="flex shrink-0 items-center gap-2">
					{#if release?.htmlUrl}
						<a class="app-focus inline-flex h-9 items-center justify-center gap-2 rounded-md border border-gray-300 px-3 text-sm font-medium text-gray-700 hover:border-gray-400 dark:border-gray-700 dark:text-gray-300 dark:hover:border-gray-500" href={release.htmlUrl} target="_blank" rel="noreferrer">
							<ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
							Release
						</a>
					{/if}
					{#if release?.available}
						<ActionButton variant="primary" size="sm" loading={queueing} loadingLabel="Queuing update" disabled={!canUpdate} on:click={triggerUpdate}>
							{busy || queuedLocally ? 'Update in progress' : `Update to ${release.tagName}`}
						</ActionButton>
					{/if}
				</div>
			</div>
		</section>

		<section class="border-t border-[color:var(--workspace-divider)] px-4 py-4">
			<div class="mb-4">
				<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Update progress</h2>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Progress reflects host updater phases; no synthetic percentage is shown.</p>
			</div>

			<div class="max-w-3xl">
				{#each updateSteps as step, index}
					{@const state = stepState(index)}
					<div class="relative flex gap-3 pb-4 last:pb-0">
						{#if index < updateSteps.length - 1}
							<div class={`absolute left-[7px] top-4 h-[calc(100%-0.25rem)] w-px ${state === 'done' ? 'bg-gray-400 dark:bg-gray-600' : 'bg-gray-200 dark:bg-neutral-800'}`}></div>
						{/if}
						<div class="relative z-10 mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-white dark:bg-neutral-950">
							{#if state === 'done'}
								<span class="flex h-4 w-4 items-center justify-center rounded-full bg-gray-950 text-white dark:bg-white dark:text-gray-950"><Check class="h-2.5 w-2.5" strokeWidth={3} aria-hidden="true" /></span>
							{:else if state === 'active'}
								<LoaderCircle class="h-4 w-4 animate-spin text-gray-700 dark:text-gray-300" aria-hidden="true" />
							{:else if state === 'error'}
								<AlertTriangle class="h-4 w-4 text-gray-700 dark:text-gray-300" aria-hidden="true" />
							{:else}
								<Circle class="h-4 w-4 text-gray-300 dark:text-gray-700" aria-hidden="true" />
							{/if}
						</div>
						<div class="min-w-0">
							<p class={`text-sm font-medium ${state === 'pending' ? 'text-gray-400 dark:text-gray-600' : 'text-gray-900 dark:text-gray-100'}`}>{step.label}</p>
							<p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">{step.description}</p>
						</div>
					</div>
				{/each}
			</div>
		</section>

		{#if status?.updatedAt}
			<footer class="border-t border-[color:var(--workspace-divider)] px-4 py-2.5 text-[11px] text-gray-400 dark:text-gray-500">
				Last updater state: {formatTimestamp(status.updatedAt)}
			</footer>
		{/if}
	</div>
</div>
