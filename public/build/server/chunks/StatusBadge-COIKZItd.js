import { r as fallback, k as attr_class, l as stringify, m as escape_html, t as bind_props } from './renderer-EjaZHhrY.js';

function StatusBadge($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let c, isPulsing;
    let status = $$props["status"];
    let pulse = fallback($$props["pulse"], false);
    const cfg = {
      running: {
        label: "Running",
        classes: "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-200",
        dot: "bg-emerald-500"
      },
      stopped: {
        label: "Stopped",
        classes: "border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400",
        dot: "bg-gray-400"
      },
      crashed: {
        label: "Crashed",
        classes: "border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200",
        dot: "bg-red-500"
      },
      building: {
        label: "Building",
        classes: "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200",
        dot: "bg-amber-500"
      },
      pending: {
        label: "Pending",
        classes: "border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-200",
        dot: "bg-sky-500"
      },
      queued: {
        label: "Queued",
        classes: "border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-200",
        dot: "bg-sky-500"
      },
      cloning: {
        label: "Cloning",
        classes: "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200",
        dot: "bg-amber-500"
      },
      starting: {
        label: "Starting",
        classes: "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200",
        dot: "bg-amber-500"
      },
      failed: {
        label: "Failed",
        classes: "border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200",
        dot: "bg-red-500"
      },
      rolled_back: {
        label: "Rolled back",
        classes: "border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400",
        dot: "bg-gray-400"
      }
    };
    c = cfg[status] ?? {
      label: status,
      classes: "border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400",
      dot: "bg-gray-400"
    };
    isPulsing = pulse && ["building", "cloning", "starting", "queued"].includes(status);
    $$renderer2.push(`<span${attr_class(`inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-[11px] font-medium leading-none ${stringify(c.classes)}`)}>`);
    if (isPulsing) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<span class="relative flex h-1.5 w-1.5"><span${attr_class(`absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 ${stringify(c.dot)}`)}></span> <span${attr_class(`relative inline-flex h-1.5 w-1.5 rounded-full ${stringify(c.dot)}`)}></span></span>`);
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<span${attr_class(`h-1.5 w-1.5 rounded-full ${stringify(c.dot)}`)}></span>`);
    }
    $$renderer2.push(`<!--]--> ${escape_html(c.label)}</span>`);
    bind_props($$props, { status, pulse });
  });
}

export { StatusBadge as S };
//# sourceMappingURL=StatusBadge-COIKZItd.js.map
