export interface InfoDisclosureState {
  expanded: boolean;
  open: () => void;
  close: () => void;
  toggle: () => void;
}

export function infoDisclosureState(initial = false): InfoDisclosureState {
  const state: InfoDisclosureState = {
    expanded: initial,
    open() {
      state.expanded = true;
    },
    close() {
      state.expanded = false;
    },
    toggle() {
      state.expanded = !state.expanded;
    },
  };
  return state;
}
