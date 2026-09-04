<script lang="ts">
	const clients = [
		{ label: 'OpenAI Codex', id: 'codex', color: '#10A37F' },
		{ label: 'Claude Code', id: 'claude', color: '#D97757' },
		{ label: 'GitHub Copilot', id: 'copilot', color: '#8534F3' },
		{ label: 'Cursor', id: 'cursor', color: '#111111' },
		{ label: 'Windsurf', id: 'windsurf', color: '#0B100F' },
		{ label: 'Cline', id: 'cline', color: '#18181B' },
		{ label: 'Gemini CLI', id: 'gemini', color: '#4285F4' },
		{ label: 'Replit Agent', id: 'replit', color: '#F26207' },
		{ label: 'OpenCode', id: 'opencode', color: '#2563EB' },
		{ label: 'Roo Code', id: 'roocode', color: '#7C3AED' }
	] as const;
</script>

<div class="grid border-t border-[color:var(--workspace-divider)] sm:grid-cols-2 xl:grid-cols-5" aria-label="Supported MCP clients">
	{#each clients as client, index}
		<div
			class={`flex min-w-0 items-center gap-3 px-4 py-3 ${index > 0 ? 'border-t border-[color:var(--workspace-divider)] sm:border-t-0' : ''} ${index % 2 === 1 ? 'sm:border-l sm:border-[color:var(--workspace-divider)]' : ''} ${index >= 2 ? 'sm:border-t sm:border-[color:var(--workspace-divider)]' : ''} ${index % 5 !== 0 ? 'xl:border-l xl:border-[color:var(--workspace-divider)]' : 'xl:border-l-0'} ${index >= 5 ? 'xl:border-t xl:border-[color:var(--workspace-divider)]' : 'xl:border-t-0'}`}
		>
			<span class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-gray-200 bg-white dark:border-neutral-700 dark:bg-white" style={`color: ${client.color}`} aria-hidden="true">
				<svg class="h-5 w-5"><use href={`/brands/agents/agent-logos.svg#${client.id}`}></use></svg>
			</span>
			<div class="min-w-0">
				<p class="truncate text-sm font-medium text-gray-950 dark:text-white">{client.label}</p>
				<p class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">MCP-compatible</p>
			</div>
		</div>
	{/each}
</div>
