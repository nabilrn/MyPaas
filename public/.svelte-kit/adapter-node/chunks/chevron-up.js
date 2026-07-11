import { i as spread_props } from "./renderer.js";
import { I as Icon } from "./Icon.js";
function Chevron_down($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [["path", { "d": "m6 9 6 6 6-6" }]];
  Icon($$renderer, spread_props([{ name: "chevron-down" }, props, { iconNode }]));
}
function Chevron_up($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [["path", { "d": "m18 15-6-6-6 6" }]];
  Icon($$renderer, spread_props([{ name: "chevron-up" }, props, { iconNode }]));
}
export {
  Chevron_up as C,
  Chevron_down as a
};
