<script lang="ts">
	import markBlack from '../../assets/new-assets/logoonly_black.png';
	import wordmarkBlack from '../../assets/new-assets/logowithtext_black.png';

	export let compact = false;
	export let className = '';
	export let imageClass = '';

	$: src = compact ? markBlack : wordmarkBlack;
	$: sizing = compact ? 'h-9 w-9' : 'h-9 w-[138px]';
</script>

<span class={`inline-flex min-w-0 items-center ${className}`.trim()}>
	<span class="sr-only">MyPaas</span>
	<img
		{src}
		alt=""
		aria-hidden="true"
		class={`${sizing} object-contain dark:invert ${compact ? 'object-center' : 'object-left'} ${imageClass}`.trim()}
	/>
</span>
