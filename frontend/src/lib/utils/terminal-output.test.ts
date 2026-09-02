import { describe, expect, it } from 'vitest';

import { sanitizeTerminalOutput } from './terminal-output';

describe('sanitizeTerminalOutput', () => {
	it('removes CSI cursor/status control sequences from plain shell output', () => {
		expect(sanitizeTerminalOutput('mypaas$ \u001b[6nip a\r\n')).toBe('mypaas$ ip a\r\n');
	});

	it('removes common ANSI styling while preserving command text', () => {
		expect(sanitizeTerminalOutput('\u001b[31merror\u001b[0m\n')).toBe('error\n');
	});

	it('removes OSC metadata sequences', () => {
		expect(sanitizeTerminalOutput('\u001b]0;title\u0007mypaas$ ')).toBe('mypaas$ ');
	});
});
