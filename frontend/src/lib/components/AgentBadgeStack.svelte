<script lang="ts">
	const agents = [
		{ label: 'OpenAI Codex', id: 'codex' },
		{ label: 'Claude Code', id: 'claude' },
		{ label: 'GitHub Copilot', id: 'copilot' },
		{ label: 'Cursor', id: 'cursor' },
		{ label: 'Windsurf', id: 'windsurf' },
		{ label: 'Cline', id: 'cline' },
		{ label: 'Gemini CLI', id: 'gemini' },
		{ label: 'Replit Agent', id: 'replit' },
		{ label: 'OpenCode', id: 'opencode' },
		{ label: 'Roo Code', id: 'roocode' }
	];
</script>

<div class="flex items-center -space-x-1.5" aria-label="Popular AI coding agents and tools">
	{#each agents as agent, index}
		<span
			class="relative inline-flex h-7 w-7 items-center justify-center rounded-full border border-gray-200 bg-white text-gray-900 ring-2 ring-white transition-transform hover:z-20 hover:-translate-y-0.5 dark:border-neutral-700 dark:bg-neutral-900 dark:text-white dark:ring-neutral-950"
			style={`z-index: ${agents.length - index + 1}`}
			title={agent.label}
		>
			<svg class="h-4 w-4" aria-hidden="true"><use href={`/brands/agents/agent-logos.svg#${agent.id}`}></use></svg>
			<span class="sr-only">{agent.label}</span>
		</span>
	{/each}
	<span class="relative z-0 ml-0.5 inline-flex h-6 min-w-6 items-center justify-center rounded-full border border-gray-200 bg-gray-50 px-1.5 font-mono text-[8px] font-semibold tracking-tight text-gray-500 ring-2 ring-white dark:border-neutral-700 dark:bg-neutral-900 dark:text-gray-400 dark:ring-neutral-950" title="And other compatible agent workflows">etc.</span>
</div>
