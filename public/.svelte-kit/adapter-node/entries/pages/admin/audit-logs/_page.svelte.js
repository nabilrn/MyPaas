import { i as spread_props, h as head, e as ensure_array_like, c as escape_html, f as attr, a as attr_class } from "../../../../chunks/renderer.js";
import { I as IconButton } from "../../../../chunks/IconButton.js";
import { P as PageHeader } from "../../../../chunks/PageHeader.js";
import { T as TableShell, P as Pagination } from "../../../../chunks/TableShell.js";
import { R as Refresh_cw } from "../../../../chunks/refresh-cw.js";
import { I as Icon } from "../../../../chunks/Icon.js";
import { C as Chevron_up, a as Chevron_down } from "../../../../chunks/chevron-up.js";
function User($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    ["path", { "d": "M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" }],
    ["circle", { "cx": "12", "cy": "7", "r": "4" }]
  ];
  Icon($$renderer, spread_props([{ name: "user" }, props, { iconNode }]));
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let visibleRows;
    const pageSize = 25;
    let rows = [];
    let loading = true;
    let error = "";
    let expanded = /* @__PURE__ */ new Set();
    let currentPage = 0;
    let hasNext = false;
    function statusClass(status) {
      const code = Number(status);
      if (!Number.isFinite(code)) {
        return "border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300";
      }
      if (code >= 500) {
        return "border-red-500/30 bg-red-50 text-red-700 dark:border-red-500/40 dark:bg-red-950/30 dark:text-red-200";
      }
      if (code >= 400) {
        return "border-yellow-500/30 bg-yellow-50 text-yellow-800 dark:border-yellow-500/40 dark:bg-yellow-950/30 dark:text-yellow-100";
      }
      if (code >= 200 && code < 300) {
        return "border-brand-500/30 bg-brand-50 text-brand-900 dark:border-brand-500/40 dark:bg-brand-500/10 dark:text-brand-100";
      }
      return "border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300";
    }
    function formatDateTime(value) {
      return new Date(value).toLocaleString();
    }
    visibleRows = rows.slice(0, pageSize);
    let $$settled = true;
    let $$inner_renderer;
    function $$render_inner($$renderer3) {
      head("j8xf1e", $$renderer3, ($$renderer4) => {
        $$renderer4.title(($$renderer5) => {
          $$renderer5.push(`<title>Audit Logs · MyPaas Admin</title>`);
        });
      });
      $$renderer3.push(`<div class="page-shell py-6">`);
      PageHeader($$renderer3, {
        title: "Audit logs",
        description: "Recent authenticated changes across projects, deployments, env vars, and admin users.",
        $$slots: {
          actions: ($$renderer4) => {
            {
              IconButton($$renderer4, {
                label: "Refresh audit logs",
                variant: "brand",
                loading,
                children: ($$renderer5) => {
                  Refresh_cw($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                },
                $$slots: { default: true }
              });
              $$renderer4.push(`<!----> `);
              IconButton($$renderer4, {
                label: "User whitelist",
                href: "/admin/users",
                variant: "default",
                children: ($$renderer5) => {
                  User($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                },
                $$slots: { default: true }
              });
              $$renderer4.push(`<!---->`);
            }
          }
        }
      });
      $$renderer3.push(`<!----> `);
      TableShell($$renderer3, {
        title: "Event stream",
        description: "Review what changed, which resource was touched, and the response code returned by the control plane.",
        loading,
        loadingRows: 3,
        error,
        empty: rows.length === 0,
        emptyTitle: "No audit logs yet.",
        emptyDescription: "Authenticated admin and deployment events will appear here after changes are made.",
        children: ($$renderer4) => {
          $$renderer4.push(`<table class="min-w-full divide-y divide-gray-100 dark:divide-gray-800"><thead><tr class="bg-gray-50/70 dark:bg-gray-900/70"><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Action</th><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Resource</th><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Status</th><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Time</th><th class="px-5 py-3"></th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-gray-800"><!--[-->`);
          const each_array = ensure_array_like(visibleRows);
          for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
            let row = each_array[$$index];
            $$renderer4.push(`<tr class="align-top hover:bg-gray-50/80 dark:hover:bg-gray-900/70"><td class="px-5 py-4"><p class="font-mono text-sm font-medium text-gray-950 dark:text-white">${escape_html(row.action)}</p> <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">${escape_html(row.ipAddress ?? "unknown ip")}</p></td><td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300">${escape_html(row.resourceType ?? "—")} `);
            if (row.resourceId) {
              $$renderer4.push("<!--[0-->");
              $$renderer4.push(`<span class="block max-w-56 truncate font-mono text-xs text-gray-400"${attr("title", row.resourceId)}>${escape_html(row.resourceId)}</span>`);
            } else {
              $$renderer4.push("<!--[-1-->");
            }
            $$renderer4.push(`<!--]--></td><td class="px-5 py-4 text-sm text-gray-600 dark:text-gray-300"><span${attr_class(`inline-flex rounded-md border px-2 py-1 font-mono text-xs font-medium ${statusClass(row.metadata.status)}`)}>${escape_html(String(row.metadata.status ?? "—"))}</span></td><td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">${escape_html(formatDateTime(row.createdAt))}</td><td class="px-5 py-4 text-right">`);
            IconButton($$renderer4, {
              label: `${expanded.has(row.id) ? "Hide" : "Show"} audit log details`,
              variant: "ghost",
              children: ($$renderer5) => {
                if (expanded.has(row.id)) {
                  $$renderer5.push("<!--[0-->");
                  Chevron_up($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                } else {
                  $$renderer5.push("<!--[-1-->");
                  Chevron_down($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                }
                $$renderer5.push(`<!--]-->`);
              },
              $$slots: { default: true }
            });
            $$renderer4.push(`<!----></td></tr> `);
            if (expanded.has(row.id)) {
              $$renderer4.push("<!--[0-->");
              $$renderer4.push(`<tr class="bg-gray-50 dark:bg-gray-950/40"><td colspan="5" class="px-5 py-4"><div class="grid gap-3 lg:grid-cols-[14rem_minmax(0,1fr)]"><div class="space-y-2 text-xs text-gray-500 dark:text-gray-400"><p><span class="block font-medium text-gray-700 dark:text-gray-200">IP address</span> <span class="font-mono">${escape_html(row.ipAddress ?? "unknown")}</span></p> <p><span class="block font-medium text-gray-700 dark:text-gray-200">User agent</span> <span class="line-clamp-4 break-words">${escape_html(row.userAgent ?? "unknown")}</span></p></div> <pre class="max-h-80 overflow-auto rounded-md border border-gray-800 bg-gray-950 p-3 text-xs leading-5 text-gray-100">${escape_html(JSON.stringify(row.metadata, null, 2))}</pre></div></td></tr>`);
            } else {
              $$renderer4.push("<!--[-1-->");
            }
            $$renderer4.push(`<!--]-->`);
          }
          $$renderer4.push(`<!--]--></tbody></table>`);
        },
        $$slots: {
          default: true,
          footer: ($$renderer4) => {
            {
              Pagination($$renderer4, {
                pageSize,
                totalShown: visibleRows.length,
                hasNext,
                loading,
                label: "Audit logs",
                get page() {
                  return currentPage;
                },
                set page($$value) {
                  currentPage = $$value;
                  $$settled = false;
                }
              });
            }
          }
        }
      });
      $$renderer3.push(`<!----></div>`);
    }
    do {
      $$settled = true;
      $$inner_renderer = $$renderer2.copy();
      $$render_inner($$inner_renderer);
    } while (!$$settled);
    $$renderer2.subsume($$inner_renderer);
  });
}
export {
  _page as default
};
