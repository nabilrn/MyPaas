import { describe, expect, it } from 'vitest';
import rootLayout from '../../routes/+layout.svelte?raw';
import favicon from '../../assets/brand/mypaas-favicon.svg?raw';

describe('brand favicon contract', () => {
	it('uses a dedicated white SVG favicon instead of inheriting currentColor', () => {
		expect(rootLayout).toContain("import favicon from '../assets/brand/mypaas-favicon.svg'");
		expect(favicon).toContain('fill="#fff"');
		expect(favicon).not.toContain('fill="currentColor"');
	});
});
