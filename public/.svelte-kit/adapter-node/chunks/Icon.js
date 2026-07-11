import { g as getContext, o as attributes, k as clsx, e as ensure_array_like, p as element, q as derived } from "./renderer.js";
/**
 * @file
 * @license @lucide/svelte v1.24.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */
const defaultAttributes = {
  xmlns: "http://www.w3.org/2000/svg",
  width: 24,
  height: 24,
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  "stroke-width": 2,
  "stroke-linecap": "round",
  "stroke-linejoin": "round"
};
/**
 * @file
 * @license @lucide/svelte v1.24.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */
const hasA11yProp = (props) => {
  for (const prop in props) {
    if (prop.startsWith("aria-") || prop === "role" || prop === "title") {
      return true;
    }
  }
  return false;
};
/**
 * @file
 * @license @lucide/svelte v1.24.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */
const LucideContext = Symbol("lucide-context");
const getLucideContext = () => getContext(LucideContext);
function Icon($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    const globalProps = getLucideContext() ?? {};
    const {
      name,
      color = globalProps.color ?? "currentColor",
      size = globalProps.size ?? 24,
      strokeWidth = globalProps.strokeWidth ?? 2,
      absoluteStrokeWidth = globalProps.absoluteStrokeWidth ?? false,
      iconNode = [],
      children,
      $$slots,
      $$events,
      ...props
    } = $$props;
    const calculatedStrokeWidth = derived(() => absoluteStrokeWidth ? Number(strokeWidth) * 24 / Number(size) : strokeWidth);
    $$renderer2.push(`<svg${attributes(
      {
        ...defaultAttributes,
        ...!children && !hasA11yProp(props) && { "aria-hidden": "true" },
        ...props,
        width: size,
        height: size,
        stroke: color,
        "stroke-width": calculatedStrokeWidth(),
        class: clsx([
          "lucide-icon lucide",
          globalProps.class,
          name && `lucide-${name}`,
          props.class
        ])
      },
      void 0,
      void 0,
      void 0,
      3
    )}><!--[-->`);
    const each_array = ensure_array_like(iconNode);
    for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
      let [tag, attrs] = each_array[$$index];
      element($$renderer2, tag, () => {
        $$renderer2.push(`${attributes({ ...attrs }, void 0, void 0, void 0, 3)}`);
      });
    }
    $$renderer2.push(`<!--]-->`);
    children?.($$renderer2);
    $$renderer2.push(`<!----></svg>`);
  });
}
export {
  Icon as I
};
