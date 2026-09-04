import { describe, expect, it } from 'vitest';
import containersPage from '../../routes/containers/+page.svelte?raw';
import auditPage from '../../routes/admin/audit-logs/+page.svelte?raw';
import deploymentsPage from '../../routes/projects/[id]/deployments/+page.svelte?raw';
import toastComponent from '../components/Toast.svelte?raw';
import designContract from '../../../DESIGN.md?raw';

describe('operational UI consistency', () => {
	it('separates container lifecycle state from health using project-table density', () => {
		expect(containersPage).toContain('<th>State</th><th>Health</th>');
		expect(containersPage).toContain("case 'unhealthy': return 'Unhealthy'");
		expect(containersPage).toContain("default: return 'No check'");
		expect(containersPage).toContain('class="h-[3.75rem]"');
		expect(containersPage).toContain('workspace-divider');
	});

	it('keeps audit rows compact and reports hidden probes honestly', () => {
		expect(auditPage).toContain('<th>Result</th>');
		expect(auditPage).toContain('routine probe');
		expect(auditPage).toContain('class="h-[3.75rem]"');
		expect(auditPage).toContain('totalShown={visibleRows.length}');
	});

	it('lets users jump back to the latest deployment output without forced scrolling', () => {
		expect(deploymentsPage).toContain('logNeedsLatest');
		expect(deploymentsPage).toContain('isLogNearBottom');
		expect(deploymentsPage).toContain('scrollToLatest');
		expect(deploymentsPage).toContain('>Latest');
		expect(deploymentsPage).toContain('Waiting for output.');
	});

	it('uses a compact neutral toast with semantic icon color', () => {
		expect(toastComponent).toContain('border-[color:var(--workspace-divider)]');
		expect(toastComponent).toContain('px-3 py-2');
		expect(toastComponent).toContain('text-[13px]');
		expect(toastComponent).not.toContain('bg-green-50');
		expect(toastComponent).not.toContain('shadow-lg');
	});

	it('documents Overview strokes, Projects table grammar, and direct UI copy', () => {
		expect(designContract).toContain('Project Overview is the source of truth for strokes');
		expect(designContract).toContain('Projects inventory table is the visual source of truth');
		expect(designContract).toContain('control-plane` is valid in engineering docs');
		expect(designContract).toContain('SHOULD NOT appear in ordinary user-facing dashboard copy');
	});
});
