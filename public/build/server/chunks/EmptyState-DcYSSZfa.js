import { r as fallback, k as attr_class, m as escape_html, n as attr, t as bind_props, p as spread_props, l as stringify } from './renderer-EjaZHhrY.js';
import { I as Icon } from './Icon-9ZFIo3zS.js';
import { P as Plus } from './plus-BgLmOiR6.js';

function Package_open($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    ["path", { "d": "M12 22v-9" }],
    [
      "path",
      {
        "d": "M15.17 2.21a1.67 1.67 0 0 1 1.63 0L21 4.57a1.93 1.93 0 0 1 0 3.36L8.82 14.79a1.655 1.655 0 0 1-1.64 0L3 12.43a1.93 1.93 0 0 1 0-3.36z"
      }
    ],
    [
      "path",
      {
        "d": "M20 13v3.87a2.06 2.06 0 0 1-1.11 1.83l-6 3.08a1.93 1.93 0 0 1-1.78 0l-6-3.08A2.06 2.06 0 0 1 4 16.87V13"
      }
    ],
    [
      "path",
      {
        "d": "M21 12.43a1.93 1.93 0 0 0 0-3.36L8.83 2.2a1.64 1.64 0 0 0-1.63 0L3 4.57a1.93 1.93 0 0 0 0 3.36l12.18 6.86a1.636 1.636 0 0 0 1.63 0z"
      }
    ]
  ];
  Icon($$renderer, spread_props([{ name: "package-open" }, props, { iconNode }]));
}
function EmptyState($$renderer, $$props) {
  let title = $$props["title"];
  let description = fallback($$props["description"], "");
  let actionLabel = fallback($$props["actionLabel"], "");
  let actionHref = fallback($$props["actionHref"], "");
  let compact = fallback($$props["compact"], false);
  $$renderer.push(`<div${attr_class(`flex flex-col items-center justify-center px-6 text-center ${stringify(compact ? "py-10" : "py-16")}`)}><div${attr_class(`mb-4 flex ${stringify(compact ? "h-10 w-10" : "h-12 w-12")} items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-400 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-500`)}>`);
  Package_open($$renderer, {
    class: compact ? "h-5 w-5" : "h-6 w-6",
    "aria-hidden": "true"
  });
  $$renderer.push(`<!----></div> <h3${attr_class(`${stringify(compact ? "text-sm" : "text-base")} font-semibold text-gray-900 dark:text-white`)}>${escape_html(title)}</h3> `);
  if (description) {
    $$renderer.push("<!--[0-->");
    $$renderer.push(`<p class="mt-1 max-w-md text-sm text-gray-500 dark:text-gray-400">${escape_html(description)}</p>`);
  } else {
    $$renderer.push("<!--[-1-->");
  }
  $$renderer.push(`<!--]--> `);
  if (actionLabel && actionHref) {
    $$renderer.push("<!--[0-->");
    $$renderer.push(`<a${attr("href", actionHref)} class="mt-4 inline-flex min-h-9 items-center gap-2 rounded-md bg-brand-700 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-brand-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:bg-brand-500 dark:text-gray-950 dark:hover:bg-brand-100 dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950">`);
    Plus($$renderer, { class: "h-4 w-4", "aria-hidden": "true" });
    $$renderer.push(`<!----> ${escape_html(actionLabel)}</a>`);
  } else {
    $$renderer.push("<!--[-1-->");
  }
  $$renderer.push(`<!--]--></div>`);
  bind_props($$props, { title, description, actionLabel, actionHref, compact });
}

export { EmptyState as E };
//# sourceMappingURL=EmptyState-DcYSSZfa.js.map
