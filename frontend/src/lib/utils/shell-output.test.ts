import { describe, expect, it } from 'vitest';

import { sanitizeShellOutput, shouldPreserveCopyShortcut } from './shell-output';

describe('shell output helpers', () => {
	it('removes terminal cursor-position queries without touching normal output', () => {
		expect(sanitizeShellOutput('mypaas$ \u001b[6nip a\n')).toBe('mypaas$ ip a\n');
		expect(sanitizeShellOutput('before\u001b[?6nafter')).toBe('beforeafter');
	});

	it('preserves Ctrl+C for copying selected shell output or command text', () => {
		expect(shouldPreserveCopyShortcut('selected output', 0, 0)).toBe(true);
		expect(shouldPreserveCopyShortcut('', 1, 4)).toBe(true);
		expect(shouldPreserveCopyShortcut('', 2, 2)).toBe(false);
	});
});
