<script lang="ts">
	import { createEventDispatcher, tick } from 'svelte';
	import ActionButton from './ActionButton.svelte';
	import { trapDialogFocus } from '$lib/utils/dialogFocus';

	export let open = false;
	export let title = 'Confirm action';
	export let description = '';
	export let confirmLabel = 'Confirm';
	export let cancelLabel = 'Cancel';
	export let variant: 'primary' | 'danger' = 'primary';
	export let busy = false;
	export let busyLabel = '';

	const dispatch = createEventDispatcher<{ confirm: void; cancel: void }>();
	let dialog: HTMLDivElement | null = null;
	let returnFocus: HTMLElement | null = null;
	let lastOpen = false;

	$: if (open !== lastOpen) {
		const nextOpen = open;
		lastOpen = open;
		if (typeof document !== 'undefined') {
			if (nextOpen) {
				returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
				void tick().then(() => dialog?.focus());
			} else if (returnFocus) {
				const target = returnFocus;
				returnFocus = null;
				void tick().then(() => target.focus());
			}
		}
	}

	function cancel() {
		if (!busy) dispatch('cancel');
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			cancel();
			return;
		}
		trapDialogFocus(event, dialog);
	}
</script>

{#if open}
	<div class="fixed inset-0 z-50 flex items-center justify-center p-4" on:keydown={handleKeydown}>
		<button type="button" class="absolute inset-0 cursor-default bg-gray-950/45" aria-label="Close confirmation" on:click={cancel}></button>
		<div bind:this={dialog} class="overlay relative w-full max-w-lg" role="dialog" aria-modal="true" aria-labelledby="confirm-action-title" tabindex="-1">
			<div class="panel-header">
				<h2 id="confirm-action-title" class="panel-title">{title}</h2>
				{#if description}<p class="panel-description">{description}</p>{/if}
			</div>
			{#if $$slots.default}
				<div class="border-t border-[color:var(--workspace-divider)] px-4 py-3 text-sm text-gray-600 dark:text-gray-300 sm:px-5">
					<slot />
				</div>
			{/if}
			<div class="flex justify-end gap-2 border-t border-[color:var(--workspace-divider)] px-4 py-3 sm:px-5">
				<ActionButton variant="ghost" size="sm" disabled={busy} on:click={cancel}>{cancelLabel}</ActionButton>
				<ActionButton {variant} size="sm" loading={busy} loadingLabel={busyLabel || confirmLabel} on:click={() => dispatch('confirm')}>{confirmLabel}</ActionButton>
			</div>
		</div>
	</div>
{/if}
