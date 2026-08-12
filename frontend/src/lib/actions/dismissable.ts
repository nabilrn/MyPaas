type DismissableOptions = {
	enabled?: boolean;
	onDismiss: () => void;
};

export function dismissable(node: HTMLElement, options: DismissableOptions) {
	let current = options;

	function handlePointerDown(event: PointerEvent) {
		if (!current.enabled) return;
		if (!(event.target instanceof Node)) return;
		if (node.contains(event.target)) return;
		current.onDismiss();
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (!current.enabled || event.key !== 'Escape') return;
		current.onDismiss();
	}

	document.addEventListener('pointerdown', handlePointerDown, true);
	document.addEventListener('keydown', handleKeyDown);

	return {
		update(next: DismissableOptions) {
			current = next;
		},
		destroy() {
			document.removeEventListener('pointerdown', handlePointerDown, true);
			document.removeEventListener('keydown', handleKeyDown);
		}
	};
}
