import { describe, expect, it } from 'vitest';
import toastComponent from '../components/Toast.svelte?raw';

describe('toast contrast contract', () => {
	it('keeps notifications compact while making them visually distinct from the workspace', () => {
		expect(toastComponent).toContain('border-gray-300 bg-white');
		expect(toastComponent).toContain('dark:bg-neutral-900');
		expect(toastComponent).toContain('shadow-[0_10px_30px');
		expect(toastComponent).toContain('accentStyles');
		expect(toastComponent).toContain('iconSurfaceStyles');
		expect(toastComponent).toContain("role={t.kind === 'error' ? 'alert' : 'status'}");
		expect(toastComponent).not.toContain('bg-[var(--app-surface)]');
	});
});
