import { i as spread_props } from "./renderer.js";
import { I as Icon } from "./Icon.js";
function Plus($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [["path", { "d": "M5 12h14" }], ["path", { "d": "M12 5v14" }]];
  Icon($$renderer, spread_props([{ name: "plus" }, props, { iconNode }]));
}
export {
  Plus as P
};
