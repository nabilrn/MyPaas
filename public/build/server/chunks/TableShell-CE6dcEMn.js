import { q as sanitize_slots, r as fallback, m as escape_html, d as slot, j as ensure_array_like, k as attr_class, w as clsx, t as bind_props, n as attr } from './renderer-EjaZHhrY.js';
import { A as ActionButton } from './ActionButton-WfeBQzx6.js';
import { E as EmptyState } from './EmptyState-DcYSSZfa.js';

function ErrorState($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let title = fallback($$props["title"], "Could not load data");
    let message = fallback($$props["message"], "Something went wrong.");
    let retryLabel = fallback($$props["retryLabel"], "Retry");
    $$renderer2.push(`<div class="p-5"><div class="rounded-md border border-red-200 bg-red-50/80 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/25 dark:text-red-200"><p class="font-semibold">${escape_html(title)}</p> <p class="mt-1 max-w-2xl text-red-600 dark:text-red-200/80">${escape_html(message)}</p> <div class="mt-3">`);
    ActionButton($$renderer2, {
      variant: "secondary",
      size: "xs",
      children: ($$renderer3) => {
        $$renderer3.push(`<!---->${escape_html(retryLabel)}`);
      },
      $$slots: { default: true }
    });
    $$renderer2.push(`<!----></div></div></div>`);
    bind_props($$props, { title, message, retryLabel });
  });
}
function Pagination($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let start, end;
    let page = fallback($$props["page"], 0);
    let pageSize = fallback($$props["pageSize"], 20);
    let totalShown = fallback($$props["totalShown"], 0);
    let hasNext = fallback($$props["hasNext"], false);
    let loading = fallback($$props["loading"], false);
    let label = fallback($$props["label"], "Rows");
    start = totalShown === 0 ? 0 : page * pageSize + 1;
    end = page * pageSize + totalShown;
    $$renderer2.push(`<div class="flex flex-col gap-3 border-t border-gray-100 bg-gray-50/70 px-5 py-3 text-sm dark:border-gray-800 dark:bg-gray-900/70 sm:flex-row sm:items-center sm:justify-between" role="navigation"${attr("aria-label", `${label} pagination`)}><p class="text-xs text-gray-500 dark:text-gray-400" aria-live="polite">${escape_html(label)}: ${escape_html(start)}-${escape_html(end)}${escape_html(hasNext ? "+" : "")}</p> <div class="flex items-center gap-2">`);
    ActionButton($$renderer2, {
      disabled: page === 0 || loading,
      ariaLabel: `Previous ${label.toLowerCase()} page`,
      variant: "secondary",
      size: "xs",
      children: ($$renderer3) => {
        $$renderer3.push(`<!---->Previous`);
      },
      $$slots: { default: true }
    });
    $$renderer2.push(`<!----> <span class="min-w-16 text-center text-xs font-medium text-gray-500 dark:text-gray-400" aria-live="polite">Page ${escape_html(page + 1)}</span> `);
    ActionButton($$renderer2, {
      disabled: !hasNext || loading,
      ariaLabel: `Next ${label.toLowerCase()} page`,
      variant: "secondary",
      size: "xs",
      children: ($$renderer3) => {
        $$renderer3.push(`<!---->Next`);
      },
      $$slots: { default: true }
    });
    $$renderer2.push(`<!----></div></div>`);
    bind_props($$props, { page, pageSize, totalShown, hasNext, loading, label });
  });
}
function TableShell($$renderer, $$props) {
  const $$slots = sanitize_slots($$props);
  $$renderer.component(($$renderer2) => {
    let title = fallback($$props["title"], "");
    let description = fallback($$props["description"], "");
    let loading = fallback($$props["loading"], false);
    let error = fallback($$props["error"], "");
    let empty = fallback($$props["empty"], false);
    let emptyTitle = fallback($$props["emptyTitle"], "No rows yet.");
    let emptyDescription = fallback($$props["emptyDescription"], "");
    let loadingRows = fallback($$props["loadingRows"], 3);
    let contentClass = fallback($$props["contentClass"], "overflow-x-auto");
    const skeletonRows = Array.from({ length: loadingRows }, (_, index) => index);
    $$renderer2.push(`<section class="surface min-w-0 overflow-hidden">`);
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
      $$renderer2.push(`<!--]--></div> <div class="flex shrink-0 flex-wrap items-center gap-2"><!--[-->`);
      slot($$renderer2, $$props, "actions", {});
      $$renderer2.push(`<!--]--></div></div>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> `);
    if (loading) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="space-y-3 p-5" aria-busy="true" aria-live="polite"><!--[-->`);
      const each_array = ensure_array_like(skeletonRows);
      for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
        each_array[$$index];
        $$renderer2.push(`<div class="h-12 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800/80"></div>`);
      }
      $$renderer2.push(`<!--]--></div>`);
    } else if (error) {
      $$renderer2.push("<!--[1-->");
      ErrorState($$renderer2, { message: error });
    } else if (empty) {
      $$renderer2.push("<!--[2-->");
      EmptyState($$renderer2, {
        title: emptyTitle,
        description: emptyDescription,
        compact: true
      });
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<!--[-->`);
      slot($$renderer2, $$props, "notice", {});
      $$renderer2.push(`<!--]--> <div${attr_class(clsx(contentClass))}><!--[-->`);
      slot($$renderer2, $$props, "default", {});
      $$renderer2.push(`<!--]--></div> <!--[-->`);
      slot($$renderer2, $$props, "footer", {});
      $$renderer2.push(`<!--]-->`);
    }
    $$renderer2.push(`<!--]--></section>`);
    bind_props($$props, {
      title,
      description,
      loading,
      error,
      empty,
      emptyTitle,
      emptyDescription,
      loadingRows,
      contentClass
    });
  });
}

export { Pagination as P, TableShell as T };
//# sourceMappingURL=TableShell-CE6dcEMn.js.map
