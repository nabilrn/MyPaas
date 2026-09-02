const CURSOR_POSITION_QUERY = /\u001b\[(?:\?6|6)n/g;

export function sanitizeShellOutput(value: string) {
	return value.replace(CURSOR_POSITION_QUERY, '');
}

export function shouldPreserveCopyShortcut(
	pageSelection: string,
	inputSelectionStart: number | null,
	inputSelectionEnd: number | null
) {
	if (pageSelection.length > 0) return true;
	return inputSelectionStart !== null && inputSelectionEnd !== null && inputSelectionStart !== inputSelectionEnd;
}
