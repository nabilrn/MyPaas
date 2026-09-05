<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { AlertTriangle, Bell, CheckCircle2, ExternalLink, LoaderCircle, RotateCcw } from '@lucide/svelte';
	import { dismissable } from '$lib/actions/dismissable';
	import { isUpdateBusy, type UpdateSnapshot } from '$lib/system-update';
	import ActionButton from './ActionButton.svelte';
	import IconButton from './IconButton.svelte';

	export let enabled = false;

	let open = false;
	let snapshot: UpdateSnapshot | null = null;
	let loading = false;
	let mounted = false;
	let poll: ReturnType<typeof setInterval> | undefined;

	$: busy = isUpdateBusy(snapshot);
	$: releaseAvailable = Boolean(snapshot?.release?.available);
	$: lastFailure = !releaseAvailable && !busy && (snapshot?.status.state === 'failed' || snapshot?.status.state === 'rolled_back' || snapshot?.status.state === 'blocked');
	$: attention = releaseAvailable || busy || lastFailure;
	$: bellLabel = releaseAvailable ? 'Notifications, MyPaaS update available' : busy ? 'Notifications, MyPaaS update in progress' : 'Notifications';

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
			snapshot = await response.json() as UpdateSnapshot;
		} catch {
			// Keep the last useful snapshot while the dashboard or API restarts.
		} finally {
			if (showLoading) loading = false;
		}
	}

	async function openUpdatePage() {
		open = false;
		await goto('/admin/update');
	}

	function shortSha(value: string) {
		return value ? value.slice(0, 12) : 'unknown';
	}

	function statusLabel() {
		if (busy) return 'Update in progress';
		if (snapshot?.status.state === 'rolled_back') return 'Last update rolled back';
		if (snapshot?.status.state === 'failed') return 'Last update failed';
		if (snapshot?.status.state === 'blocked') return 'Last update was blocked';
		if (snapshot?.status.state === 'succeeded') return 'Update completed';
		return 'Updater idle';
	}
</script>

{#if enabled}
	<div class="relative" use:dismissable={{ enabled: open, onDismiss: close }}>
		<div class="relative">
			<IconButton label={bellLabel} variant="ghost" on:click={toggle}>
				<Bell class="h-[18px] w-[18px]" aria-hidden="true" />
			</IconButton>
			{#if attention}
				<span class="pointer-events-none absolute right-0.5 top-0.5 h-2 w-2 rounded-full bg-gray-950 ring-2 ring-white dark:bg-white dark:ring-neutral-950" aria-hidden="true"></span>
			{/if}
		</div>

		{#if open}
			<div class="overlay absolute right-0 mt-2 w-[min(23rem,calc(100vw-2rem))] overflow-hidden">
				<div class="border-b border-gray-100 px-4 py-3 dark:border-neutral-800">
					<p class="text-sm font-semibold text-gray-950 dark:text-white">System update</p>
					<p class="mt-0.5 text-[13px] text-gray-500 dark:text-gray-400">Qualified MyPaaS releases and host updater status.</p>
				</div>

				{#if loading && !snapshot}
					<div class="flex items-center justify-center gap-2 px-4 py-8 text-sm text-gray-500 dark:text-gray-400">
						<LoaderCircle class="h-4 w-4 animate-spin" aria-hidden="true" />
						Checking release status
					</div>
				{:else if snapshot}
					<div>
						{#if busy}
							<div class="px-4 py-3">
								<div class="flex items-start gap-2.5">
									<LoaderCircle class="mt-0.5 h-4 w-4 shrink-0 animate-spin text-gray-500" aria-hidden="true" />
									<div class="min-w-0">
										<p class="text-sm font-medium text-gray-950 dark:text-white">Update in progress</p>
										<p class="mt-0.5 text-[13px] leading-5 text-gray-500 dark:text-gray-400">{snapshot.status.message || 'The host updater is running.'}</p>
									</div>
								</div>
								<ActionButton variant="secondary" size="xs" full className="mt-3" on:click={openUpdatePage}>View progress</ActionButton>
							</div>
						{:else if releaseAvailable && snapshot.release}
							<div class="px-4 py-3">
								<div class="flex items-start justify-between gap-3">
									<div class="min-w-0">
										<div class="flex items-center gap-2">
											<p class="text-sm font-semibold text-gray-950 dark:text-white">{snapshot.release.tagName}</p>
											{#if snapshot.release.prerelease}<span class="chip px-1.5 py-0.5 text-[10px]">RC</span>{/if}
										</div>
										<p class="mt-0.5 text-[13px] text-gray-500 dark:text-gray-400">New qualified release available</p>
									</div>
									{#if snapshot.release.htmlUrl}
										<a class="app-focus inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:bg-neutral-900 dark:hover:text-white" href={snapshot.release.htmlUrl} target="_blank" rel="noreferrer" aria-label="View release on GitHub">
											<ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
										</a>
									{/if}
								</div>
								<ActionButton variant="secondary" size="xs" full className="mt-3" on:click={openUpdatePage}>View update</ActionButton>
							</div>
						{:else if lastFailure}
							<div class="px-4 py-3">
								<div class="flex items-start gap-2.5">
									{#if snapshot.status.state === 'rolled_back'}
										<RotateCcw class="mt-0.5 h-4 w-4 shrink-0 text-gray-500" aria-hidden="true" />
									{:else}
										<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-gray-500" aria-hidden="true" />
									{/if}
									<div class="min-w-0">
										<p class="text-sm font-medium text-gray-950 dark:text-white">{statusLabel()}</p>
										<p class="mt-0.5 text-[13px] leading-5 text-gray-500 dark:text-gray-400">{snapshot.status.message || 'Open System update for details.'}</p>
										<p class="mt-1 font-mono text-[11px] text-gray-400 dark:text-gray-500">build {shortSha(snapshot.status.currentSha)}</p>
									</div>
								</div>
								<ActionButton variant="secondary" size="xs" full className="mt-3" on:click={openUpdatePage}>View details</ActionButton>
							</div>
						{:else}
							<div class="px-4 py-5 text-center">
								<CheckCircle2 class="mx-auto h-5 w-5 text-gray-400" aria-hidden="true" />
								<p class="mt-2 text-sm font-medium text-gray-700 dark:text-gray-200">MyPaaS is up to date</p>
								{#if snapshot.release}<p class="mt-1 text-[13px] text-gray-500 dark:text-gray-400">{snapshot.release.tagName} is installed.</p>{/if}
								<ActionButton variant="ghost" size="xs" className="mt-2" on:click={openUpdatePage}>Open updater</ActionButton>
							</div>
						{/if}
					</div>
				{:else}
					<div class="px-4 py-8 text-center">
						<Bell class="mx-auto h-5 w-5 text-gray-300 dark:text-gray-700" aria-hidden="true" />
						<p class="mt-2 text-sm font-medium text-gray-700 dark:text-gray-200">Status unavailable</p>
						<p class="mt-1 text-[13px] text-gray-500 dark:text-gray-400">Open System update to retry the status check.</p>
						<ActionButton variant="ghost" size="xs" className="mt-2" on:click={openUpdatePage}>Open updater</ActionButton>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
