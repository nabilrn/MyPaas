import { describe, expect, it } from 'vitest';
import createProject from '../../routes/projects/new/+page.svelte?raw';

describe('create project shared CPU contract', () => {
	it('does not expose a numeric CPU control and submits shared CPU', () => {
		expect(createProject).toContain('cpuLimit: 0');
		expect(createProject).toContain('Uses available host CPU; no per-project hard cap.');
		expect(createProject).not.toContain('id="cpu" type="number"');
		expect(createProject).not.toContain('>CPU cores</label>');
	});
});
