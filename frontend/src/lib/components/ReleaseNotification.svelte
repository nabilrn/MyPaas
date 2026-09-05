<script lang="ts">
	import { onMount } from 'svelte';
	import { AlertTriangle, Bell, CheckCircle2, ExternalLink, LoaderCircle, RotateCcw } from '@lucide/svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import { dismissable } from '$lib/actions/dismissable';
	import ActionButton from './ActionButton.svelte';
	import IconButton from './IconButton.svelte';

	type UpdateState = 'idle' | 'checking' | 'updating' | 'succeeded' | 'failed' | 'rolled_back' | 'blocked';
	type Snapshot = {
		status: {
			state: UpdateState;
			channel: string;
			currentSha: string;
			targetSha: string;
			targetVersion: string;
			message: string;
			updatedAt: string;
		};
		release: null | {
			tagName: string;
			name: string;
			targetSha: string;
			prerelease: boolean;
			publishedAt: string;
			htmlUrl: string;
			available: boolean;
		};
	};

	export let enabled = false;

	let open = false;
	let snapshot: Snapshot | null = null;
	let loading = false;
	let queueing = false;
	let mounted = false;
	let poll: ReturnType<typeof setInterval> | undefined;

	$: busy = snapshot?.status.state === 'checking' || snapshot?.status.state === 'updating';
	$: attention = Boolean(snapshot?.release?.available)
		|| snapshot?.status.state === 'failed'
		|| snapshot?.status.state === 'rolled_back'
		|| snapshot?.status.state === 'blocked';

	onMount(() => {
		mounted = true;
		syncPolling();
		return () => {
			mounted = false;
			if (poll) clearInterval(poll);
			poll = undefined;
		};
	});

	$: if (mounted) syncPolling();

	function syncPolling() {
		if (enabled && !poll) {
			void refresh();
			poll = setInterval(() => void refresh(false), 30_000);
			return;
		}
		if (!enabled && poll) {
			clearInterval(poll);
			poll = undefined;
			snapshot = null;
			open = false;
		}
	}

	function close() {
		open = false;
	}

	function toggle() {
		open = !open;
		if (open) void refresh();
	}

	async function refresh(showLoading = true) {
		if (!enabled) return;
		if (showLoading) loading = true;
		try {
			const response = await fetch('/internal/system-update', { cache: 'no-store', credentials: 'include' });
			if (!response.ok) return;
			snapshot = await response.json() as Snapshot;
		} catch {
			// A control-plane restart is expected while an update is running.
		} finally {
			if (showLoading) loading = false;
		}
	}

	async function queueUpdate() {
		if (queueing || busy) return;
		queueing = true;
		try {
			await api.admin.triggerUpdate();
			toast.info('Update queued');
			await refresh(false);
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to queue update');
		} finally {
			queueing = false;
		}
	}

	function shortSha(value: string) {
		return value ? value.slice(0, 12) : 'unknown';
	}

	function stateLabel(state: UpdateState) {
		switch (state) {
			case 'checking': return 'Checking for update';
			case 'updating': return 'Updating MyPaaS';
			case 'succeeded': return 'Update completed';
			case 'rolled_back': return 'Update rolled back';
			case 'failed': return 'Update failed';
			case 'blocked': return 'Update blocked';
			default: return 'Updater idle';
		}
	}
</script>

{#if enabled}
	<div class="relative" use:dismissable={{ enabled: open, onDismiss: close }}>
		<div class="relative">
			<IconButton label="Notifications" variant="ghost" on:click={toggle}>
				<Bell class="h-[18px] w-[18px]" aria-hidden="true" />
			</IconButton>
			{#if attention}
				<span class="pointer-events-none absolute right-1.5 top-1.5 h-1.5 w-1.5 rounded-full bg-gray-950 ring-2 ring-white dark:bg-white dark:ring-neutral-950" aria-hidden="true"></span>
			{/if}
		</div>

		{#if open}
			<div class="overlay absolute right-0 mt-2 w-[min(23rem,calc(100vw-2rem))] overflow-hidden">
				<div class="border-b border-gray-100 px-4 py-3 dark:border-neutral-800">
					<p class="text-sm font-semibold text-gray-950 dark:text-white">System update</p>
					<p class="mt-0.5 text-[13px] text-gray-500 dark:text-gray-400">Published MyPaaS releases and host updater status.</p>
				</div>

				{#if loading && !snapshot}
					<div class="flex items-center justify-center gap-2 px-4 py-8 text-sm text-gray-500 dark:text-gray-400">
						<LoaderCircle class="h-4 w-4 animate-spin" aria-hidden="true" />
						Checking release status
					</div>
				{:else if snapshot}
					<div class="space-y-0">
						<div class="px-4 py-3">
							<div class="flex items-start gap-2.5">
								{#if busy}
									<LoaderCircle class="mt-0.5 h-4 w-4 shrink-0 animate-spin text-gray-500" aria-hidden="true" />
								{:else if snapshot.status.state === 'failed' || snapshot.status.state === 'blocked'}
									<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-gray-500" aria-hidden="true" />
								{:else if snapshot.status.state === 'rolled_back'}
									<RotateCcw class="mt-0.5 h-4 w-4 shrink-0 text-gray-500" aria-hidden="true" />
								{:else}
									<CheckCircle2 class="mt-0.5 h-4 w-4 shrink-0 text-gray-400" aria-hidden="true" />
								{/if}
								<div class="min-w-0">
									<p class="text-sm font-medium text-gray-950 dark:text-white">{stateLabel(snapshot.status.state)}</p>
									<p class="mt-0.5 text-[13px] leading-5 text-gray-500 dark:text-gray-400">{snapshot.status.message || 'No updater activity.'}</p>
									<p class="mt-1 font-mono text-[11px] text-gray-400 dark:text-gray-500">build {shortSha(snapshot.status.currentSha)}</p>
								</div>
							</div>
						</div>

						{#if snapshot.release}
							<div class="border-t border-gray-100 px-4 py-3 dark:border-neutral-800">
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0">
										<div class="flex items-center gap-2">
											<p class="text-sm font-semibold text-gray-950 dark:text-white">{snapshot.release.tagName}</p>
											{#if snapshot.release.prerelease}<span class="chip px-1.5 py-0.5 text-[10px]">RC</span>{/if}
										</div>
										<p class="mt-0.5 truncate text-[13px] text-gray-500 dark:text-gray-400">{snapshot.release.available ? 'New published release available' : 'Current release is installed'}</p>
									</div>
									{#if snapshot.release.htmlUrl}
										<a class="app-focus inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-neutral-900 dark:hover:text-white" href={snapshot.release.htmlUrl} target="_blank" rel="noreferrer" aria-label="View release on GitHub">
											<ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
										</a>
									{/if}
								</div>

								{#if snapshot.release.available}
									<ActionButton variant="secondary" size="xs" full className="mt-3" disabled={busy || queueing} on:click={queueUpdate}>
										{queueing ? 'Queuing update…' : busy ? 'Update in progress' : `Update to ${snapshot.release.tagName}`}
									</ActionButton>
								{/if}
							</div>
						{/if}
					</div>
				{:else}
					<div class="px-4 py-8 text-center">
						<Bell class="mx-auto h-5 w-5 text-gray-300 dark:text-gray-700" aria-hidden="true" />
						<p class="mt-2 text-sm font-medium text-gray-700 dark:text-gray-200">Status unavailable</p>
						<p class="mt-1 text-[13px] text-gray-500 dark:text-gray-400">The host update status could not be read.</p>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
