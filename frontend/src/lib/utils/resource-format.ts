export function formatCpuLimit(value: number): string {
	if (!Number.isFinite(value) || value < 0) return '-';
	return Number(value.toFixed(2)).toString();
}
