<script lang="ts">
	const agents = [
		{ label: 'OpenAI Codex', mark: 'CX' },
		{ label: 'Claude Code', mark: 'CC' },
		{ label: 'GitHub Copilot', mark: 'CP' },
		{ label: 'MCP-compatible agent', mark: 'AI' }
	];
</script>

<div class="flex items-center -space-x-2" aria-label="Compatible AI agents">
	{#each agents as agent, index}
		<span
			class="relative inline-flex h-8 w-8 items-center justify-center rounded-full border border-gray-200 bg-white font-mono text-[9px] font-semibold tracking-tight text-gray-700 ring-2 ring-white transition-transform hover:z-10 hover:-translate-y-0.5 dark:border-neutral-700 dark:bg-neutral-900 dark:text-gray-200 dark:ring-neutral-950"
			style={`z-index: ${agents.length - index}`}
			title={agent.label}
			aria-label={agent.label}
		>
			{agent.mark}
		</span>
	{/each}
</div>
