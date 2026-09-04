import { describe, expect, it } from 'vitest';
import { formatCpuLimit } from './resource-format';

describe('formatCpuLimit', () => {
	it('removes floating-point noise from configured CPU limits', () => {
		expect(formatCpuLimit(0.35000000000000003)).toBe('0.35');
	});

	it('keeps compact meaningful precision', () => {
		expect(formatCpuLimit(0.2)).toBe('0.2');
		expect(formatCpuLimit(1)).toBe('1');
		expect(formatCpuLimit(1.25)).toBe('1.25');
	});

	it('rejects invalid values', () => {
		expect(formatCpuLimit(Number.NaN)).toBe('-');
		expect(formatCpuLimit(-1)).toBe('-');
	});
});
