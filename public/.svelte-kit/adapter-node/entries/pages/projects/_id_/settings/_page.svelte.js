import { r as ssr_context, s as store_get, h as head, u as unsubscribe_stores } from "../../../../../chunks/renderer.js";
import { p as page } from "../../../../../chunks/stores.js";
import "@sveltejs/kit/internal";
import "../../../../../chunks/exports.js";
import "../../../../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../../../../chunks/root.js";
import "../../../../../chunks/state.svelte.js";
/* empty css                                                               */
/* empty css                                                             */
import "../../../../../chunks/toast.js";
import { a as projectHost } from "../../../../../chunks/urls.js";
function onDestroy(fn) {
  /** @type {SSRContext} */
  ssr_context.r.on_destroy(fn);
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    onDestroy(() => {
    });
    projectHost("your-app", store_get($$store_subs ??= {}, "$page", page).url.hostname);
    head("1o60xg9", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>Settings · MyPaas</title>`);
      });
    });
    {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="space-y-4"><div class="surface h-48 animate-pulse"></div> <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]"><div class="surface h-64 animate-pulse"></div> <div class="surface h-64 animate-pulse"></div></div></div>`);
    }
    $$renderer2.push(`<!--]--> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]-->`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}
export {
  _page as default
};
