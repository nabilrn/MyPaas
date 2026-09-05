import { describe, expect, it } from 'vitest';
import mcpPage from '../../routes/admin/mcp/+page.svelte?raw';
import clientGrid from '../components/AgentClientGrid.svelte?raw';

describe('MCP administration workspace', () => {
	it('uses the full administration canvas with compact structural sections', () => {
		expect(mcpPage).toContain('admin-mcp-workspace w-full');
		expect(mcpPage).not.toContain('max-w-5xl');
		expect(mcpPage).toContain('Agent friendly');
		expect(mcpPage).toContain('Agent capabilities');
		expect(mcpPage).toContain('Connect an agent');
		expect(mcpPage).toContain('ConfirmActionDialog');
	});

	it('renders named agent brands from the LobeHub static SVG catalog', () => {
		for (const agent of ['OpenAI Codex', 'Claude Code', 'GitHub Copilot', 'Cursor', 'Windsurf', 'Cline', 'Roo Code', 'Amp', 'Junie', 'Qoder', 'Replit']) {
			expect(clientGrid).toContain(agent);
		}
		expect(clientGrid).toContain('@lobehub/icons-static-svg@1.94.0/icons');
		expect(clientGrid).toContain('agent-marquee-track');
		expect(clientGrid).toContain('agent-tooltip');
		expect(clientGrid).not.toContain('/brands/agents/agent-logos.svg#');
	});
});
