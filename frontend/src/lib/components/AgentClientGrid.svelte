<script lang="ts">
	const clients = [
		{ label: 'OpenAI Codex', id: 'codex', color: '#111111' },
		{ label: 'Claude Code', id: 'claude', color: '#D97757' },
		{ label: 'GitHub Copilot', id: 'copilot', color: '#8534F3' },
		{ label: 'Cursor', id: 'cursor', color: '#111111' },
		{ label: 'Windsurf', id: 'windsurf', color: '#111111' },
		{ label: 'Cline', id: 'cline', color: '#111111' },
		{ label: 'Gemini CLI', id: 'gemini', color: '#4E82EE' },
		{ label: 'Replit Agent', id: 'replit', color: '#F26207' },
		{ label: 'OpenCode', id: 'opencode', color: '#514F4F' },
		{ label: 'Roo Code', id: 'roocode', color: '#111111' }
	] as const;
</script>

<div class="grid grid-cols-2 gap-x-4 gap-y-5 border-t border-[color:var(--workspace-divider)] px-4 py-5 sm:grid-cols-5 xl:grid-cols-10" aria-label="Supported MCP clients">
	{#each clients as client}
		<div class="agent-client app-focus flex min-w-0 items-center justify-center rounded-md py-1" role="img" aria-label={client.label} title={client.label} tabindex="0">
			<span class="agent-client-mark inline-flex h-14 w-14 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white shadow-sm dark:border-neutral-700 dark:bg-white" style={`color: ${client.color}`} aria-hidden="true">
				{#if client.id === 'gemini'}
					<svg class="h-8 w-8" viewBox="0 0 24 24" role="presentation">
						<defs>
							<linearGradient id="gemini-client-gradient" x1="2" y1="2" x2="22" y2="22" gradientUnits="userSpaceOnUse">
								<stop offset="0" stop-color="#4285F4" />
								<stop offset="0.34" stop-color="#9B72CB" />
								<stop offset="0.68" stop-color="#D96570" />
								<stop offset="1" stop-color="#F9AB00" />
							</linearGradient>
						</defs>
						<path fill="url(#gemini-client-gradient)" d="M20.616 10.835a14.147 14.147 0 01-4.45-3.001 14.111 14.111 0 01-3.678-6.452.503.503 0 00-.975 0 14.134 14.134 0 01-3.679 6.452 14.155 14.155 0 01-4.45 3.001c-.65.28-1.318.505-2.002.678a.502.502 0 000 .975c.684.172 1.35.397 2.002.677a14.147 14.147 0 014.45 3.001 14.112 14.112 0 013.679 6.453.502.502 0 00.975 0c.172-.685.397-1.351.677-2.003a14.145 14.145 0 013.001-4.45 14.113 14.113 0 016.453-3.678.503.503 0 000-.975 13.245 13.245 0 01-2.003-.678z" />
					</svg>
				{:else}
					<svg class="h-8 w-8" role="presentation"><use href={`/brands/agents/agent-logos.svg#${client.id}`}></use></svg>
				{/if}
			</span>
		</div>
	{/each}
</div>

<style>
	@keyframes agent-hop {
		0% { transform: translateY(0); }
		45% { transform: translateY(-9px); }
		72% { transform: translateY(-4px); }
		100% { transform: translateY(-6px); }
	}

	@media (prefers-reduced-motion: no-preference) {
		.agent-client:hover .agent-client-mark,
		.agent-client:focus-visible .agent-client-mark {
			animation: agent-hop 380ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.agent-client:hover .agent-client-mark,
		.agent-client:focus-visible .agent-client-mark {
			transform: translateY(-4px);
		}
	}
</style>
