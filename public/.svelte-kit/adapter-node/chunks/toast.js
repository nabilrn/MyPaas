import { w as writable } from "./index.js";
function createToastStore() {
  const { subscribe, update } = writable([]);
  function add(kind, message, durationMs = 4e3) {
    const id = crypto.randomUUID();
    update((list) => [...list, { id, kind, message }]);
    setTimeout(() => remove(id), durationMs);
  }
  function remove(id) {
    update((list) => list.filter((t) => t.id !== id));
  }
  return {
    subscribe,
    success: (msg) => add("success", msg),
    error: (msg) => add("error", msg, 6e3),
    warning: (msg) => add("warning", msg),
    info: (msg) => add("info", msg),
    remove
  };
}
const toast = createToastStore();
export {
  toast as t
};
