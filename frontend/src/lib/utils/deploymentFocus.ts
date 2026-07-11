export function normalizeDeploymentFocus(value: string | null): string {
	return value?.trim() ?? '';
}

export function expandFocusedDeployment(current: ReadonlySet<string>, focusId: string): Set<string> {
	return focusId ? new Set([...current, focusId]) : new Set(current);
}

export function pinFocusedDeployment<T extends { id: string }>(rows: readonly T[], focused: T | null): T[] {
	if (!focused) return [...rows];
	return [focused, ...rows.filter((row) => row.id !== focused.id)];
}
