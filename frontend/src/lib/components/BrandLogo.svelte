<script lang="ts">
	import mark from '../../assets/brand/mypaas-icon.svg';
	import wordmark from '../../assets/brand/mypaas-logo.svg';

	export let compact = false;
	export let className = '';
	export let imageClass = '';

	$: source = compact ? mark : wordmark;
	$: defaultSizing = compact ? 'h-[22px] w-auto' : 'h-[22px] w-auto max-w-28';
	$: sizing = imageClass.trim() ? '' : defaultSizing;
</script>

<span class={`inline-flex min-w-0 items-center ${className}`.trim()}>
	<span class="sr-only">MyPaas</span>
	<img
		src={source}
		alt=""
		aria-hidden="true"
		class={`${sizing} object-contain object-left dark:invert ${imageClass}`.trim()}
	/>
</span>
