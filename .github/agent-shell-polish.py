from pathlib import Path


def replace_once(path: str, old: str, new: str):
    p = Path(path)
    text = p.read_text()
    if old not in text:
        raise SystemExit(f"missing replacement anchor in {path}: {old[:120]!r}")
    p.write_text(text.replace(old, new, 1))


replace_once(
    "frontend/src/lib/components/BrandLogo.svelte",
    "\t$: sizing = compact ? 'h-6 w-auto' : 'w-32 h-auto';",
    "\t$: sizing = compact ? 'h-[22px] w-auto' : 'w-28 h-auto';",
)

replace_once(
    "frontend/src/lib/components/Navbar.svelte",
    "\timport { ArrowRightLeft, Bot, ChevronLeft, ClipboardList, FolderKanban, LogOut, Moon, Settings, Sun, Users } from '@lucide/svelte';",
    "\timport { ArrowRightLeft, Bot, ClipboardList, FolderKanban, LogOut, Moon, Settings, Sun, Users } from '@lucide/svelte';",
)
replace_once(
    "frontend/src/lib/components/Navbar.svelte",
    "\timport IconButton from '$components/IconButton.svelte';\n",
    "\timport IconButton from '$components/IconButton.svelte';\n\timport SidebarPanelIcon from '$components/SidebarPanelIcon.svelte';\n",
)
replace_once(
    "frontend/src/lib/components/Navbar.svelte",
    "\t\t\t\t<BrandLogo compact />",
    "\t\t\t\t<SidebarPanelIcon collapsed className=\"h-[18px] w-[18px]\" />",
)
replace_once(
    "frontend/src/lib/components/Navbar.svelte",
    "\t\t\t\t<ChevronLeft class=\"h-4 w-4\" aria-hidden=\"true\" />",
    "\t\t\t\t<SidebarPanelIcon className=\"h-[18px] w-[18px]\" />",
)
replace_once(
    "frontend/src/lib/components/Navbar.svelte",
    "{$sidebarCollapsed ? 'justify-center px-2' : 'justify-between gap-2.5 px-5'}",
    "{$sidebarCollapsed ? 'justify-center px-2' : 'justify-between gap-2.5 px-4'}",
)

Path("frontend/src/lib/components/StorageCapacityMetric.svelte").write_text('''<script lang="ts">
\texport let label = 'Storage';
\texport let value = 'Unavailable';
\texport let detail = 'Host telemetry unavailable';
\texport let percent = 0;
\texport let className = '';

\t$: usedPercent = Math.min(Math.max(Number.isFinite(percent) ? percent : 0, 0), 100);
\t$: available = !/unavailable/i.test(`${value} ${detail}`);
\t$: fillTop = 76 - (56 * usedPercent) / 100;
\t$: fillClass = usedPercent >= 90
\t\t? 'fill-red-500/65 dark:fill-red-400/55'
\t\t: usedPercent >= 80
\t\t\t? 'fill-amber-500/60 dark:fill-amber-300/50'
\t\t\t: 'fill-gray-700/55 dark:fill-gray-200/45';
</script>

<article class={`grid h-full min-h-40 min-w-0 grid-cols-[minmax(0,1fr)_6rem] items-center gap-4 p-4 ${className}`.trim()} aria-label={`${label}: ${value}`}>
\t<div class="min-w-0">
\t\t<p class="metric-label truncate">{label}</p>
\t\t<p class="metric-value mt-2 truncate text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{value}</p>
\t\t{#if available}
\t\t\t<p class="metric-value mt-4 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{usedPercent.toFixed(0)}%</p>
\t\t{:else}
\t\t\t<p class="mt-4 text-sm font-medium text-gray-400 dark:text-gray-500">No sample</p>
\t\t{/if}
\t\t<p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{detail}</p>
\t</div>

\t<div
\t\tclass="flex items-center justify-center"
\t\trole={available ? 'progressbar' : undefined}
\t\taria-label={available ? `${label} used` : undefined}
\t\taria-valuemin={available ? 0 : undefined}
\t\taria-valuemax={available ? 100 : undefined}
\t\taria-valuenow={available ? Math.round(usedPercent) : undefined}
\t>
\t\t<svg viewBox="0 0 96 104" class="h-28 w-24" aria-hidden="true">
\t\t\t{#if available && usedPercent > 0}
\t\t\t\t<rect x="20" y={fillTop} width="56" height={76 - fillTop} class={fillClass} />
\t\t\t\t<ellipse cx="48" cy={fillTop} rx="28" ry="7" class={fillClass} />
\t\t\t\t<ellipse cx="48" cy="76" rx="28" ry="7" class={fillClass} />
\t\t\t{/if}
\t\t\t<ellipse cx="48" cy="20" rx="28" ry="8" fill="none" class="stroke-gray-500 dark:stroke-gray-400" stroke-width="1.5" />
\t\t\t<path d="M20 20v56c0 4.4 12.5 8 28 8s28-3.6 28-8V20" fill="none" class="stroke-gray-500 dark:stroke-gray-400" stroke-width="1.5" />
\t\t\t<path d="M20 76c0 4.4 12.5 8 28 8s28-3.6 28-8" fill="none" class="stroke-gray-500 dark:stroke-gray-400" stroke-width="1.5" />
\t\t\t{#if !available}
\t\t\t\t<path d="M32 49h32" class="stroke-gray-300 dark:stroke-gray-700" stroke-width="1.5" stroke-dasharray="3 3" />
\t\t\t{/if}
\t\t</svg>
\t</div>
</article>
''')

replace_once(
    "frontend/src/routes/projects/+page.svelte",
    "\timport { ExternalLink, FolderGit2, GitBranch, Package, Play, Plus, RefreshCw, Rocket, Search, Square, TriangleAlert, X } from '@lucide/svelte';",
    "\timport { ExternalLink, FolderGit2, GitBranch, LoaderCircle, Package, Play, Plus, RefreshCw, Rocket, Search, Square, TriangleAlert, X } from '@lucide/svelte';",
)
replace_once(
    "frontend/src/routes/projects/+page.svelte",
    "\t$: syncLabel = error ? 'Refresh needs attention' : loading ? 'Refreshing' : 'Up to date';\n\t$: syncDotClass = error ? 'bg-amber-500' : loading ? 'bg-gray-400 animate-pulse' : 'bg-gray-500 dark:bg-gray-400';",
    "\t$: dashboardRefreshing = loading || projectsInFlight || hostStatsInFlight;\n\t$: syncLabel = error ? 'Refresh needs attention' : dashboardRefreshing ? 'Refreshing' : 'Up to date';\n\t$: syncDotClass = error ? 'bg-amber-500' : dashboardRefreshing ? 'bg-gray-400 animate-pulse' : 'bg-gray-500 dark:bg-gray-400';",
)
replace_once(
    "frontend/src/routes/projects/+page.svelte",
    '<ActionButton variant="secondary" loading={loading} loadingLabel="Refreshing" on:click={() => refreshDashboardData()}>',
    '<ActionButton variant="secondary" loading={dashboardRefreshing} loadingLabel="Refreshing" on:click={() => refreshDashboardData()}>',
)
replace_once(
    "frontend/src/routes/projects/+page.svelte",
    '\t\t<svelte:fragment slot="actions">\n\t\t\t<a\n',
    '\t\t<svelte:fragment slot="actions">\n\t\t\t<div class="flex items-center gap-3">\n\t\t\t\t{#if hostStatsInFlight}<LoaderCircle class="h-3.5 w-3.5 animate-spin text-gray-400 motion-reduce:animate-none dark:text-gray-500" aria-hidden="true" />{/if}\n\t\t\t\t<a\n',
)
replace_once(
    "frontend/src/routes/projects/+page.svelte",
    '\t\t\t</a>\n\t\t</svelte:fragment>\n\n\t\t{#if hostStats}',
    '\t\t\t\t</a>\n\t\t\t</div>\n\t\t</svelte:fragment>\n\n\t\t{#if hostStats}',
)
replace_once(
    "frontend/src/routes/projects/+page.svelte",
    '''\t\t{:else}\n\t\t\t<div class="grid sm:grid-cols-2 xl:grid-cols-4" aria-busy="true">\n\t\t\t\t{#each Array(4) as _}\n\t\t\t\t\t<div class="h-40 animate-pulse border-b border-gray-100 bg-gray-100/70 last:border-b-0 dark:border-neutral-800 dark:bg-neutral-800/60 sm:border-r xl:border-b-0"></div>\n\t\t\t\t{/each}\n\t\t\t</div>\n\t\t{/if}''',
    '''\t\t{:else}\n\t\t\t<div class="flex h-40 items-center justify-center gap-2 text-sm text-gray-500 dark:text-gray-400" aria-busy="true" aria-live="polite">\n\t\t\t\t<LoaderCircle class="h-5 w-5 animate-spin motion-reduce:animate-none" aria-hidden="true" />\n\t\t\t\t<span>Loading host telemetry…</span>\n\t\t\t</div>\n\t\t{/if}''',
)
replace_once(
    "frontend/src/routes/projects/+page.svelte",
    '<span class="text-xs text-gray-400 dark:text-gray-500">Loading…</span>',
    '<span class="inline-flex items-center text-gray-400 dark:text-gray-500"><LoaderCircle class="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" aria-hidden="true" /><span class="sr-only">Loading metrics</span></span>',
)

replace_once(
    "frontend/src/routes/admin/mcp/+page.svelte",
    "\timport ActionButton from '$components/ActionButton.svelte';\n",
    "\timport ActionButton from '$components/ActionButton.svelte';\n\timport AgentBadgeStack from '$components/AgentBadgeStack.svelte';\n",
)
replace_once(
    "frontend/src/routes/admin/mcp/+page.svelte",
    '<SectionPanel title="What MCP enables" description="The MyPaaS MCP server translates agent tool calls into authenticated MyPaaS API operations." contentClass="p-0">\n\t\t<div',
    '<SectionPanel title="What MCP enables" description="The MyPaaS MCP server translates agent tool calls into authenticated MyPaaS API operations." contentClass="p-0">\n\t\t<svelte:fragment slot="actions"><AgentBadgeStack /></svelte:fragment>\n\t\t<div',
)
