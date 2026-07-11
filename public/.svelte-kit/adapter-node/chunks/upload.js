import { i as spread_props } from "./renderer.js";
import { I as Icon } from "./Icon.js";
function Upload($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    ["path", { "d": "M12 3v12" }],
    ["path", { "d": "m17 8-5-5-5 5" }],
    ["path", { "d": "M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" }]
  ];
  Icon($$renderer, spread_props([{ name: "upload" }, props, { iconNode }]));
}
export {
  Upload as U
};
