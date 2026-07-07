<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import StatusBadge from './StatusBadge.svelte';
	import type { Project } from '$types';

	export let project: Project;
	export let publicProjectHost = '';
	export let publicProjectURL = '';
	export let logsHref = '';
	export let pendingAction: 'stop' | 'restart' | 'deploy' | null = null;

	const dispatch = createEventDispatcher<{
		deploy: void;
		restart: void;
		stop: void;
	}>();

	$: isBusy = project.status === 'building' || pendingAction !== null;
	$: deployDisabled = project.status === 'building' || (pendingAction !== null && pendingAction !== 'deploy');
	$: restartDisabled = pendingAction !== null && pendingAction !== 'restart';
	$: stopDisabled = project.status === 'stopped' || (pendingAction !== null && pendingAction !== 'stop');
	$: routeState = project.allocatedPort ? `${project.appPort} -> ${project.allocatedPort}` : `${project.appPort} -> pending`;
	$: actionState = pendingAction
		? `${pendingAction[0].toUpperCase()}${pendingAction.slice(1)} in progress`
		: project.status === 'building'
			? 'Deployment build is running'
			: project.status === 'running'
				? 'Serving traffic'
				: 'Waiting for action';
</script>

<section class="control-panel overflow-hidden">
	<div class="grid lg:grid-cols-[minmax(0,1fr)_22rem]">
		<div class="min-w-0 p-5">
			<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
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
						<svg class="h-3.5 w-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
						</svg>
					</a>
				</div>
				<p class="shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400">{actionState}</p>
			</div>

			<div class="mt-5 grid gap-3 sm:grid-cols-3">
				<div>
					<p class="metric-label">Branch</p>
					<p class="mt-1 truncate font-mono text-sm font-semibold text-gray-950 dark:text-white">{project.branch}</p>
				</div>
				<div>
					<p class="metric-label">Runtime</p>
					<p class="mt-1 truncate font-mono text-sm font-semibold text-gray-950 dark:text-white">
						{project.deployMode}{project.mainService ? ` / ${project.mainService}` : ''}
					</p>
				</div>
				<div>
					<p class="metric-label">Route</p>
					<p class="mt-1 truncate font-mono text-sm font-semibold text-gray-950 dark:text-white">{routeState}</p>
				</div>
			</div>
		</div>

		<div class="border-t border-gray-100 bg-gray-50/70 p-5 dark:border-gray-800 dark:bg-gray-900/50 lg:border-l lg:border-t-0">
			<p class="metric-label">Project actions</p>
			<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
				{isBusy ? 'One operation is active. Other actions are guarded until it finishes.' : 'Deploy from the configured branch or recover the current container.'}
			</p>
			<div class="mt-4 flex flex-wrap gap-2">
				<ActionButton
					variant="primary"
					on:click={() => dispatch('deploy')}
					disabled={deployDisabled}
					loading={pendingAction === 'deploy'}
					loadingLabel="Queueing..."
				>
					Deploy
				</ActionButton>
				<ActionButton
					on:click={() => dispatch('restart')}
					disabled={restartDisabled}
					loading={pendingAction === 'restart'}
					loadingLabel="Restarting..."
				>
					Restart
				</ActionButton>
				<ActionButton
					on:click={() => dispatch('stop')}
					disabled={stopDisabled}
					loading={pendingAction === 'stop'}
					loadingLabel="Stopping..."
				>
					Stop
				</ActionButton>
				<a
					href={logsHref}
					class="inline-flex min-h-9 items-center justify-center gap-2 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-800 transition-colors hover:border-gray-400 hover:bg-gray-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:border-gray-700 dark:bg-gray-950/80 dark:text-gray-200 dark:hover:border-gray-600 dark:hover:bg-gray-900 dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950"
				>
					Logs
				</a>
			</div>
		</div>
	</div>

	<div class="grid border-y border-gray-100 bg-gray-50/60 dark:border-gray-800 dark:bg-gray-900/60 sm:grid-cols-2 lg:grid-cols-4">
		<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-r lg:border-b-0">
			<p class="metric-label">Repository</p>
			<p class="mt-1 truncate font-mono text-xs text-gray-800 dark:text-gray-200">{project.repoUrl}</p>
		</div>
		<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 lg:border-b-0 lg:border-r">
			<p class="metric-label">App port</p>
			<p class="mt-1 font-mono text-xs text-gray-800 dark:text-gray-200">{project.appPort}</p>
		</div>
		<div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-r lg:border-b-0">
			<p class="metric-label">Allocated port</p>
			<p class="mt-1 font-mono text-xs text-gray-800 dark:text-gray-200">{project.allocatedPort ?? 'pending'}</p>
		</div>
		<div class="px-5 py-3">
			<p class="metric-label">Limits</p>
			<p class="mt-1 font-mono text-xs text-gray-800 dark:text-gray-200">{project.memoryLimitMb}MB / {project.cpuLimit} CPU</p>
		</div>
	</div>

	<slot />
</section>
