<script lang="ts">
	import { ArrowLeft, Workflow } from '@lucide/svelte';
	import { page } from '$app/stores';

	let designCanvas: HTMLDivElement | null = null;

	$: base = `/projects/${$page.params.id}/database`;
	$: schemaActive = $page.url.pathname.startsWith(`${base}/schema`);

	function graphScale(viewport: Element) {
		const layer = viewport.querySelector<HTMLElement>('[style*="transform:scale"]');
		const match = layer?.style.transform.match(/scale\(([-\d.]+)\)/);
		return match ? Number.parseFloat(match[1]) : 1;
	}

	function handleDesignWheel(event: WheelEvent) {
		if (!designCanvas || !(event.target instanceof Element)) return;
		if (event.target.closest('input, select, textarea, button, a, summary')) return;

		const viewport = event.target.closest('.overflow-auto');
		if (!(viewport instanceof HTMLElement) || !designCanvas.contains(viewport)) return;

		const mouseWheel = event.deltaMode !== 0 || Math.abs(event.deltaY) >= 48;
		const zoomGesture = event.ctrlKey || event.metaKey || mouseWheel;
		if (!zoomGesture || event.deltaY === 0) return;

		const zoomButton = designCanvas.querySelector<HTMLButtonElement>(`button[aria-label="${event.deltaY > 0 ? 'Zoom out' : 'Zoom in'}"]`);
		if (!zoomButton || zoomButton.disabled) return;

		event.preventDefault();
		const beforeScale = graphScale(viewport);
		const beforeScrollLeft = viewport.scrollLeft;
		const beforeScrollTop = viewport.scrollTop;
		const rect = viewport.getBoundingClientRect();
		const pointerX = event.clientX - rect.left;
		const pointerY = event.clientY - rect.top;

		zoomButton.click();
		requestAnimationFrame(() => {
			const afterScale = graphScale(viewport);
			if (!Number.isFinite(beforeScale) || !Number.isFinite(afterScale) || beforeScale <= 0 || afterScale <= 0) return;
			const ratio = afterScale / beforeScale;
			viewport.scrollLeft = Math.max(0, (beforeScrollLeft + pointerX) * ratio - pointerX);
			viewport.scrollTop = Math.max(0, (beforeScrollTop + pointerY) * ratio - pointerY);
		});
	}
</script>

{#if schemaActive}
	<section class="database-design-shell fixed bottom-0 left-0 right-0 top-14 z-30 flex flex-col overflow-hidden bg-gray-50 dark:bg-neutral-950 lg:left-14">
		<header class="flex h-12 shrink-0 items-center justify-between border-b border-gray-100/70 bg-white px-3 dark:border-neutral-900 dark:bg-neutral-950">
			<div class="flex min-w-0 items-center gap-3">
				<a href={base} class="app-focus inline-flex h-8 items-center gap-1.5 rounded-md border border-gray-200 px-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 hover:text-gray-950 dark:border-neutral-800 dark:text-gray-300 dark:hover:bg-neutral-900 dark:hover:text-white">
					<ArrowLeft class="h-4 w-4" aria-hidden="true" />
					Data
				</a>
				<div class="min-w-0">
					<p class="truncate text-sm font-semibold text-gray-950 dark:text-white">Schema design</p>
					<p class="truncate text-[13px] text-gray-500 dark:text-gray-400">Database Studio</p>
				</div>
			</div>
		</header>
		<div bind:this={designCanvas} class="database-design-canvas min-h-0 flex-1 overflow-hidden" on:wheel|nonpassive={handleDesignWheel}>
			<slot />
		</div>
	</section>
{:else}
	<div class="database-data-shell">
		<div class="flex min-h-12 items-center justify-between gap-3 border-b border-gray-100/70 bg-white px-4 py-2 dark:border-neutral-900 dark:bg-neutral-950 lg:px-5">
			<div class="min-w-0">
				<p class="text-sm font-semibold text-gray-950 dark:text-white">Database Studio</p>
				<p class="text-[13px] text-gray-500 dark:text-gray-400">Browse project data and schema.</p>
			</div>
			<a href={`${base}/schema`} class="app-focus inline-flex h-9 items-center gap-2 rounded-md border border-gray-200 bg-white px-3 text-sm font-medium text-gray-700 hover:bg-gray-50 hover:text-gray-950 dark:border-neutral-800 dark:bg-neutral-950 dark:text-gray-300 dark:hover:bg-neutral-900 dark:hover:text-white">
				<Workflow class="h-4 w-4" aria-hidden="true" />
				Schema design
			</a>
		</div>
		<slot />
	</div>
{/if}

<style>
	:global(.database-design-canvas > .space-y-4) {
		display: flex;
		height: 100%;
		min-height: 0;
		flex-direction: column;
		gap: 0;
		overflow: hidden;
	}

	:global(.database-design-canvas > .space-y-4 > .surface:first-child) {
		flex: none;
		border: 0;
		border-bottom: 1px solid color-mix(in oklch, var(--app-border) 60%, transparent);
		border-radius: 0;
	}

	:global(.database-design-canvas > .space-y-4 > .surface:nth-child(2)) {
		display: flex;
		min-height: 0;
		flex: 1;
		flex-direction: column;
		border: 0;
		border-radius: 0;
	}

	:global(.database-design-canvas > .space-y-4 > .surface:nth-child(2) > div:nth-child(2)) {
		height: auto !important;
		min-height: 0 !important;
		max-height: none !important;
		flex: 1;
	}

	:global(.database-design-canvas > .space-y-4 > .grid) {
		display: none;
	}
</style>
