<script lang="ts">
	export let state: 'idle' | 'preparing' | 'ready' | 'failed' | 'expired' = 'idle';

	$: preparing = state === 'preparing';
	$: ready = state === 'ready';
</script>

<div class="migration-transfer-illustration" data-state={state} aria-hidden="true">
	<svg viewBox="0 0 720 220" class="h-auto w-full" role="presentation">
		<defs>
			<marker id="migration-arrow" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
				<path d="M 0 0 L 10 5 L 0 10 z" fill="currentColor" />
			</marker>
		</defs>

		<g class="text-gray-300 dark:text-neutral-700" fill="none" stroke="currentColor" stroke-width="1.5">
			<rect x="58" y="48" width="150" height="124" rx="8" />
			<rect x="78" y="68" width="110" height="25" rx="4" />
			<rect x="78" y="101" width="110" height="25" rx="4" />
			<rect x="78" y="134" width="110" height="18" rx="4" />
			<circle cx="96" cy="80.5" r="3" />
			<circle cx="96" cy="113.5" r="3" />
			<line x1="108" y1="80.5" x2="168" y2="80.5" />
			<line x1="108" y1="113.5" x2="168" y2="113.5" />
			<line x1="108" y1="143" x2="168" y2="143" />

			<rect x="512" y="48" width="150" height="124" rx="8" />
			<rect x="532" y="68" width="110" height="25" rx="4" />
			<rect x="532" y="101" width="110" height="25" rx="4" />
			<rect x="532" y="134" width="110" height="18" rx="4" />
			<circle cx="550" cy="80.5" r="3" />
			<circle cx="550" cy="113.5" r="3" />
			<line x1="562" y1="80.5" x2="622" y2="80.5" />
			<line x1="562" y1="113.5" x2="622" y2="113.5" />
			<line x1="562" y1="143" x2="622" y2="143" />
		</g>

		<g class={ready ? 'text-emerald-500' : 'text-gray-700 dark:text-gray-300'} fill="none" stroke="currentColor" stroke-width="1.7">
			<path d="M 315 76 L 360 55 L 405 76 L 360 97 Z" />
			<path d="M 315 76 V 128 L 360 151 L 405 128 V 76" />
			<path d="M 360 97 V 151" />
			<path d="M 337 66 L 382 87" />
		</g>

		<g class={preparing || ready ? 'text-gray-950 dark:text-white' : 'text-gray-400 dark:text-gray-600'} fill="none" stroke="currentColor" stroke-width="1.6">
			<path d="M 224 110 H 300" stroke-dasharray="5 6" marker-end="url(#migration-arrow)" class:motion-safe:animate-pulse={preparing} />
			<path d="M 420 110 H 496" stroke-dasharray="5 6" marker-end="url(#migration-arrow)" class:motion-safe:animate-pulse={preparing} />
		</g>

		<g class="text-gray-300 dark:text-neutral-700" fill="currentColor">
			<circle cx="42" cy="110" r="3" />
			<circle cx="678" cy="110" r="3" />
			<circle cx="360" cy="35" r="3" />
			<circle cx="360" cy="172" r="3" />
		</g>
	</svg>

	<div class="grid grid-cols-3 items-start gap-3 px-2 text-center">
		<div>
			<p class="text-xs font-medium text-gray-950 dark:text-white">Current VM</p>
			<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">Capture</p>
		</div>
		<div>
			<p class="text-xs font-medium text-gray-950 dark:text-white">Migration package</p>
			<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">{preparing ? 'Preparing' : ready ? 'Ready' : 'Portable archive'}</p>
		</div>
		<div>
			<p class="text-xs font-medium text-gray-950 dark:text-white">New VM</p>
			<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">Restore</p>
		</div>
	</div>
</div>
