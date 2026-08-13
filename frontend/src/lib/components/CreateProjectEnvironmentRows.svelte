<script lang="ts">
	import { ChevronDown, X } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import IconButton from './IconButton.svelte';
	import type { EnvVarDiscovery } from '$types';

	type EnvDraft = EnvVarDiscovery & { value: string };
	type EnvRow = { draft: EnvDraft; index: number; required: boolean };
	type LocalhostWarning = { host: string; port: number; service: string; suggested: string };

	export let rows: EnvRow[] = [];
	export let localhostWarnings = new Map<number, LocalhostWarning>();

	const dispatch = createEventDispatcher<{
		keyChange: { index: number; value: string };
		valueChange: { index: number; value: string };
		remove: { index: number };
		useSuggested: { index: number; value: string };
	}>();

	$: requiredRows = rows.filter((row) => row.required);
	$: optionalRows = rows.filter((row) => !row.required);
</script>

{#if requiredRows.length > 0}
	<div class="divide-y divide-gray-100 border-y border-gray-100 dark:divide-gray-800 dark:border-gray-800">
		{#each requiredRows as row}
			<div class="py-3">
				<div class="grid gap-2 sm:grid-cols-[minmax(9rem,1fr)_minmax(12rem,1.5fr)_auto] sm:items-start">
					<div class="min-w-0">
						<input
							value={row.draft.key}
							on:input={(event) => dispatch('keyChange', { index: row.index, value: (event.currentTarget as HTMLInputElement).value })}
							class="field w-full font-mono uppercase"
						/>
						<div class="mt-1 flex flex-wrap items-center gap-1.5">
							<span class="text-[11px] font-medium text-amber-700 dark:text-amber-200">Required</span>
							{#each row.draft.services ?? [] as service}<span class="font-mono text-[10px] text-gray-500 dark:text-gray-400">{service}</span>{/each}
						</div>
					</div>
					<div class="min-w-0">
						<input
							type={row.draft.sensitive ? 'password' : 'text'}
							value={row.draft.value}
							on:input={(event) => dispatch('valueChange', { index: row.index, value: (event.currentTarget as HTMLInputElement).value })}
							placeholder={row.draft.defaultValue ? `sample: ${row.draft.defaultValue}` : ''}
							class="field w-full font-mono"
						/>
						{#if row.draft.conflict}<p class="mt-1 text-[11px] text-amber-600 dark:text-amber-300">Different defaults were detected across services.</p>{/if}
					</div>
					<div class="flex items-center justify-between gap-2 sm:justify-end">
						<span class="max-w-28 truncate text-xs text-gray-500 dark:text-gray-400" title={row.draft.source}>{row.draft.source}</span>
						<IconButton label={`Remove ${row.draft.key || 'environment variable'}`} variant="ghost" type="button" on:click={() => dispatch('remove', { index: row.index })}><X class="h-4 w-4" aria-hidden="true" /></IconButton>
					</div>
				</div>
				{#if localhostWarnings.has(row.index)}
					{@const warning = localhostWarnings.get(row.index)!}
					<div class="mt-2 text-xs text-amber-800 dark:text-amber-200">
						<span class="font-medium">{row.draft.key}</span> uses <span class="font-mono">{warning.host}</span>. In Docker, localhost means the current container.
						{#if warning.service}<button type="button" class="ml-1 underline" on:click={() => dispatch('useSuggested', { index: row.index, value: warning.suggested })}>Use {warning.suggested}</button>{/if}
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}

{#if optionalRows.length > 0}
	<details class="group {requiredRows.length > 0 ? 'mt-3' : ''} rounded-md border border-gray-200 dark:border-gray-800">
		<summary class="app-focus flex cursor-pointer list-none items-center justify-between gap-3 rounded-md px-3 py-2.5 text-left hover:bg-gray-50 dark:hover:bg-gray-900 [&::-webkit-details-marker]:hidden">
			<div>
				<p class="text-sm font-medium text-gray-800 dark:text-gray-200">Optional variables</p>
				<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{optionalRows.length} detected value{optionalRows.length === 1 ? '' : 's'} · expand only when you need to override them</p>
			</div>
			<ChevronDown class="h-4 w-4 shrink-0 text-gray-500 transition-transform group-open:rotate-180 dark:text-gray-400" aria-hidden="true" />
		</summary>
		<div class="divide-y divide-gray-100 border-t border-gray-100 px-3 dark:divide-gray-800 dark:border-gray-800">
			{#each optionalRows as row}
				<div class="py-3">
					<div class="grid gap-2 sm:grid-cols-[minmax(9rem,1fr)_minmax(12rem,1.5fr)_auto] sm:items-start">
						<div class="min-w-0">
							<input
								value={row.draft.key}
								on:input={(event) => dispatch('keyChange', { index: row.index, value: (event.currentTarget as HTMLInputElement).value })}
								class="field w-full font-mono uppercase"
							/>
							<div class="mt-1 flex flex-wrap items-center gap-1.5">
								{#each row.draft.services ?? [] as service}<span class="font-mono text-[10px] text-gray-500 dark:text-gray-400">{service}</span>{/each}
							</div>
						</div>
						<div class="min-w-0">
							<input
								type={row.draft.sensitive ? 'password' : 'text'}
								value={row.draft.value}
								on:input={(event) => dispatch('valueChange', { index: row.index, value: (event.currentTarget as HTMLInputElement).value })}
								placeholder={row.draft.defaultValue ? `sample: ${row.draft.defaultValue}` : ''}
								class="field w-full font-mono"
							/>
							{#if row.draft.conflict}<p class="mt-1 text-[11px] text-amber-600 dark:text-amber-300">Different defaults were detected across services.</p>{/if}
						</div>
						<div class="flex items-center justify-between gap-2 sm:justify-end">
							<span class="max-w-28 truncate text-xs text-gray-500 dark:text-gray-400" title={row.draft.source}>{row.draft.source}</span>
							<IconButton label={`Remove ${row.draft.key || 'environment variable'}`} variant="ghost" type="button" on:click={() => dispatch('remove', { index: row.index })}><X class="h-4 w-4" aria-hidden="true" /></IconButton>
						</div>
					</div>
					{#if localhostWarnings.has(row.index)}
						{@const warning = localhostWarnings.get(row.index)!}
						<div class="mt-2 text-xs text-amber-800 dark:text-amber-200">
							<span class="font-medium">{row.draft.key}</span> uses <span class="font-mono">{warning.host}</span>. In Docker, localhost means the current container.
							{#if warning.service}<button type="button" class="ml-1 underline" on:click={() => dispatch('useSuggested', { index: row.index, value: warning.suggested })}>Use {warning.suggested}</button>{/if}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	</details>
{/if}
