<script lang="ts">
	import { CircleCheck, CircleX, Info, TriangleAlert, X } from '@lucide/svelte';
	import { toast, type Toast } from '$stores/toast';
	import { fly } from 'svelte/transition';

	const iconByKind = {
		success: CircleCheck,
		error: CircleX,
		warning: TriangleAlert,
		info: Info
	} satisfies Record<Toast['kind'], typeof CircleCheck>;

	const iconStyles: Record<Toast['kind'], string> = {
		success: 'text-emerald-600 dark:text-emerald-300',
		error: 'text-red-600 dark:text-red-300',
		warning: 'text-amber-600 dark:text-amber-300',
		info: 'text-sky-600 dark:text-sky-300'
	};
</script>

<div class="pointer-events-none fixed bottom-3 right-3 z-50 flex flex-col items-end gap-1.5 sm:bottom-4 sm:right-4">
	{#each $toast as t (t.id)}
		{@const ToastIcon = iconByKind[t.kind]}
		<div
			transition:fly={{ x: 14, duration: 160 }}
			class="pointer-events-auto flex max-w-[min(24rem,calc(100vw-1.5rem))] items-center gap-2 rounded-md border border-[color:var(--workspace-divider)] bg-[var(--app-surface)] px-3 py-2 text-[13px] text-[var(--app-ink)] shadow-sm"
		>
			<ToastIcon class={`h-3.5 w-3.5 shrink-0 ${iconStyles[t.kind]}`} aria-hidden="true" />
			<p class="min-w-0 leading-5">{t.message}</p>
			<button
				on:click={() => toast.remove(t.id)}
				class="app-focus -mr-1 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-400 transition-colors hover:text-gray-700 dark:text-gray-500 dark:hover:text-gray-200"
				aria-label="Dismiss notification"
			>
				<X class="h-3.5 w-3.5" aria-hidden="true" />
			</button>
		</div>
	{/each}
</div>
