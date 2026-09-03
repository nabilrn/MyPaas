<script lang="ts">
	import { ExternalLink, FileText, History, MoreHorizontal, Play, Rocket, RotateCcw, Square } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import ActionLink from './ActionLink.svelte';
	import ProjectStatus from './ProjectStatus.svelte';
	import { dismissable } from '$lib/actions/dismissable';
	import { deriveProjectOperationalState } from '$lib/utils/project-operational-state';
	import type { Deployment, Project } from '$types';

	export let project: Project;
	export let latestDeployment: Deployment | null | undefined = undefined;
	export let publicProjectHost = '';
	export let publicProjectURL = '';
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
	$: sourceLabel = project.sourceType === 'registry' ? 'Container Registry' : project.repoUrl.includes('github.com') ? 'GitHub' : 'Git Repository';
	$: runtimeLabel = project.deployMode === 'compose' ? 'Docker Compose' : project.deployMode === 'dockerfile' ? 'Dockerfile' : project.deployMode === 'static' ? 'Static site' : 'Container image';
	$: sourceDetail = project.sourceType === 'registry' ? (project.imageRef ?? '') : project.branch;
	$: projectSummary = [sourceLabel, sourceDetail, runtimeLabel, project.mainService].filter(Boolean).join(' · ');

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

<section class="workspace-section border-b border-gray-100/70 dark:border-neutral-900">
	<div class="flex flex-col gap-4 px-5 py-4 sm:flex-row sm:items-start sm:justify-between">
		<div class="min-w-0">
			<div class="flex flex-wrap items-center gap-3">
				<h1 class="truncate text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{project.name}</h1>
				<ProjectStatus status={project.status} label={operationalState.statusLabel} tone={operationalState.statusTone} />
			</div>
			<a href={publicProjectURL} target="_blank" rel="noopener" class="mt-2 inline-flex max-w-full items-center gap-1.5 truncate font-mono text-xs text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white">{publicProjectHost}<ExternalLink class="h-3.5 w-3.5 shrink-0" aria-hidden="true" /></a>
			<p class="mt-2 max-w-2xl truncate text-[13px] text-gray-500 dark:text-gray-400" title={projectSummary}>{projectSummary}</p>
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
					<a href={`/projects/${project.id}/logs`} class="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-neutral-900" on:click={closeActionsMenu}><FileText class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" aria-hidden="true" />View logs</a>
					<div class="my-1 border-t border-gray-100 dark:border-neutral-800"></div>
					<button type="button" class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-200 dark:hover:bg-neutral-900" disabled={restartDisabled} on:click={restart}><RotateCcw class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" aria-hidden="true" />{pendingAction === 'restart' ? 'Restarting…' : 'Restart'}</button>
					<button type="button" class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-red-300 dark:hover:bg-red-950/30" disabled={stopDisabled} on:click={stop}><Square class="h-4 w-4 shrink-0" aria-hidden="true" />{pendingAction === 'stop' ? 'Stopping…' : 'Stop'}</button>
				</div>
			</details>
		</div>
	</div>
</section>
