<script lang="ts">
	import { ExternalLink, FileText, MoreHorizontal, Rocket, RotateCcw, Square } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import StatusBadge from './StatusBadge.svelte';
	import type { Project } from '$types';

	export let project: Project;
	export let publicProjectHost = '';
	export let publicProjectURL = '';
	export let pendingAction: 'stop' | 'restart' | 'deploy' | null = null;

	const dispatch = createEventDispatcher<{
		deploy: void;
		restart: void;
		stop: void;
	}>();

	let actionsMenu: HTMLDetailsElement | null = null;

	$: deployLoading = pendingAction === 'deploy' || (pendingAction === null && project.status === 'building');
	$: deployLoadingLabel = pendingAction === 'deploy' ? 'Queueing...' : 'Deploying...';
	$: deployDisabled = project.status === 'building' || (pendingAction !== null && pendingAction !== 'deploy');
	$: restartDisabled = pendingAction !== null && pendingAction !== 'restart';
	$: stopDisabled = project.status === 'stopped' || (pendingAction !== null && pendingAction !== 'stop');
	$: sourceLabel = project.sourceType === 'registry'
		? 'Container Registry'
		: project.repoUrl.includes('github.com')
			? 'GitHub'
			: 'Git Repository';
	$: runtimeLabel = project.deployMode === 'compose'
		? 'Docker Compose'
		: project.deployMode === 'dockerfile'
			? 'Dockerfile'
			: project.deployMode === 'static'
				? 'Static site'
				: 'Container image';
	$: sourceDetail = project.sourceType === 'registry'
		? (project.imageRef ?? '')
		: project.branch;
	$: projectSummary = [sourceLabel, sourceDetail, runtimeLabel, project.mainService].filter(Boolean).join(' · ');

	function closeActionsMenu() {
		if (actionsMenu) actionsMenu.open = false;
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

<section class="surface overflow-visible">
	<div class="flex flex-col gap-4 p-5 sm:flex-row sm:items-start sm:justify-between">
		<div class="min-w-0">
			<div class="flex flex-wrap items-center gap-3">
				<h1 class="truncate text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{project.name}</h1>
				<StatusBadge status={project.status} pulse />
			</div>

			<a
				href={publicProjectURL}
				target="_blank"
				rel="noopener"
				class="mt-2 inline-flex max-w-full items-center gap-1.5 truncate font-mono text-xs text-gray-500 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white"
			>
				{publicProjectHost}
				<ExternalLink class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
			</a>

			<p class="mt-2 max-w-2xl truncate text-xs text-gray-500 dark:text-gray-400" title={projectSummary}>
				{projectSummary}
			</p>
		</div>

		<div class="flex shrink-0 items-center gap-2">
			<ActionButton
				variant="primary"
				on:click={() => dispatch('deploy')}
				disabled={deployDisabled}
				loading={deployLoading}
				loadingLabel={deployLoadingLabel}
			>
				<Rocket slot="icon" class="h-4 w-4" />
				Deploy
			</ActionButton>

			<details bind:this={actionsMenu} class="relative">
				<summary
					class="app-focus inline-flex h-9 w-9 cursor-pointer list-none items-center justify-center rounded-md border border-gray-300 bg-white text-gray-600 transition-colors hover:border-gray-400 hover:bg-gray-50 hover:text-gray-950 dark:border-gray-700 dark:bg-neutral-950 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-neutral-900 dark:hover:text-white [&::-webkit-details-marker]:hidden"
					aria-label="More project actions"
					title="More project actions"
				>
					<MoreHorizontal class="h-4 w-4" aria-hidden="true" />
				</summary>
				<div class="overlay absolute right-0 z-30 mt-2 w-44 overflow-hidden py-1">
					<a
						href={`/projects/${project.id}/logs`}
						class="flex items-center gap-2 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-neutral-900"
						on:click={closeActionsMenu}
					>
						<FileText class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" aria-hidden="true" />
						View logs
					</a>
					<div class="my-1 border-t border-gray-100 dark:border-neutral-800"></div>
					<button
						type="button"
						class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-200 dark:hover:bg-neutral-900"
						disabled={restartDisabled}
						on:click={restart}
					>
						<RotateCcw class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" aria-hidden="true" />
						{pendingAction === 'restart' ? 'Restarting…' : 'Restart'}
					</button>
					<button
						type="button"
						class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-red-300 dark:hover:bg-red-950/30"
						disabled={stopDisabled}
						on:click={stop}
					>
						<Square class="h-4 w-4 shrink-0" aria-hidden="true" />
						{pendingAction === 'stop' ? 'Stopping…' : 'Stop'}
					</button>
				</div>
			</details>
		</div>
	</div>
</section>
