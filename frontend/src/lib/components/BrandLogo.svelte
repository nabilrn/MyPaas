<script lang="ts">
	import markBlack from '../../assets/new-assets/logoonly_black.png';
	import markWhite from '../../assets/new-assets/logoonly_white.png';
	import wordmarkBlack from '../../assets/new-assets/logowithtext_black.png';
	import wordmarkWhite from '../../assets/new-assets/logowithtext_white.png';

	export let compact = false;
	export let className = '';
	export let imageClass = '';

	// Keep the product mark visible without overpowering the navigation chrome.
	// Expanded sizing is width-led; compact sizing is height-led.
	$: sources = compact ? { light: markBlack, dark: markWhite } : { light: wordmarkBlack, dark: wordmarkWhite };
	$: sizing = compact ? 'h-[22px] w-auto' : 'w-28 h-auto';
</script>

<span class={`inline-flex min-w-0 items-center ${className}`.trim()}>
	<span class="sr-only">MyPaas</span>
	<img src={sources.light} alt="" aria-hidden="true" class={`${sizing} object-contain object-left dark:hidden ${imageClass}`.trim()} />
	<img src={sources.dark} alt="" aria-hidden="true" class={`${sizing} hidden object-contain object-left dark:block ${imageClass}`.trim()} />
</span>
