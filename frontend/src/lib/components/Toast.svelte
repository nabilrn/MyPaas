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

	const styles: Record<Toast['kind'], string> = {
		success: 'bg-green-50 border-green-200 text-green-800 dark:bg-green-900/30 dark:border-green-700 dark:text-green-300',
		error: 'bg-red-50 border-red-200 text-red-800 dark:bg-red-900/30 dark:border-red-700 dark:text-red-300',
		warning: 'bg-yellow-50 border-yellow-200 text-yellow-800 dark:bg-yellow-900/30 dark:border-yellow-700 dark:text-yellow-300',
		info: 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-900/30 dark:border-blue-700 dark:text-blue-300'
	};
</script>

<div class="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2">
	{#each $toast as t (t.id)}
		{@const ToastIcon = iconByKind[t.kind]}
		<div
			transition:fly={{ x: 20, duration: 200 }}
			class="pointer-events-auto flex max-w-sm items-start gap-3 rounded-lg border px-4 py-3 shadow-lg {styles[t.kind]}"
		>
			<ToastIcon class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
			<p class="text-sm">{t.message}</p>
			<button
				on:click={() => toast.remove(t.id)}
				class="app-focus ml-auto inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md opacity-60 transition-opacity hover:opacity-100"
				aria-label="Dismiss notification"
			>
				<X class="h-4 w-4" aria-hidden="true" />
			</button>
		</div>
	{/each}
</div>
