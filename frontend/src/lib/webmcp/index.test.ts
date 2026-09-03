import { describe, expect, it } from 'vitest';
import { buildWebMCPTools } from './index';

const mutationNames = ['deploy_project', 'start_project', 'stop_project', 'restart_project'];
const forbiddenNames = [
	'delete_project',
	'reveal_env_var',
	'set_env_vars',
	'delete_env_var',
	'update_system',
	'database_write',
	'run_shell'
];

describe('WebMCP site tools', () => {
	it('keeps tool names unique', () => {
		const tools = buildWebMCPTools({ role: 'owner' });
		const names = tools.map((tool) => tool.name);
		expect(new Set(names).size).toBe(names.length);
	});

	it('exposes the bounded project operation surface', () => {
		const names = buildWebMCPTools({ role: 'owner' }).map((tool) => tool.name);
		expect(names).toEqual(expect.arrayContaining([
			'list_projects',
			'get_project',
			'list_deployments',
			'get_deployment',
			'get_logs',
			'get_metrics_snapshot',
			'run_diagnostics',
			...mutationNames
		]));
		for (const forbidden of forbiddenNames) expect(names).not.toContain(forbidden);
	});

	it('exposes host, container and DB schema tools to every owner', () => {
		const names = buildWebMCPTools({ role: 'owner' }).map((tool) => tool.name);
		for (const name of ['get_host_stats', 'list_containers', 'get_database_schema']) {
			expect(names).toContain(name);
		}
	});

	it('marks mutations and untrusted diagnostic output correctly', () => {
		const tools = buildWebMCPTools({ role: 'owner' });
		for (const name of mutationNames) {
			expect(tools.find((tool) => tool.name === name)?.annotations?.readOnlyHint).toBe(false);
		}
		expect(tools.find((tool) => tool.name === 'run_diagnostics')?.annotations).toMatchObject({
			readOnlyHint: true,
			untrustedContentHint: true
		});
		expect(tools.find((tool) => tool.name === 'get_logs')?.annotations?.untrustedContentHint).toBe(true);
	});
});
