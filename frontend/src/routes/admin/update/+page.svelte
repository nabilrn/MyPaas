<script lang="ts">
	import { onMount } from 'svelte';
	import { beforeNavigate, goto } from '$app/navigation';
	import {
		AlertTriangle,
		CheckCircle2,
		ExternalLink,
		LoaderCircle,
		RefreshCw,
		RotateCcw
	} from '@lucide/svelte';
	import { api } from '$api';
	import ActionButton from '$components/ActionButton.svelte';
	import ConfirmActionDialog from '$components/ConfirmActionDialog.svelte';
	import { toast } from '$stores/toast';
	import {
		isUpdateBusy,
		updateStage,
		type UpdateSnapshot,
		type UpdateStatus
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
	let baselineStatusKey = statusKey(snapshot?.status ?? null);
	let pollTimer: ReturnType<typeof setTimeout> | undefined;
	let leaveDialogOpen = false;
	let pendingNavigationUrl = '';
	let allowNavigationOnce = false;

	$: busy = isUpdateBusy(snapshot);
	$: status = snapshot?.status ?? null;
	$: release = snapshot?.release ?? null;
	$: statusTargetsAvailableRelease = Boolean(
		release?.available
		&& status
		&& ((status.targetSha && status.targetSha === release.targetSha) || status.targetVersion === release.tagName)
	);
	$: terminalStateRelevant = !release?.available || statusTargetsAvailableRelease;
	$: terminalFailure = Boolean(terminalStateRelevant && (status?.state === 'failed' || status?.state === 'rolled_back' || status?.state === 'blocked'));
	$: canUpdate = Boolean(release?.available) && !busy && !queueing && !queuedLocally;
	$: pollingFast = busy || queuedLocally;
	$: progress = progressState();
	$: highlights = releaseHighlights(release?.body ?? '');
	$: guardNavigation = busy || queuedLocally;

	beforeNavigate((navigation) => {
		if (!guardNavigation) return;
		if (allowNavigationOnce) {
			allowNavigationOnce = false;
			return;
		}

		navigation.cancel();

		// SvelteKit turns cancellation of an unload navigation into the browser's
		// native leave-site confirmation. Custom dialogs are reserved for in-app
		// navigation where we can resume safely through goto().
		if (navigation.willUnload) return;

		const target = navigation.to?.url;
		if (!target) return;
		pendingNavigationUrl = `${target.pathname}${target.search}${target.hash}`;
		leaveDialogOpen = true;
	});

	onMount(() => {
		mounted = true;
		if (busy) observedThisUpdate = true;
		schedulePoll(500);
		return () => {
			mounted = false;
			if (pollTimer) clearTimeout(pollTimer);
		};
	});

	function schedulePoll(delay = pollingFast ? 500 : 30_000) {
		if (!mounted) return;
		if (pollTimer) clearTimeout(pollTimer);
		pollTimer = setTimeout(async () => {
			await refreshSnapshot();
			schedulePoll();
		}, delay);
	}

	function statusKey(value: UpdateStatus | null) {
		if (!value) return '';
		return [value.state, value.phase, value.currentSha, value.targetSha, value.targetVersion, value.updatedAt].join(':');
	}

	async function refreshSnapshot() {
		try {
			const response = await fetch('/internal/system-update', {
				cache: 'no-store',
				credentials: 'include'
			});
			if (!response.ok) throw new Error(`status returned ${response.status}`);
			const next = await response.json() as UpdateSnapshot;
			const nextKey = statusKey(next.status);
			const statusAdvanced = nextKey !== baselineStatusKey;
			connectionLost = false;

			if (queuedLocally && (isUpdateBusy(next) || statusAdvanced)) {
				queuedLocally = false;
				observedThisUpdate = true;
			}
			if (isUpdateBusy(next)) observedThisUpdate = true;
			snapshot = next;

			if (observedThisUpdate && !queuedLocally && statusAdvanced && next.status.state === 'succeeded' && !reloadScheduled) {
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
		baselineStatusKey = statusKey(status);
		try {
			await api.admin.triggerUpdate();
			queuedLocally = true;
			observedThisUpdate = false;
			toast.info('Update queued');
			await refreshSnapshot();
			schedulePoll(250);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to queue update');
		} finally {
			queueing = false;
		}
	}

	async function leaveUpdatePage() {
		if (!pendingNavigationUrl) return;
		const destination = pendingNavigationUrl;
		pendingNavigationUrl = '';
		leaveDialogOpen = false;
		allowNavigationOnce = true;
		try {
			await goto(destination);
		} finally {
			allowNavigationOnce = false;
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

	function progressState() {
		if (queuedLocally) return { step: 0, total: 6, percent: 5, label: 'Queued' };
		if (!status) return updateStage('idle');
		if (release?.available && !busy && !statusTargetsAvailableRelease) return updateStage('idle');
		return updateStage(status.phase);
	}

	function statusTitle() {
		if (queuedLocally) return 'Update queued';
		if (!status) return 'Updater status unavailable';
		if (busy) {
			return status.phase === 'verifying'
				? 'Verifying update'
				: status.phase === 'rolling_back'
					? 'Restoring previous runtime'
					: status.state === 'checking'
						? 'Preparing update'
						: 'Applying update';
		}
		if (release?.available && !terminalFailure) return 'Update available';
		switch (status.state) {
			case 'succeeded': return 'Update completed';
			case 'rolled_back': return 'Update rolled back';
			case 'failed': return 'Update failed';
			case 'blocked': return 'Update blocked';
			default: return 'MyPaaS is up to date';
		}
	}

	function statusMessage() {
		if (queuedLocally) return 'The host updater has been queued and is waiting to start.';
		if (release?.available && !busy && !terminalFailure) return `${release.tagName} is ready to install.`;
		return status?.message || 'No updater activity.';
	}

	function cleanMarkdownInline(value: string) {
		return value
			.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
			.replace(/[`*_~]/g, '')
			.trim();
	}

	function releaseHighlights(body: string) {
		if (!body.trim()) return [] as string[];
		const preferredHeadings = ['what\'s new', 'whats new', 'included changes', 'highlights', 'features', 'improvements', 'changes'];
		const excludedHeadings = ['qualification', 'test plan', 'testing', 'known issues'];
		let heading = '';
		const preferred: string[] = [];
		const fallback: string[] = [];

		for (const line of body.split(/\r?\n/)) {
			const headingMatch = line.match(/^#{1,6}\s+(.+)$/);
			if (headingMatch) {
				heading = headingMatch[1].trim().toLowerCase();
				continue;
			}
			const bulletMatch = line.match(/^\s*[-*]\s+(.+)$/);
			if (!bulletMatch) continue;
			const value = cleanMarkdownInline(bulletMatch[1]);
			if (!value) continue;
			if (!excludedHeadings.some((candidate) => heading.includes(candidate))) fallback.push(value);
			if (preferredHeadings.some((candidate) => heading.includes(candidate))) preferred.push(value);
		}

		const selected = preferred.length ? preferred : fallback;
		return selected.slice(0, 5);
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
						{:else if status?.state === 'succeeded' && !release?.available}
							<CheckCircle2 class="h-4 w-4 shrink-0 text-gray-500" aria-hidden="true" />
						{:else}
							<RefreshCw class="h-4 w-4 shrink-0 text-gray-400" aria-hidden="true" />
						{/if}
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">{statusTitle()}</h2>
					</div>
					<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{statusMessage()}</p>
					{#if connectionLost && (busy || queuedLocally)}
						<p class="mt-1 text-xs font-medium text-gray-700 dark:text-gray-300">Dashboard restarted. Reconnecting to updater status…</p>
					{/if}
				</div>

				<div class="flex shrink-0 items-center gap-2">
					{#if release?.htmlUrl}
						<a class="app-focus inline-flex h-9 items-center justify-center gap-2 rounded-md border border-gray-300 px-3 text-sm font-medium text-gray-700 hover:border-gray-400 dark:border-gray-700 dark:text-gray-300 dark:hover:border-gray-500" href={release.htmlUrl} target="_blank" rel="noreferrer">
							<ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
							Release notes
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

		{#if release && (highlights.length > 0 || release.htmlUrl)}
			<section class="border-t border-[color:var(--workspace-divider)] px-4 py-4">
				<div class="flex items-start justify-between gap-4">
					<div>
						<h2 class="text-sm font-semibold text-gray-950 dark:text-white">What’s new in {release.tagName}</h2>
						<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Release highlights from the qualified GitHub release.</p>
					</div>
				</div>
				{#if highlights.length > 0}
					<ul class="mt-3 grid gap-2 text-sm text-gray-700 dark:text-gray-300 lg:grid-cols-2">
						{#each highlights as highlight}
							<li class="flex min-w-0 gap-2">
								<span class="mt-[0.55rem] h-1 w-1 shrink-0 rounded-full bg-gray-400 dark:bg-gray-600" aria-hidden="true"></span>
								<span class="leading-5">{highlight}</span>
							</li>
						{/each}
					</ul>
				{/if}
			</section>
		{/if}

		<section class="border-t border-[color:var(--workspace-divider)] px-4 py-4">
			<div class="flex items-end justify-between gap-4">
				<div>
					<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Update progress</h2>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Stage-based progress from the host updater.</p>
				</div>
				<p class="text-xs font-medium tabular-nums text-gray-500 dark:text-gray-400">{progress.percent}%</p>
			</div>

			<div class="mt-3 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800" aria-label={`Update stage progress ${progress.percent}%`} role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={progress.percent}>
				<div
					class:active={busy || queuedLocally}
					class="update-progress-fill relative h-full w-full origin-left overflow-hidden rounded-full bg-gray-950 text-white dark:bg-white dark:text-black"
					style={`transform:scaleX(${progress.percent / 100})`}
				></div>
			</div>

			<div class="mt-3 flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
				<p class="text-sm font-medium text-gray-900 dark:text-gray-100">{progress.label}</p>
				<p class="text-xs text-gray-500 dark:text-gray-400">{progress.step > 0 ? `Step ${progress.step} of ${progress.total}` : 'Not started'}</p>
			</div>
			<p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{status?.message || 'Waiting for updater activity.'}</p>

			{#if busy || queuedLocally}
				<div class="mt-4 flex gap-2.5 border-t border-[color:var(--workspace-divider)] pt-3">
					<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" aria-hidden="true" />
					<div class="min-w-0">
						<p class="text-sm font-medium text-gray-900 dark:text-gray-100">Keep this page open while the update is running.</p>
						<p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">Do not refresh, close this tab, or restart MyPaaS manually. The dashboard may briefly reconnect while services are recreated.</p>
					</div>
				</div>
			{/if}
		</section>

		{#if status?.updatedAt}
			<footer class="border-t border-[color:var(--workspace-divider)] px-4 py-2.5 text-[11px] text-gray-400 dark:text-gray-500">
				Last updater state: {formatTimestamp(status.updatedAt)}
			</footer>
		{/if}
	</div>
</div>

<ConfirmActionDialog
	open={leaveDialogOpen}
	title="Update is still running"
	description="The host updater will continue, but leaving this page hides live progress until you return."
	confirmLabel="Leave anyway"
	cancelLabel="Stay on page"
	on:cancel={() => {
		leaveDialogOpen = false;
		pendingNavigationUrl = '';
	}}
	on:confirm={leaveUpdatePage}
>
	<p>Stay on this page if you want to see reconnect, verification, and the final update result without manually refreshing.</p>
</ConfirmActionDialog>

<style>
	.update-progress-fill {
		opacity: 1;
		transition: transform 650ms cubic-bezier(0.2, 0.8, 0.2, 1);
	}

	.update-progress-fill.active::after {
		content: '';
		position: absolute;
		top: 0;
		bottom: 0;
		left: 0;
		width: 42%;
		transform: translateX(-140%);
		background: linear-gradient(90deg, transparent, currentColor, transparent);
		opacity: 0.28;
		animation: update-progress-sheen 1.2s ease-in-out infinite;
	}

	@keyframes update-progress-sheen {
		to { transform: translateX(340%); }
	}

	@media (prefers-reduced-motion: reduce) {
		.update-progress-fill {
			transition-duration: 0ms;
		}

		.update-progress-fill.active::after {
			display: none;
			animation: none;
		}
	}
</style>