import { describe, expect, it } from 'vitest';
import mcpPage from '../../routes/admin/mcp/+page.svelte?raw';
import clientGrid from '../components/AgentClientGrid.svelte?raw';

describe('MCP administration workspace', () => {
	it('uses the full administration canvas with compact structural sections', () => {
		expect(mcpPage).toContain('admin-mcp-workspace w-full');
		expect(mcpPage).not.toContain('max-w-5xl');
		expect(mcpPage).toContain('Agent capabilities');
		expect(mcpPage).toContain('Connect a client');
		expect(mcpPage).toContain('ConfirmActionDialog');
	});

	it('renders named supported clients with their branded logo colors', () => {
		for (const client of ['OpenAI Codex', 'Claude Code', 'GitHub Copilot', 'Cursor', 'Gemini CLI']) {
			expect(clientGrid).toContain(client);
		}
		expect(clientGrid).toContain('/brands/agents/agent-logos.svg#');
		expect(clientGrid).toContain('style={`color: ${client.color}`}');
	});
});
