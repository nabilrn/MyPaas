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
</script>

<div class="agent-marquee border-t border-[color:var(--workspace-divider)]" aria-label="AI agents compatible with the MyPaaS MCP bridge">
	<div class="agent-marquee-track">
		{#each [0, 1] as copy}
			<div class="agent-marquee-set" aria-hidden={copy === 1 ? 'true' : undefined}>
				{#each agents as agent, index}
					<div
						class:agent-client-first={index === 0}
						class="agent-client"
						role={copy === 0 ? 'img' : undefined}
						aria-label={copy === 0 ? agent.label : undefined}
					>
						<span class="agent-tooltip" role="tooltip">{agent.label}</span>
						<span class="agent-client-mark" aria-hidden="true">
							<img src={`${lobeIconBase}/${agent.slug}.svg`} alt="" loading="lazy" decoding="async" />
						</span>
					</div>
				{/each}
			</div>
		{/each}
	</div>
</div>

<style>
	.agent-marquee {
		overflow: hidden;
		padding: 2.35rem 0 1.15rem;
		-webkit-mask-image: linear-gradient(to right, transparent, #000 4%, #000 96%, transparent);
		mask-image: linear-gradient(to right, transparent, #000 4%, #000 96%, transparent);
	}

	.agent-marquee-track {
		display: flex;
		width: max-content;
		will-change: transform;
		animation: agent-marquee 38s linear infinite;
	}

	.agent-marquee-set {
		display: flex;
		align-items: center;
		padding-right: 3rem;
	}

	.agent-client {
		position: relative;
		margin-left: -0.9rem;
		flex: none;
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
		transition: border-color 160ms ease, box-shadow 160ms ease;
	}

	:global(.dark) .agent-client-mark {
		border-color: rgb(64 64 64);
	}

	.agent-client-mark img {
		display: block;
		height: 1.8rem;
		width: 1.8rem;
		object-fit: contain;
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

	.agent-client:hover {
		z-index: 50;
	}

	.agent-client:hover .agent-tooltip {
		opacity: 1;
		transform: translate(-50%, 0) scale(1);
	}

	.agent-client:hover .agent-client-mark {
		border-color: rgb(156 163 175);
		box-shadow: 0 8px 18px rgb(0 0 0 / 0.14);
	}

	.agent-marquee:hover .agent-marquee-track {
		animation-play-state: paused;
	}

	@keyframes agent-marquee {
		from { transform: translateX(0); }
		to { transform: translateX(-50%); }
	}

	@keyframes agent-bounce {
		0% { transform: translateY(0); }
		42% { transform: translateY(-0.7rem); }
		68% { transform: translateY(-0.35rem); }
		100% { transform: translateY(-0.5rem); }
	}

	@media (prefers-reduced-motion: no-preference) {
		.agent-client:hover .agent-client-mark {
			animation: agent-bounce 420ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.agent-marquee {
			overflow-x: auto;
			-webkit-mask-image: none;
			mask-image: none;
		}

		.agent-marquee-track {
			animation: none;
		}

		.agent-marquee-set[aria-hidden='true'] {
			display: none;
		}

		.agent-client:hover .agent-client-mark {
			transform: translateY(-0.25rem);
		}
	}
</style>
