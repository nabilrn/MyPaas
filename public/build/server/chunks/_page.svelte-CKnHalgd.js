import { h as head, m as escape_html, j as ensure_array_like, r as fallback, k as attr_class, w as clsx, n as attr, t as bind_props } from './renderer-EjaZHhrY.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';
import { E as EmptyState } from './EmptyState-DcYSSZfa.js';
import { I as IconButton } from './IconButton-V8WSTVXe.js';
import { S as SectionPanel } from './SectionPanel-DrTEgZNe.js';
import { R as Refresh_cw } from './refresh-cw-JzODzqwL.js';
import './Icon-9ZFIo3zS.js';
import './plus-BgLmOiR6.js';

function CapacityMetricChart($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let safePercent, series, points, linePath, areaPath, toneClass;
    let label = fallback($$props["label"], "");
    let value = fallback($$props["value"], "");
    let detail = fallback($$props["detail"], "");
    let percent = fallback($$props["percent"], 0);
    let tone = fallback($$props["tone"], "neutral");
    let className = fallback($$props["className"], "");
    const chartWidth = 180;
    const chartHeight = 76;
    const sampleShape = [
      0.1,
      0.13,
      0.11,
      0.17,
      0.16,
      0.22,
      0.2,
      0.27,
      0.25,
      0.31,
      0.29,
      0.36,
      0.34,
      0.42,
      0.4,
      0.48
    ];
    function buildSeries(currentPercent) {
      const current = currentPercent / 100;
      return sampleShape.map((sample, index) => {
        if (index === sampleShape.length - 1) return current;
        if (current <= 0) return 0;
        const leadIn = 0.58 + sample * 0.62 + index / sampleShape.length * 0.18;
        return Math.max(0.03, Math.min(0.98, current * leadIn));
      });
    }
    safePercent = Math.max(0, Math.min(100, Number.isFinite(percent) ? percent : 0));
    series = buildSeries(safePercent);
    points = series.map((level, index) => {
      const x = index / Math.max(1, series.length - 1) * chartWidth;
      const y = chartHeight - level * chartHeight;
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    });
    linePath = `M ${points.join(" L ")}`;
    areaPath = `M 0 ${chartHeight} L ${points.join(" L ")} L ${chartWidth} ${chartHeight} Z`;
    toneClass = {
      neutral: {
        dot: "bg-gray-400 dark:bg-gray-500",
        text: "text-gray-600 dark:text-gray-300",
        stroke: "stroke-gray-500 dark:stroke-gray-400",
        fill: "fill-gray-400/10 dark:fill-gray-300/10"
      },
      success: {
        dot: "bg-brand-500",
        text: "text-brand-700 dark:text-brand-100",
        stroke: "stroke-brand-500 dark:stroke-brand-500",
        fill: "fill-brand-500/10 dark:fill-brand-500/15"
      },
      info: {
        dot: "bg-sky-500",
        text: "text-sky-700 dark:text-sky-300",
        stroke: "stroke-sky-500 dark:stroke-sky-300",
        fill: "fill-sky-500/10 dark:fill-sky-300/15"
      },
      warning: {
        dot: "bg-amber-500",
        text: "text-amber-700 dark:text-amber-200",
        stroke: "stroke-amber-500 dark:stroke-amber-300",
        fill: "fill-amber-500/10 dark:fill-amber-300/15"
      },
      danger: {
        dot: "bg-red-500",
        text: "text-red-700 dark:text-red-200",
        stroke: "stroke-red-500 dark:stroke-red-300",
        fill: "fill-red-500/10 dark:fill-red-300/15"
      }
    }[tone];
    $$renderer2.push(`<article${attr_class(clsx(`min-w-0 p-4 ${className}`.trim()))}${attr("aria-label", `${label} ${safePercent.toFixed(0)} percent`)}><div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex items-center gap-2"><span${attr_class(`h-1.5 w-1.5 rounded-full ${toneClass.dot}`)}></span> <p class="metric-label truncate">${escape_html(label)}</p></div> <p class="mt-1 truncate text-lg font-semibold tracking-tight text-gray-950 dark:text-white">${escape_html(value)}</p></div> <p${attr_class(`font-mono text-xs font-semibold ${toneClass.text}`)}>${escape_html(safePercent.toFixed(0))}%</p></div> <div class="mt-3 h-20 overflow-hidden rounded-md border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950"><svg class="h-full w-full"${attr("viewBox", `0 0 ${chartWidth} ${chartHeight}`)} preserveAspectRatio="none" role="img" aria-hidden="true"><g class="stroke-gray-100 dark:stroke-gray-800" stroke-width="1"><line x1="0"${attr("x2", chartWidth)}${attr("y1", chartHeight * 0.25)}${attr("y2", chartHeight * 0.25)}></line><line x1="0"${attr("x2", chartWidth)}${attr("y1", chartHeight * 0.5)}${attr("y2", chartHeight * 0.5)}></line><line x1="0"${attr("x2", chartWidth)}${attr("y1", chartHeight * 0.75)}${attr("y2", chartHeight * 0.75)}></line><line${attr("x1", chartWidth * 0.25)}${attr("x2", chartWidth * 0.25)} y1="0"${attr("y2", chartHeight)}></line><line${attr("x1", chartWidth * 0.5)}${attr("x2", chartWidth * 0.5)} y1="0"${attr("y2", chartHeight)}></line><line${attr("x1", chartWidth * 0.75)}${attr("x2", chartWidth * 0.75)} y1="0"${attr("y2", chartHeight)}></line></g><path${attr("d", areaPath)}${attr_class(clsx(toneClass.fill))}></path><path${attr("d", linePath)} fill="none"${attr_class(clsx(toneClass.stroke))} stroke-width="2" vector-effect="non-scaling-stroke"></path></svg></div> <div class="mt-2 flex items-center justify-between gap-3 text-[11px] text-gray-500 dark:text-gray-400"><p class="truncate">${escape_html(detail)}</p> <span class="shrink-0 font-mono">0-100%</span></div></article>`);
    bind_props($$props, { label, value, detail, percent, tone, className });
  });
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let metricItems, services, primary, memoryPercent, cpuPercent, runtimeSummary;
    let selectedService = "";
    let refreshing = false;
    metricItems = [];
    services = metricItems.map((item) => item.service);
    primary = metricItems.find((item) => item.service === selectedService) ?? metricItems[0] ?? null;
    memoryPercent = primary && primary.memoryLimitMb > 0 ? Math.min(primary.memoryMb / primary.memoryLimitMb * 100, 999) : 0;
    cpuPercent = primary ? Math.min(primary.cpu, 100) : 0;
    runtimeSummary = primary ? [
      { label: "Service", value: primary.service },
      { label: "Uptime", value: primary.uptime },
      {
        label: "Collected",
        value: "-"
      }
    ] : [];
    head("8nrglh", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>Metrics · MyPaas</title>`);
      });
    });
    $$renderer2.push(`<div class="space-y-4"><div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><div><h1 class="text-lg font-semibold tracking-tight text-gray-950 dark:text-white">Metrics</h1> <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">${escape_html(primary ? `Updated ${(/* @__PURE__ */ new Date("")).toLocaleTimeString()}` : "Waiting for container metrics")}</p></div> `);
    IconButton($$renderer2, {
      label: "Refresh metrics",
      variant: "brand",
      loading: refreshing,
      children: ($$renderer3) => {
        Refresh_cw($$renderer3, { class: "h-4 w-4", "aria-hidden": "true" });
      },
      $$slots: { default: true }
    });
    $$renderer2.push(`<!----></div> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> `);
    if (!primary) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="surface grid gap-0 overflow-hidden md:grid-cols-2"><!--[-->`);
      const each_array = ensure_array_like([1, 2]);
      for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
        each_array[$$index];
        $$renderer2.push(`<div class="border-b border-gray-100 p-5 dark:border-gray-800 md:border-b-0 md:border-r"><div class="h-3 w-20 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div> <div class="mt-3 h-8 w-28 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div> <div class="mt-3 h-2 w-full animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div></div>`);
      }
      $$renderer2.push(`<!--]--></div>`);
    } else if (primary) {
      $$renderer2.push("<!--[1-->");
      SectionPanel($$renderer2, {
        title: "Runtime usage",
        description: "Current CPU and memory sample for the selected service.",
        contentClass: "p-0",
        children: ($$renderer3) => {
          $$renderer3.push(`<div class="grid gap-px bg-gray-100 dark:bg-gray-800 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_18rem]">`);
          CapacityMetricChart($$renderer3, {
            label: "CPU",
            value: `${primary.cpu.toFixed(2)}%`,
            detail: "runtime sample",
            percent: cpuPercent,
            tone: cpuPercent >= 90 ? "danger" : cpuPercent >= 75 ? "warning" : "info",
            className: "bg-white dark:bg-gray-900"
          });
          $$renderer3.push(`<!----> `);
          CapacityMetricChart($$renderer3, {
            label: "Memory",
            value: `${primary.memoryMb.toFixed(1)} MB`,
            detail: `${primary.memoryLimitMb.toFixed(0)} MB limit`,
            percent: Math.min(memoryPercent, 100),
            tone: memoryPercent >= 90 ? "danger" : memoryPercent >= 75 ? "warning" : "success",
            className: "bg-white dark:bg-gray-900"
          });
          $$renderer3.push(`<!----> <div class="bg-white p-4 dark:bg-gray-900"><p class="metric-label">Runtime context</p> <div class="mt-3 divide-y divide-gray-100 dark:divide-gray-800"><!--[-->`);
          const each_array_1 = ensure_array_like(runtimeSummary);
          for (let $$index_2 = 0, $$length = each_array_1.length; $$index_2 < $$length; $$index_2++) {
            let item = each_array_1[$$index_2];
            $$renderer3.push(`<div class="flex items-center justify-between gap-3 py-2 text-xs"><span class="text-gray-500 dark:text-gray-400">${escape_html(item.label)}</span> <span class="max-w-40 truncate text-right font-medium text-gray-950 dark:text-white">${escape_html(item.value)}</span></div>`);
          }
          $$renderer3.push(`<!--]--></div></div></div>`);
        },
        $$slots: {
          default: true,
          actions: ($$renderer3) => {
            {
              if (services.length > 1) {
                $$renderer3.push("<!--[0-->");
                $$renderer3.push(`<label class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400"><span>Service</span> `);
                $$renderer3.select(
                  {
                    class: "field h-8 min-w-36 !py-1 text-xs",
                    value: selectedService
                  },
                  ($$renderer4) => {
                    $$renderer4.push(`<!--[-->`);
                    const each_array_2 = ensure_array_like(services);
                    for (let $$index_1 = 0, $$length = each_array_2.length; $$index_1 < $$length; $$index_1++) {
                      let service = each_array_2[$$index_1];
                      $$renderer4.option({ value: service }, ($$renderer5) => {
                        $$renderer5.push(`${escape_html(service)}`);
                      });
                    }
                    $$renderer4.push(`<!--]-->`);
                  }
                );
                $$renderer3.push(`</label>`);
              } else {
                $$renderer3.push("<!--[-1-->");
              }
              $$renderer3.push(`<!--]-->`);
            }
          }
        }
      });
      $$renderer2.push(`<!----> `);
      if (metricItems.length > 1) {
        $$renderer2.push("<!--[0-->");
        SectionPanel($$renderer2, {
          title: "Services",
          description: "Container-level metrics reported for this project.",
          contentClass: "p-0",
          children: ($$renderer3) => {
            $$renderer3.push(`<div class="grid divide-y divide-gray-100 dark:divide-gray-800"><!--[-->`);
            const each_array_3 = ensure_array_like(metricItems);
            for (let $$index_3 = 0, $$length = each_array_3.length; $$index_3 < $$length; $$index_3++) {
              let item = each_array_3[$$index_3];
              $$renderer3.push(`<button type="button" class="grid gap-3 px-5 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-900 sm:grid-cols-[minmax(0,1fr)_7rem_9rem_7rem]"><span class="truncate text-sm font-medium text-gray-950 dark:text-white">${escape_html(item.service)}</span> <span class="text-sm text-gray-600 dark:text-gray-300">${escape_html(item.cpu.toFixed(2))}% CPU</span> <span class="text-sm text-gray-600 dark:text-gray-300">${escape_html(item.memoryMb.toFixed(1))} / ${escape_html(item.memoryLimitMb.toFixed(0))} MB</span> <span class="text-sm text-gray-500 dark:text-gray-400">${escape_html(item.uptime)}</span></button>`);
            }
            $$renderer3.push(`<!--]--></div>`);
          },
          $$slots: { default: true }
        });
      } else {
        $$renderer2.push("<!--[-1-->");
      }
      $$renderer2.push(`<!--]-->`);
    } else {
      $$renderer2.push("<!--[2-->");
      $$renderer2.push(`<div class="surface overflow-hidden">`);
      EmptyState($$renderer2, {
        title: "No metrics yet.",
        description: "Metrics appear after the project has a running container or service.",
        compact: true
      });
      $$renderer2.push(`<!----></div>`);
    }
    $$renderer2.push(`<!--]--></div>`);
  });
}

export { _page as default };
//# sourceMappingURL=_page.svelte-CKnHalgd.js.map
