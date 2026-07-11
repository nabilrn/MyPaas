import { h as head, j as ensure_array_like, m as escape_html } from './renderer-EjaZHhrY.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';

/* empty css                                                            */
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let lastDeploy, primaryMetric;
    let project = null;
    let deployments = [];
    function formatDuration(start, end) {
      if (!end) return "-";
      const seconds = Math.max(0, Math.floor((new Date(end).getTime() - new Date(start).getTime()) / 1e3));
      return seconds < 60 ? `${seconds}s` : `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    }
    lastDeploy = deployments.find((d) => d.id === project?.activeDeploymentId) ?? deployments[0];
    primaryMetric = null;
    primaryMetric && primaryMetric.memoryLimitMb > 0 ? Math.min(primaryMetric.memoryMb / primaryMetric.memoryLimitMb * 100, 100) : 0;
    primaryMetric ? Math.min(primaryMetric.cpu, 100) : 0;
    lastDeploy?.commitSha?.slice(0, 8) ?? "-";
    lastDeploy ? formatDuration(lastDeploy.startedAt, lastDeploy.finishedAt) : "-";
    head("ffmenf", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>${escape_html("Project")} · MyPaas</title>`);
      });
    });
    {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="space-y-4"><div class="surface grid gap-0 overflow-hidden sm:grid-cols-4"><!--[-->`);
      const each_array = ensure_array_like([1, 2, 3, 4]);
      for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
        each_array[$$index];
        $$renderer2.push(`<div class="border-b border-gray-100 p-5 dark:border-gray-800 sm:border-b-0 sm:border-r"><div class="h-3 w-20 animate-pulse rounded bg-gray-100 dark:bg-gray-800"></div> <div class="mt-3 h-7 w-24 animate-pulse rounded bg-gray-200 dark:bg-gray-800"></div></div>`);
      }
      $$renderer2.push(`<!--]--></div> <div class="grid gap-4 lg:grid-cols-2"><div class="surface h-56 animate-pulse"></div> <div class="surface h-56 animate-pulse"></div></div></div>`);
    }
    $$renderer2.push(`<!--]-->`);
  });
}

export { _page as default };
//# sourceMappingURL=_page.svelte-BMWt9i_o.js.map
