import { w as writable } from './index-CjGMQA9M.js';

function resolveInitial() {
  return "light";
}
function createThemeStore() {
  const { subscribe, set, update } = writable(resolveInitial());
  return {
    subscribe,
    toggle() {
      update((t) => {
        const next = t === "light" ? "dark" : "light";
        return next;
      });
    },
    set(theme2) {
      set(theme2);
    }
  };
}
const theme = createThemeStore();

export { theme as t };
//# sourceMappingURL=theme-CpZdbxJp.js.map
