import { r as fallback, j as ensure_array_like, n as attr, m as escape_html, t as bind_props, p as spread_props } from './renderer-EjaZHhrY.js';
import { I as Icon } from './Icon-9ZFIo3zS.js';

function Chevron_right($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [["path", { "d": "m9 18 6-6-6-6" }]];
  Icon($$renderer, spread_props([{ name: "chevron-right" }, props, { iconNode }]));
}
function Breadcrumbs($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let items = fallback($$props["items"], () => [], true);
    if (items.length > 0) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<nav aria-label="Breadcrumb" class="mb-4 flex min-w-0 items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400"><!--[-->`);
      const each_array = ensure_array_like(items);
      for (let index = 0, $$length = each_array.length; index < $$length; index++) {
        let item = each_array[index];
        if (index > 0) {
          $$renderer2.push("<!--[0-->");
          Chevron_right($$renderer2, {
            class: "h-3.5 w-3.5 shrink-0 text-gray-300 dark:text-gray-700",
            "aria-hidden": "true"
          });
        } else {
          $$renderer2.push("<!--[-1-->");
        }
        $$renderer2.push(`<!--]--> `);
        if (item.href && index < items.length - 1) {
          $$renderer2.push("<!--[0-->");
          $$renderer2.push(`<a${attr("href", item.href)} class="truncate font-medium hover:text-gray-950 dark:hover:text-white">${escape_html(item.label)}</a>`);
        } else {
          $$renderer2.push("<!--[-1-->");
          $$renderer2.push(`<span class="truncate font-medium text-gray-800 dark:text-gray-200"${attr("aria-current", index === items.length - 1 ? "page" : void 0)}>${escape_html(item.label)}</span>`);
        }
        $$renderer2.push(`<!--]-->`);
      }
      $$renderer2.push(`<!--]--></nav>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]-->`);
    bind_props($$props, { items });
  });
}

export { Breadcrumbs as B };
//# sourceMappingURL=Breadcrumbs-s_0YAf0V.js.map
