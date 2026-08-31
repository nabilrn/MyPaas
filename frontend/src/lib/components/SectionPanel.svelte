<script lang="ts">
	export let title = '';
	export let description = '';
	export let padded = true;
	export let className = '';
	export let contentClass = '';

	$: bodyClass = contentClass || (padded ? 'p-5' : '');
</script>

<section class={`surface workspace-section min-w-0 overflow-hidden ${className}`.trim()}>
	{#if title || description || $$slots.actions}
		<div class="panel-header flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
			<div class="min-w-0">
				{#if title}
					<h2 class="panel-title">{title}</h2>
				{/if}
				{#if description}
					<p class="panel-description">{description}</p>
				{/if}
			</div>
			{#if $$slots.actions}
				<div class="flex shrink-0 flex-wrap items-center gap-2">
					<slot name="actions" />
				</div>
			{/if}
		</div>
	{/if}

	<div class={bodyClass}>
		<slot />
	</div>
</section>
