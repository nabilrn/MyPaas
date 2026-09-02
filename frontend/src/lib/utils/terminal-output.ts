const CSI_SEQUENCE = /\u001b\[[0-?]*[ -\/]*[@-~]/g;
const OSC_SEQUENCE = /\u001b\][^\u0007]*(?:\u0007|\u001b\\)/g;

/**
 * Host Shell renders plain text rather than emulating a terminal. Remove
 * terminal-only control sequences so cursor/status queries such as ESC[6n do
 * not leak into the visible transcript or copied output.
 */
export function sanitizeTerminalOutput(value: string): string {
	return value.replace(OSC_SEQUENCE, '').replace(CSI_SEQUENCE, '');
}
