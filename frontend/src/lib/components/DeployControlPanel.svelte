<script lang="ts">
	import { FileText, History, MoreHorizontal, Play, Rocket, RotateCcw, Square } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import ActionLink from './ActionLink.svelte';
	import ProjectStatus from './ProjectStatus.svelte';
	import { dismissable } from '$lib/actions/dismissable';
	import { deriveProjectOperationalState } from '$lib/utils/project-operational-state';
	import type { Deployment, Project } from '$types';

	export let project: Project;
	export let latestDeployment: Deployment | null | undefined = undefined;
	export let pendingAction: 'start' | 'stop' | 'restart' | 'deploy' | null = null;

	const dispatch = createEventDispatcher<{ deploy: void; start: void; restart: void; stop: void }>();
	let actionsMenu: HTMLDetailsElement | null = null;

	$: operationalState = deriveProjectOperationalState({
		project,
		latestDeployment,
		runtimeEvidence: project.deployMode === 'static' ? 'not_applicable' : undefined
	});
	$: primaryHref = operationalState.primaryAction === 'view_logs'
		? `/projects/${project.id}/logs`
		: operationalState.primaryAction === 'view_deployment'
			? `/projects/${project.id}/deployments${latestDeployment ? `?focus=${encodeURIComponent(latestDeployment.id)}` : ''}`
			: '';
	$: primaryIcon = operationalState.primaryAction === 'view_logs'
		? FileText
		: operationalState.primaryAction === 'view_deployment'
			? History
			: operationalState.primaryAction === 'start'
				? Play
				: Rocket;
	$: primaryLoading = pendingAction === 'start' || pendingAction === 'deploy';
	$: primaryLoadingLabel = pendingAction === 'start' ? 'Starting…' : pendingAction === 'deploy' ? 'Queueing…' : operationalState.primaryActionLabel;
	$: primaryDisabled = pendingAction !== null;
	$: restartDisabled = pendingAction !== null && pendingAction !== 'restart';
	$: stopDisabled = project.status === 'stopped' || (pendingAction !== null && pendingAction !== 'stop');

	function closeActionsMenu() {
		if (actionsMenu) actionsMenu.open = false;
	}

	function handlePrimaryAction() {
		if (operationalState.primaryAction === 'start') {
			dispatch('start');
			return;
		}
		dispatch('deploy');
	}

	function restart() {
		closeActionsMenu();
		dispatch('restart');
	}

	function stop() {
		closeActionsMenu();
		dispatch('stop');
	}
</script>

<section class="workspace-section border-b border-[color:var(--workspace-divider)]">
	<div class="flex min-h-14 items-center justify-between gap-3 px-4 py-2.5">
		<div class="flex min-w-0 items-center gap-2.5">
			<h1 class="truncate text-lg font-semibold tracking-tight text-gray-950 dark:text-white">{project.name}</h1>
			<ProjectStatus status={project.status} label={operationalState.statusLabel} tone={operationalState.statusTone} />
		</div>

		<div class="flex shrink-0 items-center gap-2">
			{#if primaryHref}
				<ActionLink href={primaryHref} variant="primary">
					<svelte:component this={primaryIcon} slot="icon" class="h-4 w-4" />
					{operationalState.primaryActionLabel}
				</ActionLink>
			{:else}
				<ActionButton variant="primary" on:click={handlePrimaryAction} disabled={primaryDisabled} loading={primaryLoading} loadingLabel={primaryLoadingLabel}>
					<svelte:component this={primaryIcon} slot="icon" class="h-4 w-4" />
					{operationalState.primaryActionLabel}
				</ActionButton>
			{/if}
			<details bind:this={actionsMenu} class="relative" use:dismissable={{ enabled: true, onDismiss: closeActionsMenu }}>
				<summary class="app-focus control-square inline-flex cursor-pointer list-none items-center justify-center border border-gray-300 bg-white text-gray-600 transition-colors hover:border-gray-400 hover:bg-gray-50 hover:text-gray-950 dark:border-gray-700 dark:bg-neutral-950 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-neutral-900 dark:hover:text-white [&::-webkit-details-marker]:hidden" aria-label="More project actions" title="More project actions"><MoreHorizontal class="h-4 w-4" aria-hidden="true" /></summary>
				<div class="overlay absolute right-0 z-30 mt-2 w-44 overflow-hidden py-1">
					<button type="button" class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-200 dark:hover:bg-neutral-900" disabled={restartDisabled} on:click={restart}><RotateCcw class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" aria-hidden="true" />{pendingAction === 'restart' ? 'Restarting…' : 'Restart'}</button>
					<button type="button" class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-red-300 dark:hover:bg-red-950/30" disabled={stopDisabled} on:click={stop}><Square class="h-4 w-4 shrink-0" aria-hidden="true" />{pendingAction === 'stop' ? 'Stopping…' : 'Stop'}</button>
				</div>
			</details>
		</div>
	</div>
</section>
