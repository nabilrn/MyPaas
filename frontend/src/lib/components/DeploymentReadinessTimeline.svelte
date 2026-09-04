<script lang="ts">
	import { Check, Circle, LoaderCircle } from '@lucide/svelte';
	import {
		deploymentReadinessSteps,
		deploymentReadinessSummary,
		lastDeploymentLogLine
	} from '$lib/utils/deploymentReadiness';
	import type { DeployStatus } from '$types';

	export let status: DeployStatus;
	export let buildLog: string | null = null;
	export let active = true;

	$: summary = deploymentReadinessSummary(status, active);
	$: timeline = deploymentReadinessSteps(status);
	$: lastLogLine = lastDeploymentLogLine(buildLog);
</script>

<div class="mt-3 border-t border-[color:var(--workspace-divider)] pt-2.5">
	<div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
		<div class="min-w-0">
			<p class="text-sm font-medium text-gray-950 dark:text-white">{summary.title}</p>
			{#if summary.detail}<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{summary.detail}</p>{/if}
		</div>
		{#if lastLogLine}
			<div class="min-w-0 max-w-full text-right sm:max-w-[48%]">
				<p class="text-[11px] font-medium uppercase tracking-[0.06em] text-gray-400 dark:text-gray-500">Last event</p>
				<code class="mt-0.5 block truncate font-mono text-xs text-gray-600 dark:text-gray-300" title={lastLogLine}>{lastLogLine}</code>
			</div>
		{/if}
	</div>

	<ol class="mt-2.5 grid gap-2 sm:grid-cols-4" aria-label="Deployment readiness stages">
		{#each timeline as step}
			<li class={`flex items-center gap-2 text-xs ${step.state === 'pending' ? 'text-gray-400 dark:text-gray-600' : 'text-gray-700 dark:text-gray-300'}`}>
				<span
					class={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full border ${step.state === 'complete'
						? 'border-gray-950 bg-gray-950 text-white dark:border-white dark:bg-white dark:text-gray-950'
						: step.state === 'active'
							? 'border-gray-400 bg-gray-100 text-gray-700 dark:border-gray-600 dark:bg-neutral-900 dark:text-gray-200'
							: 'border-gray-300 dark:border-neutral-700'}`}
					aria-hidden="true"
				>
					{#if step.state === 'complete'}
						<Check class="h-3 w-3" />
					{:else if step.state === 'active'}
						<LoaderCircle class="h-3 w-3 animate-spin motion-reduce:animate-none" />
					{:else}
						<Circle class="h-2.5 w-2.5" />
					{/if}
				</span>
				<span>{step.label}</span>
			</li>
		{/each}
	</ol>
</div>
