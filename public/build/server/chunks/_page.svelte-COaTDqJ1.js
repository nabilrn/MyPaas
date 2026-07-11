import { c as store_get, h as head, f as unsubscribe_stores, av as ssr_context } from './renderer-EjaZHhrY.js';
import { p as page } from './stores-BdUEhWdH.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';
import './toast-4wUWO2xn.js';
import { a as projectHost } from './urls-CHcPIehM.js';
import './index-CjGMQA9M.js';

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

export { _page as default };
//# sourceMappingURL=_page.svelte-COaTDqJ1.js.map
