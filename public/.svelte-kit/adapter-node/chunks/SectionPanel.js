import { n as sanitize_slots, j as fallback, a as attr_class, k as clsx, c as escape_html, d as slot, m as bind_props } from "./renderer.js";
function SectionPanel($$renderer, $$props) {
  const $$slots = sanitize_slots($$props);
  $$renderer.component(($$renderer2) => {
    let bodyClass;
    let title = fallback($$props["title"], "");
    let description = fallback($$props["description"], "");
    let padded = fallback($$props["padded"], true);
    let className = fallback($$props["className"], "");
    let contentClass = fallback($$props["contentClass"], "");
    bodyClass = contentClass || (padded ? "p-5" : "");
    $$renderer2.push(`<section${attr_class(clsx(`surface min-w-0 overflow-hidden ${className}`.trim()))}>`);
    if (title || description || $$slots.actions) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="panel-header flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><div class="min-w-0">`);
      if (title) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<h2 class="text-sm font-semibold text-gray-950 dark:text-white">${escape_html(title)}</h2>`);
      } else {
        $$renderer2.push("<!--[-1-->");
      }
      $$renderer2.push(`<!--]--> `);
      if (description) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<p class="mt-0.5 max-w-2xl text-xs text-gray-500 dark:text-gray-400">${escape_html(description)}</p>`);
      } else {
        $$renderer2.push("<!--[-1-->");
      }
      $$renderer2.push(`<!--]--></div> `);
      if ($$slots.actions) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<div class="flex shrink-0 flex-wrap items-center gap-2"><!--[-->`);
        slot($$renderer2, $$props, "actions", {});
        $$renderer2.push(`<!--]--></div>`);
      } else {
        $$renderer2.push("<!--[-1-->");
      }
      $$renderer2.push(`<!--]--></div>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> <div${attr_class(clsx(bodyClass))}><!--[-->`);
    slot($$renderer2, $$props, "default", {});
    $$renderer2.push(`<!--]--></div></section>`);
    bind_props($$props, { title, description, padded, className, contentClass });
  });
}
export {
  SectionPanel as S
};
