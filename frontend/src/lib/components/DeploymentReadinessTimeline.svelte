<script lang="ts">
  import { Check, Circle, LoaderCircle } from "@lucide/svelte";
  import StatusBadge from "$components/StatusBadge.svelte";
  import {
    deploymentReadinessSteps,
    deploymentReadinessSummary,
    lastDeploymentLogLine,
  } from "$lib/utils/deploymentReadiness";
  import type { DeployStatus } from "$types";

  export let status: DeployStatus;
  export let buildLog: string | null = null;
  export let active = true;

  $: summary = deploymentReadinessSummary(status, active);
  $: timeline = deploymentReadinessSteps(status);
  $: lastLogLine = lastDeploymentLogLine(buildLog);
</script>

<div class="mt-3 border-t border-gray-100/70 pt-3 dark:border-neutral-900">
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div class="min-w-0">
      <div class="flex flex-wrap items-center gap-2">
        <p class="text-sm font-medium text-gray-950 dark:text-white">
          {summary.title}
        </p>
        <StatusBadge {status} pulse={status !== "running"} />
      </div>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {summary.detail}
      </p>
    </div>
    {#if lastLogLine}
      <div class="min-w-0 max-w-full text-right sm:max-w-[45%]">
        <p class="metric-label">Last event</p>
        <code
          class="mt-1 block truncate font-mono text-xs text-gray-600 dark:text-gray-300"
          title={lastLogLine}>{lastLogLine}</code
        >
      </div>
    {/if}
  </div>

  <ol
    class="mt-3 grid gap-2 sm:grid-cols-4"
    aria-label="Deployment readiness stages"
  >
    {#each timeline as step}
      <li
        class="flex items-center gap-2 text-xs {step.state === 'pending'
          ? 'text-gray-400 dark:text-gray-600'
          : 'text-gray-700 dark:text-gray-300'}"
      >
        <span
          class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full border {step.state ===
          'complete'
            ? 'border-emerald-500 bg-emerald-500 text-white'
            : step.state === 'active'
              ? 'border-amber-500 text-amber-600 dark:text-amber-300'
              : 'border-gray-300 dark:border-neutral-700'}"
          aria-hidden="true"
        >
          {#if step.state === "complete"}
            <Check class="h-3 w-3" />
          {:else if step.state === "active"}
            <LoaderCircle
              class="h-3 w-3 animate-spin motion-reduce:animate-none"
            />
          {:else}
            <Circle class="h-2.5 w-2.5" />
          {/if}
        </span>
        <span>{step.label}</span>
      </li>
    {/each}
  </ol>
</div>
