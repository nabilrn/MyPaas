import { describe, expect, it } from 'vitest';
import dialog from '../components/ConfirmActionDialog.svelte?raw';

describe('confirmation dialog contract', () => {
	it('keeps confirmation accessible and reusable', () => {
		expect(dialog).toContain('role="dialog"');
		expect(dialog).toContain('aria-modal="true"');
		expect(dialog).toContain('tabindex="-1"');
		expect(dialog).toContain('trapDialogFocus');
		expect(dialog).toContain("event.key === 'Escape'");
		expect(dialog).toContain('returnFocus');
		expect(dialog).toContain("createEventDispatcher<{ confirm: void; cancel: void }>()");
	});

	it('uses shared action controls and workspace strokes', () => {
		expect(dialog).toContain('ActionButton');
		expect(dialog).toContain('workspace-divider');
		expect(dialog).toContain("variant: 'primary' | 'danger'");
		expect(dialog).not.toContain('window.confirm');
	});
});
