import { at as noop } from './renderer-EjaZHhrY.js';
import './root-d39-B9S_.js';

const is_legacy = noop.toString().includes("$$") || /function \w+\(\) \{\}/.test(noop.toString());
const placeholder_url = "a:";
if (is_legacy) {
  ({
    url: new URL(placeholder_url)
  });
}
//# sourceMappingURL=state.svelte-xyT85yFW.js.map
