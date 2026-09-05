<script lang="ts">
	// LobeHub's static SVG package is the framework-neutral distribution of the same icon catalog.
	// Pin the package version so the dashboard does not change underneath a deployed release.
	const lobeIconBase = 'https://unpkg.com/@lobehub/icons-static-svg@1.94.0/icons';

	const agents = [
		{ label: 'OpenAI Codex', slug: 'codex' },
		{ label: 'Claude Code', slug: 'claudecode' },
		{ label: 'GitHub Copilot', slug: 'githubcopilot' },
		{ label: 'Cursor', slug: 'cursor' },
		{ label: 'Windsurf', slug: 'windsurf' },
		{ label: 'Cline', slug: 'cline' },
		{ label: 'Roo Code', slug: 'roocode' },
		{ label: 'Amp', slug: 'amp' },
		{ label: 'Junie', slug: 'junie' },
		{ label: 'Qoder', slug: 'qoder' },
		{ label: 'Replit', slug: 'replit' },
		{ label: 'TRAE', slug: 'trae' },
		{ label: 'Kilo Code', slug: 'kilocode' },
		{ label: 'Antigravity', slug: 'antigravity' },
		{ label: 'CodeBuddy', slug: 'codebuddy' },
		{ label: 'Goose', slug: 'goose' }
	] as const;

	let activeAgent = '';
	let missingColorAssets: Record<string, boolean> = {};

	function iconUrl(slug: string) {
		const wantsColor = activeAgent === slug && !missingColorAssets[slug];
		return `${lobeIconBase}/${slug}${wantsColor ? '-color' : ''}.svg`;
	}

	function handleIconError(event: Event, slug: string) {
		if (activeAgent !== slug || missingColorAssets[slug]) return;
		missingColorAssets = { ...missingColorAssets, [slug]: true };
		(event.currentTarget as HTMLImageElement).src = `${lobeIconBase}/${slug}.svg`;
	}
</script>

<div class="agent-strip border-t border-[color:var(--workspace-divider)]" aria-label="AI agents compatible with the MyPaaS MCP bridge">
	<div class="agent-strip-row">
		{#each agents as agent, index}
			<button
				type="button"
				class:agent-client-first={index === 0}
				class="agent-client"
				aria-label={agent.label}
				aria-describedby={`agent-tooltip-${agent.slug}`}
				on:mouseenter={() => (activeAgent = agent.slug)}
				on:mouseleave={() => (activeAgent = '')}
				on:focus={() => (activeAgent = agent.slug)}
				on:blur={() => (activeAgent = '')}
				on:click={() => (activeAgent = agent.slug)}
			>
				<span id={`agent-tooltip-${agent.slug}`} class="agent-tooltip" role="tooltip">{agent.label}</span>
				<span class="agent-client-mark" aria-hidden="true">
					<img
						src={iconUrl(agent.slug)}
						alt=""
						loading="lazy"
						decoding="async"
						on:error={(event) => handleIconError(event, agent.slug)}
					/>
				</span>
			</button>
		{/each}
	</div>
</div>

<style>
	.agent-strip {
		overflow-x: auto;
		padding: 2.25rem 1rem 1.15rem;
		scrollbar-width: thin;
	}

	.agent-strip-row {
		display: flex;
		width: max-content;
		min-width: 100%;
		align-items: center;
	}

	.agent-client {
		position: relative;
		margin-left: -0.9rem;
		flex: none;
		border: 0;
		background: transparent;
		padding: 0;
		outline: none;
	}

	.agent-client-first {
		margin-left: 0;
	}

	.agent-client-mark {
		display: inline-flex;
		height: 3.65rem;
		width: 3.65rem;
		align-items: center;
		justify-content: center;
		border-radius: 9999px;
		border: 1px solid rgb(209 213 219);
		background: #fff;
		box-shadow: 0 1px 2px rgb(0 0 0 / 0.08);
		transition: transform 160ms ease, border-color 160ms ease, box-shadow 160ms ease;
	}

	:global(.dark) .agent-client-mark {
		border-color: rgb(64 64 64);
	}

	.agent-client-mark img {
		display: block;
		height: 1.8rem;
		width: 1.8rem;
		object-fit: contain;
		filter: grayscale(1);
		transition: filter 160ms ease;
	}

	.agent-tooltip {
		pointer-events: none;
		position: absolute;
		left: 50%;
		bottom: calc(100% + 0.55rem);
		z-index: 60;
		width: max-content;
		max-width: 11rem;
		transform: translate(-50%, 0.25rem) scale(0.97);
		border: 1px solid rgb(229 231 235);
		border-radius: 0.375rem;
		background: rgb(255 255 255 / 0.98);
		padding: 0.3rem 0.5rem;
		font-size: 0.6875rem;
		font-weight: 500;
		line-height: 1rem;
		color: rgb(17 24 39);
		box-shadow: 0 6px 18px rgb(0 0 0 / 0.12);
		opacity: 0;
		transition: opacity 140ms ease, transform 140ms ease;
		white-space: nowrap;
	}

	:global(.dark) .agent-tooltip {
		border-color: rgb(64 64 64);
		background: rgb(23 23 23 / 0.98);
		color: #fff;
	}

	.agent-client:hover,
	.agent-client:focus-visible {
		z-index: 50;
	}

	.agent-client:hover .agent-tooltip,
	.agent-client:focus-visible .agent-tooltip {
		opacity: 1;
		transform: translate(-50%, 0) scale(1);
	}

	.agent-client:hover .agent-client-mark,
	.agent-client:focus-visible .agent-client-mark {
		transform: translateY(-0.3rem);
		border-color: rgb(156 163 175);
		box-shadow: 0 6px 14px rgb(0 0 0 / 0.12);
	}

	.agent-client:hover .agent-client-mark img,
	.agent-client:focus-visible .agent-client-mark img {
		filter: grayscale(0);
	}

	@media (prefers-reduced-motion: reduce) {
		.agent-client-mark,
		.agent-client-mark img,
		.agent-tooltip {
			transition-duration: 0ms;
		}
	}
</style>
