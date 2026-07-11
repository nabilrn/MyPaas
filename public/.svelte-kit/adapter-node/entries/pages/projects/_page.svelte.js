import { i as spread_props, j as fallback, c as escape_html, f as attr, e as ensure_array_like, a as attr_class, k as clsx, b as stringify, l as attr_style, m as bind_props, n as sanitize_slots, d as slot, h as head, s as store_get, u as unsubscribe_stores } from "../../../chunks/renderer.js";
import { p as page } from "../../../chunks/stores.js";
/* empty css                                                         */
import { B as Breadcrumbs } from "../../../chunks/Breadcrumbs.js";
import { S as StatusBadge } from "../../../chunks/StatusBadge.js";
import { I as IconButton } from "../../../chunks/IconButton.js";
import { P as PageHeader } from "../../../chunks/PageHeader.js";
import { T as TableShell, P as Pagination } from "../../../chunks/TableShell.js";
import { S as SectionPanel } from "../../../chunks/SectionPanel.js";
import "../../../chunks/toast.js";
import { p as projectURL } from "../../../chunks/urls.js";
import { I as Icon } from "../../../chunks/Icon.js";
import { R as Refresh_cw } from "../../../chunks/refresh-cw.js";
import { P as Plus } from "../../../chunks/plus.js";
function External_link($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    ["path", { "d": "M15 3h6v6" }],
    ["path", { "d": "M10 14 21 3" }],
    [
      "path",
      {
        "d": "M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"
      }
    ]
  ];
  Icon($$renderer, spread_props([{ name: "external-link" }, props, { iconNode }]));
}
function Pause($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    [
      "rect",
      { "x": "14", "y": "3", "width": "5", "height": "18", "rx": "1" }
    ],
    [
      "rect",
      { "x": "5", "y": "3", "width": "5", "height": "18", "rx": "1" }
    ]
  ];
  Icon($$renderer, spread_props([{ name: "pause" }, props, { iconNode }]));
}
function Play($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    [
      "path",
      {
        "d": "M5 5a2 2 0 0 1 3.008-1.728l11.997 6.998a2 2 0 0 1 .003 3.458l-12 7A2 2 0 0 1 5 19z"
      }
    ]
  ];
  Icon($$renderer, spread_props([{ name: "play" }, props, { iconNode }]));
}
function Search($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    ["path", { "d": "m21 21-4.34-4.34" }],
    ["circle", { "cx": "11", "cy": "11", "r": "8" }]
  ];
  Icon($$renderer, spread_props([{ name: "search" }, props, { iconNode }]));
}
function FleetStatusChart($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let total, visibleSegments, activeLabel, ringSegments, toneClass, dotClass;
    let segments = fallback($$props["segments"], () => [], true);
    let title = fallback($$props["title"], "Fleet health");
    let subtitle = fallback($$props["subtitle"], "");
    const radius = 38;
    const circumference = 2 * Math.PI * radius;
    total = segments.reduce((sum, item) => sum + item.value, 0);
    visibleSegments = segments.filter((item) => item.value > 0);
    activeLabel = total > 0 ? `${total} total` : "No projects";
    ringSegments = visibleSegments.map((segment, index) => {
      const previous = visibleSegments.slice(0, index).reduce((sum, item) => sum + item.value, 0);
      const length = total > 0 ? segment.value / total * circumference : 0;
      const offset = total > 0 ? previous / total * circumference : 0;
      return { ...segment, length, offset };
    });
    toneClass = {
      success: "stroke-emerald-500",
      info: "stroke-sky-500",
      warning: "stroke-amber-500",
      danger: "stroke-red-500",
      neutral: "stroke-gray-400"
    };
    dotClass = {
      success: "bg-emerald-500",
      info: "bg-sky-500",
      warning: "bg-amber-500",
      danger: "bg-red-500",
      neutral: "bg-gray-400"
    };
    $$renderer2.push(`<section class="surface h-full overflow-hidden p-5"><div class="flex items-start justify-between gap-4"><div><h2 class="text-sm font-semibold text-gray-950 dark:text-white">${escape_html(title)}</h2> `);
    if (subtitle) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">${escape_html(subtitle)}</p>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--></div> <span class="shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400">${escape_html(activeLabel)}</span></div> <div class="mt-5 grid gap-5 sm:grid-cols-[9rem_minmax(0,1fr)] sm:items-center"><div class="relative mx-auto h-32 w-32"><svg viewBox="0 0 100 100" class="h-full w-full -rotate-90" role="img"${attr("aria-label", title)}><circle cx="50" cy="50"${attr("r", radius)} fill="none" class="stroke-gray-100 dark:stroke-gray-800" stroke-width="10"></circle><!--[-->`);
    const each_array = ensure_array_like(ringSegments);
    for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
      let segment = each_array[$$index];
      $$renderer2.push(`<circle cx="50" cy="50"${attr("r", radius)} fill="none"${attr_class(clsx(toneClass[segment.tone]))} stroke-width="10" stroke-linecap="round"${attr("stroke-dasharray", `${segment.length} ${circumference - segment.length}`)}${attr("stroke-dashoffset", -segment.offset)}></circle>`);
    }
    $$renderer2.push(`<!--]--></svg> <div class="absolute inset-0 flex flex-col items-center justify-center text-center"><span class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">${escape_html(total)}</span> <span class="text-[11px] text-gray-500 dark:text-gray-400">projects</span></div></div> <div class="space-y-3"><!--[-->`);
    const each_array_1 = ensure_array_like(segments);
    for (let $$index_1 = 0, $$length = each_array_1.length; $$index_1 < $$length; $$index_1++) {
      let segment = each_array_1[$$index_1];
      $$renderer2.push(`<div><div class="flex items-center justify-between gap-3 text-xs"><span class="inline-flex min-w-0 items-center gap-2 text-gray-600 dark:text-gray-300"><span${attr_class(`h-2 w-2 shrink-0 rounded-full ${stringify(dotClass[segment.tone])}`)}></span> <span class="truncate">${escape_html(segment.label)}</span></span> <span class="font-mono font-medium text-gray-950 dark:text-white">${escape_html(segment.value)}</span></div> <div class="mt-1 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800"><div${attr_class(`h-full rounded-full ${stringify(dotClass[segment.tone])}`)}${attr_style(`width: ${total > 0 ? Math.max(2, segment.value / total * 100) : 0}%`)}></div></div></div>`);
    }
    $$renderer2.push(`<!--]--></div></div></section>`);
    bind_props($$props, { segments, title, subtitle });
  });
}
function StatTile($$renderer, $$props) {
  const $$slots = sanitize_slots($$props);
  $$renderer.component(($$renderer2) => {
    let toneClass, valueClass;
    let label = fallback($$props["label"], "");
    let value = fallback($$props["value"], "");
    let detail = fallback($$props["detail"], "");
    let tone = fallback($$props["tone"], "neutral");
    toneClass = {
      neutral: "bg-gray-400",
      success: "bg-emerald-500",
      info: "bg-sky-500",
      warning: "bg-amber-500",
      danger: "bg-red-500"
    }[tone];
    valueClass = {
      neutral: "text-gray-950 dark:text-white",
      success: "text-emerald-700 dark:text-emerald-200",
      info: "text-sky-700 dark:text-sky-200",
      warning: "text-amber-700 dark:text-amber-200",
      danger: "text-red-700 dark:text-red-200"
    }[tone];
    $$renderer2.push(`<div class="soft-panel min-w-0 p-4"><div class="flex items-center gap-2"><span${attr_class(`h-1.5 w-1.5 rounded-full ${toneClass}`)}></span> <p class="metric-label truncate">${escape_html(label)}</p></div> <p${attr_class(`mt-2 truncate text-2xl font-semibold tracking-tight ${valueClass}`)}>${escape_html(value)}</p> `);
    if (detail) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<p class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">${escape_html(detail)}</p>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> `);
    if ($$slots.default) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="mt-3"><!--[-->`);
      slot($$renderer2, $$props, "default", {});
      $$renderer2.push(`<!--]--></div>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--></div>`);
    bind_props($$props, { label, value, detail, tone });
  });
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let normalizedSearch, filteredProjects, projectPercent, runningCount, buildingCount, issueCount, stoppedCount, pendingCount, dockerfileCount, composeCount, staticCount, latestProject, healthyCopy, syncLabel, syncDetail, syncDotClass, healthSegments, deployModeSegments, maxPage, pageStart, visibleProjects, hasNext;
    const pageSize = 20;
    const breadcrumbs = [{ label: "Projects" }];
    let projects = [];
    let loading = true;
    let projectActionId = "";
    let currentPage = 0;
    let searchQuery = "";
    let projectUptimes = {};
    let uptimeLoadingIds = /* @__PURE__ */ new Set();
    function projectPrimaryAction(project) {
      if (project.status === "building") return "busy";
      if (project.status === "running") return "stop";
      if (project.status === "stopped") return "start";
      return "deploy";
    }
    function projectPrimaryLabel(project) {
      const action = projectPrimaryAction(project);
      if (projectActionId === project.id) {
        return "Deployment in progress";
      }
      if (action === "busy") return "Deployment in progress";
      if (action === "stop") return "Stop project";
      if (action === "start") return "Start project";
      return "Deploy project";
    }
    function projectPrimaryVariant(project) {
      const action = projectPrimaryAction(project);
      if (action === "stop") return "danger";
      if (action === "busy") return "ghost";
      return "primary";
    }
    function formatDate(value) {
      return new Date(value).toLocaleDateString(void 0, { month: "short", day: "numeric" });
    }
    function formatDateTime(value) {
      return new Date(value).toLocaleString(void 0, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit"
      });
    }
    function appUrl(project) {
      return projectURL(project.subdomain, store_get($$store_subs ??= {}, "$page", page).url.protocol, store_get($$store_subs ??= {}, "$page", page).url.hostname);
    }
    normalizedSearch = searchQuery.trim().toLowerCase();
    filteredProjects = normalizedSearch ? projects.filter((project) => [
      project.name,
      project.subdomain,
      project.repoUrl,
      project.branch,
      project.deployMode,
      project.mainService ?? "",
      project.status
    ].join(" ").toLowerCase().includes(normalizedSearch)) : projects;
    projectPercent = 0;
    runningCount = projects.filter((project) => project.status === "running").length;
    buildingCount = projects.filter((project) => project.status === "building").length;
    issueCount = projects.filter((project) => project.status === "crashed").length;
    stoppedCount = projects.filter((project) => project.status === "stopped").length;
    pendingCount = projects.filter((project) => project.status === "pending").length;
    dockerfileCount = projects.filter((project) => project.deployMode === "dockerfile").length;
    composeCount = projects.filter((project) => project.deployMode === "compose").length;
    staticCount = projects.filter((project) => project.deployMode === "static").length;
    latestProject = [...projects].sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())[0];
    healthyCopy = issueCount > 0 ? `${issueCount} project${issueCount !== 1 ? "s" : ""} need attention` : `${runningCount} running, no crashed projects`;
    syncLabel = "Syncing workspace";
    syncDetail = "Waiting for first refresh";
    syncDotClass = "bg-sky-500 animate-pulse";
    healthSegments = [
      { label: "Running", value: runningCount, tone: "success" },
      { label: "Building", value: buildingCount, tone: "warning" },
      { label: "Stopped", value: stoppedCount, tone: "neutral" },
      { label: "Pending", value: pendingCount, tone: "info" },
      { label: "Crashed", value: issueCount, tone: "danger" }
    ];
    deployModeSegments = [
      {
        label: "Dockerfile",
        value: dockerfileCount,
        barClass: "bg-sky-500",
        textClass: "text-sky-700 dark:text-sky-300"
      },
      {
        label: "Compose",
        value: composeCount,
        barClass: "bg-brand-500",
        textClass: "text-brand-700 dark:text-brand-100"
      },
      {
        label: "Static",
        value: staticCount,
        barClass: "bg-gray-400 dark:bg-gray-500",
        textClass: "text-gray-600 dark:text-gray-300"
      }
    ];
    deployModeSegments.reduce((sum, segment) => sum + segment.value, 0);
    deployModeSegments.reduce((top, segment) => segment.value > top.value ? segment : top, deployModeSegments[0]);
    maxPage = Math.max(0, Math.ceil(filteredProjects.length / pageSize) - 1);
    if (currentPage > maxPage) {
      currentPage = maxPage;
    }
    pageStart = currentPage * pageSize;
    visibleProjects = filteredProjects.slice(pageStart, pageStart + pageSize);
    hasNext = pageStart + pageSize < filteredProjects.length;
    let $$settled = true;
    let $$inner_renderer;
    function $$render_inner($$renderer3) {
      head("rqn88j", $$renderer3, ($$renderer4) => {
        $$renderer4.title(($$renderer5) => {
          $$renderer5.push(`<title>Projects · MyPaas</title>`);
        });
      });
      $$renderer3.push(`<div class="page-shell py-6"><div class="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between"><div class="min-w-0">`);
      Breadcrumbs($$renderer3, { items: breadcrumbs });
      $$renderer3.push(`<!----> `);
      PageHeader($$renderer3, {
        title: "Deployment control plane",
        description: `${projects.length} project${projects.length !== 1 ? "s" : ""} connected. Watch health, capacity, and deploy actions from one operational surface.`,
        className: "!mb-0"
      });
      $$renderer3.push(`<!----></div> <div class="flex w-full flex-col items-stretch gap-2 sm:w-auto sm:items-end"><div class="flex min-h-10 w-full items-center gap-2 rounded-md border border-gray-200 bg-white px-3 py-2 text-left shadow-sm shadow-gray-950/[0.03] dark:border-gray-800 dark:bg-gray-950 sm:min-w-[15rem]"><span${attr_class(`h-2 w-2 shrink-0 rounded-full ${syncDotClass}`)}></span> <div class="min-w-0"><p class="truncate text-xs font-medium text-gray-900 dark:text-white">${escape_html(syncLabel)}</p> <p class="truncate text-[11px] text-gray-500 dark:text-gray-400">${escape_html(syncDetail)}</p></div></div> <div class="flex justify-end gap-2">`);
      IconButton($$renderer3, {
        label: "Refresh dashboard data",
        variant: "brand",
        loading,
        children: ($$renderer4) => {
          Refresh_cw($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
        },
        $$slots: { default: true }
      });
      $$renderer3.push(`<!----> `);
      IconButton($$renderer3, {
        label: "New project",
        href: "/projects/new",
        variant: "primary",
        children: ($$renderer4) => {
          Plus($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
        },
        $$slots: { default: true }
      });
      $$renderer3.push(`<!----></div></div></div> <div class="mb-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">`);
      StatTile($$renderer3, {
        label: "Fleet health",
        value: issueCount > 0 ? "Attention" : "Healthy",
        detail: healthyCopy,
        tone: issueCount > 0 ? "danger" : "success"
      });
      $$renderer3.push(`<!----> `);
      StatTile($$renderer3, {
        label: "Running now",
        value: String(runningCount),
        detail: `${buildingCount} building, ${pendingCount} pending`,
        tone: buildingCount > 0 ? "warning" : "success"
      });
      $$renderer3.push(`<!----> `);
      StatTile($$renderer3, {
        label: "Latest activity",
        value: latestProject?.name ?? "No activity",
        detail: latestProject ? formatDateTime(latestProject.updatedAt) : "Create a project to start deploying",
        tone: "neutral",
        children: ($$renderer4) => {
          if (latestProject) {
            $$renderer4.push("<!--[0-->");
            $$renderer4.push(`<a${attr("href", `/projects/${stringify(latestProject.id)}`)} class="text-xs font-medium text-brand-700 hover:underline dark:text-brand-100">Open project</a>`);
          } else {
            $$renderer4.push("<!--[-1-->");
          }
          $$renderer4.push(`<!--]-->`);
        },
        $$slots: { default: true }
      });
      $$renderer3.push(`<!----> `);
      StatTile($$renderer3, {
        label: "Project slots",
        value: `${projects.length}`,
        detail: "Waiting for quota data",
        tone: projectPercent >= 80 ? "warning" : "info"
      });
      $$renderer3.push(`<!----></div> <div class="mb-5 grid gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">`);
      SectionPanel($$renderer3, {
        title: "Capacity and deploy modes",
        description: "Configured quota, live resource shape, and runtime composition across connected projects.",
        contentClass: "p-0",
        children: ($$renderer4) => {
          {
            $$renderer4.push("<!--[-1-->");
            $$renderer4.push(`<div class="grid gap-0 sm:grid-cols-2 xl:grid-cols-4" aria-busy="true"><div class="h-36 animate-pulse border-b border-gray-100 bg-gray-100/70 dark:border-gray-800 dark:bg-gray-800/60 sm:border-r xl:border-b-0"></div> <div class="h-36 animate-pulse border-b border-gray-100 bg-gray-100/70 dark:border-gray-800 dark:bg-gray-800/60 sm:border-r xl:border-b-0"></div> <div class="h-36 animate-pulse border-b border-gray-100 bg-gray-100/70 dark:border-gray-800 dark:bg-gray-800/60 sm:border-r sm:border-b-0"></div> <div class="h-36 animate-pulse bg-gray-100/70 dark:bg-gray-800/60"></div></div>`);
          }
          $$renderer4.push(`<!--]-->`);
        },
        $$slots: { default: true }
      });
      $$renderer3.push(`<!----> `);
      FleetStatusChart($$renderer3, {
        segments: healthSegments,
        title: "Fleet health",
        subtitle: "Status composition across connected projects."
      });
      $$renderer3.push(`<!----></div> `);
      TableShell($$renderer3, {
        title: "Project inventory",
        description: "Runtime state, deployment mode, capacity, and quick actions.",
        loading,
        loadingRows: 3,
        error: "",
        empty: filteredProjects.length === 0,
        emptyTitle: normalizedSearch ? "No projects match this search" : "No projects yet",
        emptyDescription: normalizedSearch ? "Try a project name, subdomain, branch, deploy mode, or status." : "Connect a Git repository and MyPaas will build it from Dockerfile or Compose.",
        contentClass: "overflow-hidden",
        children: ($$renderer4) => {
          $$renderer4.push(`<div class="hidden w-full grid-cols-[minmax(0,1.35fr)_minmax(0,0.8fr)_minmax(0,1.35fr)_minmax(0,1.05fr)_minmax(0,0.55fr)_minmax(0,0.55fr)_4.75rem] items-center gap-x-4 border-b border-gray-100 bg-gray-50/70 px-4 py-2 text-xs font-medium text-gray-500 dark:border-gray-800 dark:bg-gray-900/70 dark:text-gray-400 lg:grid"><span>Project</span> <span>Status</span> <span>App URL</span> <span>Runtime</span> <span>Uptime</span> <span>Updated</span> <span class="text-right">Actions</span></div> <div class="divide-y divide-gray-100 dark:divide-gray-800"><!--[-->`);
          const each_array_2 = ensure_array_like(visibleProjects);
          for (let $$index_2 = 0, $$length = each_array_2.length; $$index_2 < $$length; $$index_2++) {
            let project = each_array_2[$$index_2];
            $$renderer4.push(`<div class="grid gap-y-3 px-4 py-4 transition-colors hover:bg-gray-50/80 dark:hover:bg-gray-900/70 lg:w-full lg:grid-cols-[minmax(0,1.35fr)_minmax(0,0.8fr)_minmax(0,1.35fr)_minmax(0,1.05fr)_minmax(0,0.55fr)_minmax(0,0.55fr)_4.75rem] lg:items-center lg:gap-x-4"><div class="min-w-0"><a${attr("href", `/projects/${stringify(project.id)}`)} class="block truncate text-sm font-semibold text-gray-950 hover:underline dark:text-white">${escape_html(project.name)}</a> <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">${escape_html(project.subdomain)}</p></div> <div>`);
            StatusBadge($$renderer4, { status: project.status, pulse: true });
            $$renderer4.push(`<!----></div> <a${attr("href", appUrl(project))} target="_blank" rel="noopener" class="truncate font-mono text-xs text-gray-600 hover:text-gray-950 hover:underline dark:text-gray-300 dark:hover:text-white">${escape_html(appUrl(project).replace(/^https?:\/\//, ""))}</a> <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400"><span class="rounded border border-gray-200 px-1.5 py-0.5 font-mono dark:border-gray-800">${escape_html(project.deployMode)}</span> `);
            if (project.mainService) {
              $$renderer4.push("<!--[0-->");
              $$renderer4.push(`<span class="truncate">${escape_html(project.mainService)}</span>`);
            } else {
              $$renderer4.push("<!--[-1-->");
            }
            $$renderer4.push(`<!--]--> <span>${escape_html(project.memoryLimitMb)}MB</span></div> <div class="font-mono text-xs text-gray-500 dark:text-gray-400">${escape_html(projectUptimes[project.id] ?? (uptimeLoadingIds.has(project.id) ? "Loading" : "-"))}</div> <div class="text-xs text-gray-500 dark:text-gray-400">${escape_html(formatDate(project.updatedAt))}</div> <div class="flex items-center justify-start gap-1.5 lg:justify-end">`);
            IconButton($$renderer4, {
              label: "Open project",
              href: `/projects/${stringify(project.id)}`,
              variant: "default",
              children: ($$renderer5) => {
                External_link($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
              },
              $$slots: { default: true }
            });
            $$renderer4.push(`<!----> `);
            IconButton($$renderer4, {
              label: projectPrimaryLabel(project),
              variant: projectPrimaryVariant(project),
              loading: projectActionId === project.id || projectPrimaryAction(project) === "busy",
              disabled: projectPrimaryAction(project) === "busy",
              children: ($$renderer5) => {
                if (projectPrimaryAction(project) === "stop") {
                  $$renderer5.push("<!--[0-->");
                  Pause($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                } else {
                  $$renderer5.push("<!--[-1-->");
                  Play($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                }
                $$renderer5.push(`<!--]-->`);
              },
              $$slots: { default: true }
            });
            $$renderer4.push(`<!----></div></div>`);
          }
          $$renderer4.push(`<!--]--></div>`);
        },
        $$slots: {
          default: true,
          actions: ($$renderer4) => {
            {
              $$renderer4.push(`<div class="flex flex-col gap-2 sm:flex-row sm:items-center"><label class="relative block w-full sm:w-72"><span class="sr-only">Search projects</span> `);
              Search($$renderer4, {
                class: "pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400 dark:text-gray-500",
                "aria-hidden": "true"
              });
              $$renderer4.push(`<!----> <input type="text" inputmode="search"${attr("value", searchQuery)} placeholder="Search projects" class="field h-9 w-full !pl-10 !pr-9"/> `);
              {
                $$renderer4.push("<!--[-1-->");
              }
              $$renderer4.push(`<!--]--></label> `);
              IconButton($$renderer4, {
                label: "Refresh dashboard data",
                variant: "ghost",
                loading,
                children: ($$renderer5) => {
                  Refresh_cw($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                },
                $$slots: { default: true }
              });
              $$renderer4.push(`<!----></div>`);
            }
          },
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
                totalShown: visibleProjects.length,
                hasNext,
                loading,
                label: "Projects",
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
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}
export {
  _page as default
};
