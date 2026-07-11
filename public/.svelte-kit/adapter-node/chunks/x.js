import { i as spread_props } from "./renderer.js";
import { I as Icon } from "./Icon.js";
function X($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    ["path", { "d": "M18 6 6 18" }],
    ["path", { "d": "m6 6 12 12" }]
  ];
  Icon($$renderer, spread_props([{ name: "x" }, props, { iconNode }]));
}
export {
  X
};
