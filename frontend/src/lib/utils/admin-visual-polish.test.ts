import { describe, expect, it } from 'vitest';
import agentClientGrid from '../components/AgentClientGrid.svelte?raw';
import backupPage from '../../routes/admin/backup/+page.svelte?raw';
import migrationPage from '../../routes/admin/migration/+page.svelte?raw';

describe('administration visual polish', () => {
	it('keeps MCP agent marks overlapping, static, and discoverable on hover or focus', () => {
		expect(agentClientGrid).toContain('border-radius: 9999px');
		expect(agentClientGrid).toContain('margin-left: -0.9rem');
		expect(agentClientGrid).toContain('filter: grayscale(1)');
		expect(agentClientGrid).toContain('filter: grayscale(0)');
		expect(agentClientGrid).toContain('transform: translateY(-0.3rem)');
		expect(agentClientGrid).toContain('role="tooltip"');
		expect(agentClientGrid).toContain('tabindex="0"');
		expect(agentClientGrid).toContain("{ label: 'OpenAI Codex', slug: 'codex' }");
		expect(agentClientGrid).toContain("{ label: 'GitHub Copilot', slug: 'githubcopilot' }");
		expect(agentClientGrid).toContain("{ label: 'Windsurf', slug: 'windsurf' }");
		expect(agentClientGrid).toContain("{ label: 'Roo Code', slug: 'roocode' }");
		expect(agentClientGrid).not.toContain('@keyframes agent-marquee');
		expect(agentClientGrid).not.toContain('will-change: transform');
	});

	it('uses the Cloudflare mark instead of a generic cloud glyph for R2', () => {
		expect(backupPage).toContain('aria-label="Cloudflare"');
		expect(backupPage).toContain('text-[#F38020]');
		expect(backupPage).not.toContain('<Cloud class=');
	});

	it('removes the doubled migration top divider and centers a larger transfer illustration', () => {
		expect(migrationPage).toContain('<section class="border-b border-[color:var(--workspace-divider)]">');
		expect(migrationPage).not.toContain('<section class="border-y border-[color:var(--workspace-divider)]">');
		expect(migrationPage).toContain('min-h-[20rem] items-center justify-center');
		expect(migrationPage).toContain('max-w-[46rem]');
		expect(migrationPage).toContain('Compose named volumes');
	});
});
