import { describe, expect, it } from 'vitest';
import agentClientGrid from '../components/AgentClientGrid.svelte?raw';
import backupPage from '../../routes/admin/backup/+page.svelte?raw';
import migrationPage from '../../routes/admin/migration/+page.svelte?raw';

describe('administration visual polish', () => {
	it('keeps supported MCP clients icon-first with brand-aware marks and hover lift', () => {
		expect(agentClientGrid).toContain('h-14 w-14');
		expect(agentClientGrid).toContain('sm:grid-cols-5 xl:grid-cols-10');
		expect(agentClientGrid).toContain('@keyframes agent-hop');
		expect(agentClientGrid).toContain("{ label: 'OpenAI Codex', id: 'codex', color: '#111111' }");
		expect(agentClientGrid).toContain("{ label: 'GitHub Copilot', id: 'copilot', color: '#8534F3' }");
		expect(agentClientGrid).toContain("{ label: 'Windsurf', id: 'windsurf', color: '#111111' }");
		expect(agentClientGrid).toContain("{ label: 'Roo Code', id: 'roocode', color: '#111111' }");
		expect(agentClientGrid).toContain('gemini-client-gradient');
		expect(agentClientGrid).not.toContain('MCP-compatible</p>');
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
