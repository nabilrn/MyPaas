import { c as store_get, f as unsubscribe_stores, h as head, m as escape_html, k as attr_class, l as stringify, j as ensure_array_like, n as attr, p as spread_props } from './renderer-EjaZHhrY.js';
import { p as page } from './stores-BdUEhWdH.js';
import { S as StatusBadge } from './StatusBadge-COIKZItd.js';
import { A as ActionButton } from './ActionButton-WfeBQzx6.js';
import { I as IconButton } from './IconButton-V8WSTVXe.js';
import { T as TableShell, P as Pagination } from './TableShell-CE6dcEMn.js';
import './toast-4wUWO2xn.js';
import { C as Chevron_up, a as Chevron_down } from './chevron-up-DR5OZgDX.js';
import { I as Icon } from './Icon-9ZFIo3zS.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';
import './EmptyState-DcYSSZfa.js';
import './plus-BgLmOiR6.js';
import './index-CjGMQA9M.js';

function Rotate_ccw($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    [
      "path",
      { "d": "M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" }
    ],
    ["path", { "d": "M3 3v5h5" }]
  ];
  Icon($$renderer, spread_props([{ name: "rotate-ccw" }, props, { iconNode }]));
}
function normalizeDeploymentFocus(value) {
  return value?.trim() ?? "";
}
function expandFocusedDeployment(current, focusId) {
  return focusId ? /* @__PURE__ */ new Set([...current, focusId]) : new Set(current);
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let visibleDeployments, activeCount, healthyCount, failedCount;
    const pageSize = 20;
    let deployments = [];
    let loading = true;
    let expanded = /* @__PURE__ */ new Set();
    let rollingBackId = "";
    let confirmRollbackId = "";
    let currentPage = 0;
    let hasNext = false;
    let focusId = "";
    let appliedFocusId = "";
    function isPipelineActive(status) {
      return status === "queued" || status === "cloning" || status === "building" || status === "starting";
    }
    function formatDuration(start, end) {
      if (!end) return "-";
      const ms = new Date(end).getTime() - new Date(start).getTime();
      const s = Math.floor(ms / 1e3);
      return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
    }
    function formatDate(value) {
      return new Date(value).toLocaleString();
    }
    visibleDeployments = deployments.slice(0, pageSize);
    activeCount = visibleDeployments.filter((item) => isPipelineActive(item.status)).length;
    healthyCount = visibleDeployments.filter((item) => ["running", "stopped", "rolled_back"].includes(item.status)).length;
    failedCount = visibleDeployments.filter((item) => item.status === "failed").length;
    focusId = normalizeDeploymentFocus(store_get($$store_subs ??= {}, "$page", page).url.searchParams.get("focus"));
    if (focusId !== appliedFocusId) {
      appliedFocusId = focusId;
      if (focusId) {
        currentPage = 0;
        expanded = expandFocusedDeployment(expanded, focusId);
      }
    }
    let $$settled = true;
    let $$inner_renderer;
    function $$render_inner($$renderer3) {
      head("h9sfdg", $$renderer3, ($$renderer4) => {
        $$renderer4.title(($$renderer5) => {
          $$renderer5.push(`<title>Deployments · MyPaas</title>`);
        });
      });
      TableShell($$renderer3, {
        title: "Deployment history",
        description: "Latest build attempts, commit metadata, and rollback actions.",
        loading,
        loadingRows: 3,
        error: "",
        empty: deployments.length === 0,
        emptyTitle: "No deployments yet.",
        emptyDescription: "Trigger a deploy from the project actions panel to create the first deployment record.",
        contentClass: "",
        children: ($$renderer4) => {
          $$renderer4.push(`<div class="grid border-b border-gray-100 bg-gray-50/60 dark:border-gray-800 dark:bg-gray-900/50 sm:grid-cols-3"><div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-b-0 sm:border-r"><p class="metric-label">Active pipeline</p> <p class="mt-1 font-mono text-lg font-semibold text-gray-950 dark:text-white">${escape_html(activeCount)}</p></div> <div class="border-b border-gray-100 px-5 py-3 dark:border-gray-800 sm:border-b-0 sm:border-r"><p class="metric-label">Recoverable targets</p> <p class="mt-1 font-mono text-lg font-semibold text-gray-950 dark:text-white">${escape_html(healthyCount)}</p></div> <div class="px-5 py-3"><p class="metric-label">Failed attempts</p> <p${attr_class(`mt-1 font-mono text-lg font-semibold ${stringify(failedCount > 0 ? "text-red-600 dark:text-red-300" : "text-gray-950 dark:text-white")}`)}>${escape_html(failedCount)}</p></div></div> <div class="divide-y divide-gray-100 dark:divide-gray-800"><!--[-->`);
          const each_array = ensure_array_like(visibleDeployments);
          for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
            let d = each_array[$$index];
            $$renderer4.push(`<div${attr("id", `deployment-${d.id}`)}${attr_class(`scroll-mt-6 px-5 py-4 transition-colors ${focusId === d.id ? "bg-brand-50/60 dark:bg-brand-900/20" : ""}`)}${attr("aria-current", focusId === d.id ? "true" : void 0)}><div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_9rem_8rem_auto] lg:items-center"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><span class="font-mono text-sm font-semibold text-gray-950 dark:text-white">${escape_html(d.commitSha?.slice(0, 8) ?? "-")}</span> `);
            StatusBadge($$renderer4, { status: d.status });
            $$renderer4.push(`<!----> <span class="rounded border border-gray-200 px-1.5 py-0.5 text-[11px] font-medium capitalize text-gray-500 dark:border-gray-800 dark:text-gray-400">${escape_html(d.triggeredBy)}</span></div> <p class="mt-1 truncate text-sm text-gray-600 dark:text-gray-400">${escape_html(d.commitMessage || "No commit message")}</p> `);
            if (d.errorMsg) {
              $$renderer4.push("<!--[0-->");
              $$renderer4.push(`<p class="mt-1 text-xs text-red-600 dark:text-red-300">${escape_html(d.errorMsg)}</p>`);
            } else {
              $$renderer4.push("<!--[-1-->");
            }
            $$renderer4.push(`<!--]--></div> <p class="text-xs text-gray-500 dark:text-gray-400">${escape_html(formatDate(d.startedAt))}</p> <p class="font-mono text-xs text-gray-500 dark:text-gray-400">${escape_html(formatDuration(d.startedAt, d.finishedAt))}</p> <div class="flex shrink-0 gap-2 lg:justify-end">`);
            IconButton($$renderer4, {
              label: `${expanded.has(d.id) ? "Hide" : "Show"} build log for ${d.commitSha?.slice(0, 8) ?? "deployment"}`,
              children: ($$renderer5) => {
                if (expanded.has(d.id)) {
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
            $$renderer4.push(`<!----> `);
            if (d.status === "running" || d.status === "stopped") {
              $$renderer4.push("<!--[0-->");
              if (confirmRollbackId === d.id) {
                $$renderer4.push("<!--[0-->");
                ActionButton($$renderer4, {
                  variant: "ghost",
                  size: "xs",
                  children: ($$renderer5) => {
                    $$renderer5.push(`<!---->Cancel`);
                  },
                  $$slots: { default: true }
                });
                $$renderer4.push(`<!----> `);
                ActionButton($$renderer4, {
                  variant: "danger",
                  size: "xs",
                  disabled: rollingBackId !== "",
                  loading: rollingBackId === d.id,
                  loadingLabel: "Rolling back...",
                  children: ($$renderer5) => {
                    $$renderer5.push(`<!---->Confirm rollback`);
                  },
                  $$slots: { default: true }
                });
                $$renderer4.push(`<!---->`);
              } else {
                $$renderer4.push("<!--[-1-->");
                IconButton($$renderer4, {
                  label: `Rollback deployment ${d.commitSha?.slice(0, 8) ?? d.id}`,
                  variant: "danger",
                  disabled: rollingBackId !== "",
                  children: ($$renderer5) => {
                    Rotate_ccw($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                  },
                  $$slots: { default: true }
                });
              }
              $$renderer4.push(`<!--]-->`);
            } else {
              $$renderer4.push("<!--[-1-->");
            }
            $$renderer4.push(`<!--]--></div></div> `);
            if (expanded.has(d.id)) {
              $$renderer4.push("<!--[0-->");
              $$renderer4.push(`<div class="mt-4 overflow-hidden rounded-md border border-gray-800 bg-gray-950"><div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-800 px-3 py-2"><p class="font-mono text-[11px] font-semibold uppercase tracking-wider text-gray-300">Build output</p> <p class="text-[11px] text-gray-500">`);
              if (isPipelineActive(d.status)) {
                $$renderer4.push("<!--[0-->");
                $$renderer4.push(`${escape_html(d.buildLog ? "Live, refreshes every 3 seconds" : "Waiting for output")}`);
              } else {
                $$renderer4.push("<!--[-1-->");
                $$renderer4.push(`${escape_html(d.buildLog ? "Final output" : "No output captured")}`);
              }
              $$renderer4.push(`<!--]--></p></div> `);
              if (d.buildLog) {
                $$renderer4.push("<!--[0-->");
                $$renderer4.push(`<pre class="max-h-80 overflow-auto p-3 text-xs leading-5 text-gray-100">${escape_html(d.buildLog)}</pre>`);
              } else {
                $$renderer4.push("<!--[-1-->");
                $$renderer4.push(`<div class="px-3 py-6 text-center text-xs leading-5 text-gray-400"${attr("role", isPipelineActive(d.status) ? "status" : void 0)}>${escape_html(isPipelineActive(d.status) ? `Pipeline is ${d.status}. Build output will appear here automatically.` : "This deployment did not produce build output.")}</div>`);
              }
              $$renderer4.push(`<!--]--></div>`);
            } else {
              $$renderer4.push("<!--[-1-->");
            }
            $$renderer4.push(`<!--]--></div>`);
          }
          $$renderer4.push(`<!--]--></div>`);
        },
        $$slots: {
          default: true,
          notice: ($$renderer4) => {
            {
              {
                $$renderer4.push("<!--[-1-->");
              }
              $$renderer4.push(`<!--]-->`);
            }
          },
          footer: ($$renderer4) => {
            {
              Pagination($$renderer4, {
                pageSize,
                totalShown: visibleDeployments.length,
                hasNext,
                loading,
                label: "Deployments",
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
    }
    do {
      $$settled = true;
      $$inner_renderer = $$renderer2.copy();
      $$render_inner($$inner_renderer);
    } while (!$$settled);
    $$renderer2.subsume($$inner_renderer);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}

export { _page as default };
//# sourceMappingURL=_page.svelte-DHhwNdh5.js.map
