import { q as sanitize_slots, r as fallback, k as attr_class, w as clsx, m as escape_html, d as slot, t as bind_props } from './renderer-EjaZHhrY.js';

function PageHeader($$renderer, $$props) {
  const $$slots = sanitize_slots($$props);
  $$renderer.component(($$renderer2) => {
    let title = fallback($$props["title"], "");
    let description = fallback($$props["description"], "");
    let className = fallback($$props["className"], "");
    $$renderer2.push(`<header${attr_class(clsx(`mb-6 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between ${className}`.trim()))}><div class="min-w-0"><h1 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">${escape_html(title)}</h1> `);
    if (description) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">${escape_html(description)}</p>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> `);
    if ($$slots.meta) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="mt-3 flex flex-wrap items-center gap-2"><!--[-->`);
      slot($$renderer2, $$props, "meta", {});
      $$renderer2.push(`<!--]--></div>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--></div> `);
    if ($$slots.actions) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="flex shrink-0 flex-col gap-2 sm:flex-row sm:items-center"><!--[-->`);
      slot($$renderer2, $$props, "actions", {});
      $$renderer2.push(`<!--]--></div>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--></header>`);
    bind_props($$props, { title, description, className });
  });
}

export { PageHeader as P };
//# sourceMappingURL=PageHeader-CxbUKF9J.js.map
