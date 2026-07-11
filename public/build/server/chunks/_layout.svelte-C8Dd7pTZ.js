import { c as store_get, j as ensure_array_like, f as unsubscribe_stores } from './renderer-EjaZHhrY.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';
import { p as page } from './stores-BdUEhWdH.js';
import { B as Breadcrumbs } from './Breadcrumbs-s_0YAf0V.js';
import './toast-4wUWO2xn.js';
import './Icon-9ZFIo3zS.js';
import './index-CjGMQA9M.js';

function _layout($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let base, pathname, currentPath, routeActiveHref, activeHref, breadcrumbs;
    const tabs = [
      { label: "Overview", href: "" },
      { label: "Deployments", href: "/deployments" },
      { label: "Logs", href: "/logs" },
      { label: "Metrics", href: "/metrics" },
      { label: "Environment", href: "/env" },
      { label: "Database", href: "/database" },
      { label: "Settings", href: "/settings" }
    ];
    function normalizePath(value) {
      return value.length > 1 && value.endsWith("/") ? value.slice(0, -1) : value;
    }
    function tabPath(href, currentBase) {
      return normalizePath(`${currentBase}${href}`);
    }
    function isTabActive(href, currentPathname, currentBase) {
      const targetPath = tabPath(href, currentBase);
      return href === "" ? currentPathname === targetPath : currentPathname === targetPath || currentPathname.startsWith(`${targetPath}/`);
    }
    base = `/projects/${store_get($$store_subs ??= {}, "$page", page).params.id}`;
    pathname = store_get($$store_subs ??= {}, "$page", page).url.pathname;
    currentPath = normalizePath(pathname);
    routeActiveHref = tabs.slice().reverse().find((t) => isTabActive(t.href, currentPath, base))?.href ?? "";
    activeHref = routeActiveHref;
    tabs.find((tab) => tab.href === activeHref) ?? tabs[0];
    breadcrumbs = [
      { label: "Projects", href: "/projects" },
      { label: "Project" }
    ];
    $$renderer2.push(`<div class="page-shell py-5">`);
    Breadcrumbs($$renderer2, { items: breadcrumbs });
    $$renderer2.push(`<!----> `);
    {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="surface p-5"><div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between"><div class="min-w-0 flex-1"><div class="h-5 w-48 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div> <div class="mt-3 h-3 w-full max-w-xl animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div></div> <div class="h-9 w-40 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div></div> <div class="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4"><!--[-->`);
      const each_array = ensure_array_like([1, 2, 3, 4]);
      for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
        each_array[$$index];
        $$renderer2.push(`<div class="h-12 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>`);
      }
      $$renderer2.push(`<!--]--></div></div>`);
    }
    $$renderer2.push(`<!--]--></div>`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}

export { _layout as default };
//# sourceMappingURL=_layout.svelte-C8Dd7pTZ.js.map
