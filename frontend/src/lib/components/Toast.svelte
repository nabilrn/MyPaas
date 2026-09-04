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
		success: 'text-emerald-700 dark:text-emerald-300',
		error: 'text-red-700 dark:text-red-300',
		warning: 'text-amber-700 dark:text-amber-300',
		info: 'text-sky-700 dark:text-sky-300'
	};

	const iconSurfaceStyles: Record<Toast['kind'], string> = {
		success: 'bg-emerald-50 dark:bg-emerald-950/55',
		error: 'bg-red-50 dark:bg-red-950/55',
		warning: 'bg-amber-50 dark:bg-amber-950/55',
		info: 'bg-sky-50 dark:bg-sky-950/55'
	};

	const accentStyles: Record<Toast['kind'], string> = {
		success: 'bg-emerald-500',
		error: 'bg-red-500',
		warning: 'bg-amber-500',
		info: 'bg-sky-500'
	};
</script>

<div class="pointer-events-none fixed bottom-3 right-3 z-50 flex flex-col items-end gap-2 sm:bottom-4 sm:right-4" aria-live="polite" aria-relevant="additions">
	{#each $toast as t (t.id)}
		{@const ToastIcon = iconByKind[t.kind]}
		<div
			transition:fly={{ x: 14, duration: 160 }}
			role={t.kind === 'error' ? 'alert' : 'status'}
			class="pointer-events-auto relative flex min-w-[14rem] max-w-[min(26rem,calc(100vw-1.5rem))] items-center gap-2.5 overflow-hidden rounded-md border border-gray-300 bg-white px-3 py-2.5 text-[13px] font-medium text-gray-950 shadow-[0_10px_30px_rgba(15,23,42,0.16)] dark:border-neutral-700 dark:bg-neutral-900 dark:text-white dark:shadow-[0_10px_30px_rgba(0,0,0,0.45)]"
		>
			<span class={`absolute inset-y-0 left-0 w-0.5 ${accentStyles[t.kind]}`} aria-hidden="true"></span>
			<span class={`inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full ${iconSurfaceStyles[t.kind]}`} aria-hidden="true">
				<ToastIcon class={`h-3.5 w-3.5 ${iconStyles[t.kind]}`} />
			</span>
			<p class="min-w-0 flex-1 leading-5">{t.message}</p>
			<button
				on:click={() => toast.remove(t.id)}
				class="app-focus -mr-1 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-neutral-800 dark:hover:text-white"
				aria-label="Dismiss notification"
			>
				<X class="h-3.5 w-3.5" aria-hidden="true" />
			</button>
		</div>
	{/each}
</div>
