import { h as head, k as attr_class, l as stringify, m as escape_html, n as attr, j as ensure_array_like, p as spread_props } from './renderer-EjaZHhrY.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';
import { I as IconButton } from './IconButton-V8WSTVXe.js';
import { S as SectionPanel } from './SectionPanel-DrTEgZNe.js';
import './toast-4wUWO2xn.js';
import { I as Icon } from './Icon-9ZFIo3zS.js';
import { U as Upload } from './upload-BW7UXErS.js';
import { T as Trash_2 } from './trash-2-KJN1o-8B.js';
import { R as Refresh_cw } from './refresh-cw-JzODzqwL.js';
import './index-CjGMQA9M.js';

function Copy($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    [
      "rect",
      {
        "width": "14",
        "height": "14",
        "x": "8",
        "y": "8",
        "rx": "2",
        "ry": "2"
      }
    ],
    [
      "path",
      {
        "d": "M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"
      }
    ]
  ];
  Icon($$renderer, spread_props([{ name: "copy" }, props, { iconNode }]));
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let services, filteredLogs, renderedLogs, streamDescription;
    const renderLimit = 1e3;
    let logs = [];
    let reloadingHistory = false;
    let filter = "";
    let selectedService = "all";
    services = [
      "all",
      ...Array.from(new Set(logs.map((log) => log.service))).sort()
    ];
    filteredLogs = logs.filter((log) => {
      const query = filter.trim().toLowerCase();
      const matchesFilter = query === "" || log.line.toLowerCase().includes(query) || log.service.toLowerCase().includes(query);
      return matchesFilter;
    });
    renderedLogs = filteredLogs.length > renderLimit ? filteredLogs.slice(-renderLimit) : filteredLogs;
    filteredLogs.length - renderedLogs.length;
    streamDescription = "Connecting to the project log stream.";
    head("197zwn7", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>Logs · MyPaas</title>`);
      });
    });
    $$renderer2.push(`<div class="flex h-[calc(100vh-16rem)] min-h-[32rem] flex-col">`);
    SectionPanel($$renderer2, {
      title: "Log stream",
      description: streamDescription,
      className: "flex min-h-0 flex-1 flex-col",
      contentClass: "flex min-h-0 flex-1 flex-col gap-3 p-4",
      children: ($$renderer3) => {
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> <div class="scrollbar-thin relative flex-1 overflow-auto rounded-md border border-gray-800 bg-gray-950 p-4 font-mono text-xs leading-5 text-gray-100 shadow-sm" aria-live="polite">`);
        {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<div class="space-y-2"><!--[-->`);
          const each_array = ensure_array_like([1, 2, 3, 4, 5, 6]);
          for (let $$index_1 = 0, $$length = each_array.length; $$index_1 < $$length; $$index_1++) {
            each_array[$$index_1];
            $$renderer3.push(`<div class="grid grid-cols-[4.5rem_minmax(0,1fr)] gap-2 sm:grid-cols-[5.5rem_7rem_minmax(0,1fr)]"><span class="h-4 animate-pulse rounded bg-gray-800"></span> <span class="h-4 animate-pulse rounded bg-gray-800"></span> <span class="h-4 animate-pulse rounded bg-gray-800"></span></div>`);
          }
          $$renderer3.push(`<!--]--></div>`);
        }
        $$renderer3.push(`<!--]--></div> <div class="flex flex-wrap items-center justify-between gap-3 text-xs text-gray-500 dark:text-gray-400"><div>Showing ${escape_html(filteredLogs.length)} of ${escape_html(logs.length)} lines. Keeping latest 5000 lines in memory.</div> <div class="flex items-center gap-3">`);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> `);
        IconButton($$renderer3, {
          label: "Clear local log view",
          variant: "ghost",
          type: "button",
          disabled: logs.length === 0,
          children: ($$renderer4) => {
            Trash_2($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
          },
          $$slots: { default: true }
        });
        $$renderer3.push(`<!----> `);
        IconButton($$renderer3, {
          label: "Reload log history",
          variant: "brand",
          type: "button",
          loading: reloadingHistory,
          children: ($$renderer4) => {
            Refresh_cw($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
          },
          $$slots: { default: true }
        });
        $$renderer3.push(`<!----></div></div>`);
      },
      $$slots: {
        default: true,
        actions: ($$renderer3) => {
          {
            $$renderer3.push(`<div class="flex flex-col gap-2 sm:flex-row sm:items-center"><span class="inline-flex min-h-9 items-center gap-1.5 rounded-md border border-gray-200 bg-gray-50 px-2.5 text-xs font-medium text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300"><span${attr_class(`h-1.5 w-1.5 rounded-full ${stringify("bg-amber-500")}`)}></span> ${escape_html(filteredLogs.length)} visible</span> <input type="search"${attr("value", filter)} placeholder="Filter logs" class="field h-9 w-full sm:w-56"/> `);
            $$renderer3.select({ value: selectedService, class: "field h-9" }, ($$renderer4) => {
              $$renderer4.push(`<!--[-->`);
              const each_array_2 = ensure_array_like(services);
              for (let $$index = 0, $$length = each_array_2.length; $$index < $$length; $$index++) {
                let service = each_array_2[$$index];
                $$renderer4.option({ value: service }, ($$renderer5) => {
                  $$renderer5.push(`${escape_html(service === "all" ? "All services" : service)}`);
                });
              }
              $$renderer4.push(`<!--]-->`);
            });
            $$renderer3.push(` `);
            IconButton($$renderer3, {
              label: "Copy visible logs",
              variant: "default",
              disabled: filteredLogs.length === 0,
              children: ($$renderer4) => {
                {
                  $$renderer4.push("<!--[-1-->");
                  Copy($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
                }
                $$renderer4.push(`<!--]-->`);
              },
              $$slots: { default: true }
            });
            $$renderer3.push(`<!----> `);
            IconButton($$renderer3, {
              label: "Download visible logs",
              variant: "default",
              disabled: filteredLogs.length === 0,
              children: ($$renderer4) => {
                Upload($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
              },
              $$slots: { default: true }
            });
            $$renderer3.push(`<!----></div>`);
          }
        }
      }
    });
    $$renderer2.push(`<!----></div>`);
  });
}

export { _page as default };
//# sourceMappingURL=_page.svelte-Ci608TqH.js.map
