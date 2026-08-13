from pathlib import Path


def replace_once(path: str, old: str, new: str):
    p = Path(path)
    text = p.read_text()
    if old not in text:
        raise SystemExit(f"missing replacement anchor in {path}: {old[:120]!r}")
    p.write_text(text.replace(old, new, 1))


replace_once(
    "backend/internal/deployment/metrics_response.go",
    'import (\n\t"time"\n',
    'import (\n\t"strings"\n\t"time"\n',
)
replace_once(
    "backend/internal/deployment/metrics_response.go",
    '\tfor _, item := range metrics {\n\t\tif collectedAt.IsZero() || item.CollectedAt.After(collectedAt) {',
    '\tfor _, item := range metrics {\n\t\tif strings.EqualFold(strings.TrimSpace(item.Uptime), "n/a") {\n\t\t\tcontinue\n\t\t}\n\t\tif collectedAt.IsZero() || item.CollectedAt.After(collectedAt) {',
)

Path("backend/internal/deployment/metrics_response_test.go").write_text('''package deployment

import (
\t"testing"
\t"time"

\t"mypaas/internal/container"
)

func TestMetricsSnapshotSkipsSyntheticIdleSamples(t *testing.T) {
\tnow := time.Date(2026, 8, 13, 9, 0, 0, 0, time.UTC)
\tresp := MetricsSnapshotFromContainers([]container.Metrics{
\t\t{Service: "static", CPUPercent: 0, MemoryMB: 0, MemoryLimitMB: 64, Uptime: "n/a", CollectedAt: now},
\t\t{Service: "app", CPUPercent: 3.5, MemoryMB: 12, MemoryLimitMB: 256, Uptime: "2m", CollectedAt: now.Add(time.Second)},
\t})
\tif len(resp.Items) != 1 {
\t\tt.Fatalf("expected one live runtime metric, got %d", len(resp.Items))
\t}
\tif resp.Items[0].Service != "app" {
\t\tt.Fatalf("expected app metric, got %q", resp.Items[0].Service)
\t}
\tif resp.CollectedAt != now.Add(time.Second).Format(time.RFC3339) {
\t\tt.Fatalf("unexpected collectedAt %q", resp.CollectedAt)
\t}
}

func TestMetricsSnapshotWithOnlyIdleSamplesHasNoCollectionTime(t *testing.T) {
\tresp := MetricsSnapshotFromContainers([]container.Metrics{{Service: "static", Uptime: "n/a", CollectedAt: time.Now().UTC()}})
\tif len(resp.Items) != 0 {
\t\tt.Fatalf("expected no runtime metrics, got %d", len(resp.Items))
\t}
\tif resp.CollectedAt != "" {
\t\tt.Fatalf("expected empty collectedAt, got %q", resp.CollectedAt)
\t}
}
''')

Path("frontend/src/lib/utils/project-metric-history.ts").write_text('''import type { ContainerMetrics } from '$types';

export type ProjectMetricHistory = Record<string, {
\tcpu: number[];
\tmemoryPercent: number[];
}>;

export function appendProjectMetricHistory(
\thistory: ProjectMetricHistory,
\titems: ContainerMetrics[],
\tlimit = 120
): ProjectMetricHistory {
\tconst boundedLimit = Math.max(1, Math.floor(limit));
\tconst next: ProjectMetricHistory = { ...history };
\tfor (const item of items) {
\t\tif (!item.service) continue;
\t\tconst previous = history[item.service] ?? { cpu: [], memoryPercent: [] };
\t\tconst cpu = Number.isFinite(item.cpu) ? Math.max(0, item.cpu) : 0;
\t\tconst memoryPercent = item.memoryLimitMb > 0 && Number.isFinite(item.memoryMb)
\t\t\t? Math.max(0, (item.memoryMb / item.memoryLimitMb) * 100)
\t\t\t: 0;
\t\tnext[item.service] = {
\t\t\tcpu: [...previous.cpu, cpu].slice(-boundedLimit),
\t\t\tmemoryPercent: [...previous.memoryPercent, memoryPercent].slice(-boundedLimit)
\t\t};
\t}
\treturn next;
}
''')

Path("frontend/src/lib/utils/project-metric-history.test.ts").write_text('''import { describe, expect, it } from 'vitest';

import { appendProjectMetricHistory } from './project-metric-history';

const metric = (service: string, cpu: number, memoryMb: number, memoryLimitMb = 100) => ({
\tservice,
\tcpu,
\tmemoryMb,
\tmemoryLimitMb,
\tuptime: '1m'
});

describe('appendProjectMetricHistory', () => {
\tit('retains repeated samples instead of only value changes', () => {
\t\tlet history = appendProjectMetricHistory({}, [metric('app', 2, 10)], 3);
\t\thistory = appendProjectMetricHistory(history, [metric('app', 2, 10)], 3);
\t\texpect(history.app.cpu).toEqual([2, 2]);
\t\texpect(history.app.memoryPercent).toEqual([10, 10]);
\t});

\tit('keeps independent bounded history per service', () => {
\t\tlet history = appendProjectMetricHistory({}, [metric('api', 1, 10), metric('worker', 4, 40)], 2);
\t\thistory = appendProjectMetricHistory(history, [metric('api', 2, 20)], 2);
\t\thistory = appendProjectMetricHistory(history, [metric('api', 3, 30)], 2);
\t\texpect(history.api.cpu).toEqual([2, 3]);
\t\texpect(history.worker.cpu).toEqual([4]);
\t});
});
''')

replace_once(
    "frontend/src/routes/projects/+page.svelte",
    "\t$: getDerivedStatus = (project: Project) => {\n\t\tif (project.status === 'running' && projectUptimes[project.id] === '-') return 'crashed';\n\t\treturn project.status;\n\t};",
    "\t$: getDerivedStatus = (project: Project) => {\n\t\tif (project.deployMode !== 'static' && project.status === 'running' && projectUptimes[project.id] === '-') return 'crashed';\n\t\treturn project.status;\n\t};",
)
replace_once(
    "frontend/src/routes/projects/+page.svelte",
    "\tasync function loadUptimesFor(rows: Project[]) {\n\t\tconst pending = rows.filter((project) => !(project.id in projectUptimes) && !uptimeLoadingIds.has(project.id));",
    "\tasync function loadUptimesFor(rows: Project[]) {\n\t\tconst staticUptimes = Object.fromEntries(rows.filter((project) => project.deployMode === 'static').map((project) => [project.id, 'n/a']));\n\t\tif (Object.keys(staticUptimes).length > 0) projectUptimes = { ...projectUptimes, ...staticUptimes };\n\t\tconst pending = rows.filter((project) => project.deployMode !== 'static' && !(project.id in projectUptimes) && !uptimeLoadingIds.has(project.id));",
)

replace_once("frontend/src/routes/projects/[id]/+page.svelte", "\timport ResourceMeter from '$components/ResourceMeter.svelte';\n", "")
replace_once(
    "frontend/src/routes/projects/[id]/+page.svelte",
    "\t$: memoryPercent = primaryMetric && primaryMetric.memoryLimitMb > 0\n\t\t? Math.min((primaryMetric.memoryMb / primaryMetric.memoryLimitMb) * 100, 100)\n\t\t: 0;\n\t$: cpuPercent = primaryMetric ? Math.min(primaryMetric.cpu, 100) : 0;\n",
    "",
)
replace_once(
    "frontend/src/routes/projects/[id]/+page.svelte",
    "\t\t{#if project.status !== 'running'}",
    "\t\t{#if project.status === 'building' || project.status === 'crashed' || project.status === 'pending'}",
)
p = Path("frontend/src/routes/projects/[id]/+page.svelte")
text = p.read_text()
title = text.index(">Runtime resources</h2>")
start = text.rfind('\t\t<section class="surface overflow-hidden">', 0, title)
end_marker = '\n\t\t<div class="grid gap-4 lg:grid-cols-[minmax(0,1.25fr)_minmax(20rem,0.75fr)]">'
end = text.index(end_marker, title)
replacement = '''\t\t{#if project.deployMode !== 'static'}
\t\t\t<section class="surface overflow-hidden">
\t\t\t\t<div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-neutral-800">
\t\t\t\t\t<div>
\t\t\t\t\t\t<h2 class="text-sm font-semibold text-gray-950 dark:text-white">Runtime summary</h2>
\t\t\t\t\t\t<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{runtimeSummary.label} · {metricsUpdatedLabel}</p>
\t\t\t\t\t</div>
\t\t\t\t\t<ActionLink href={`${base}/metrics`} variant="ghost" size="xs">
\t\t\t\t\t\t<Activity slot="icon" class="h-3.5 w-3.5" />
\t\t\t\t\t\tDiagnostics
\t\t\t\t\t</ActionLink>
\t\t\t\t</div>
\t\t\t\t{#if primaryMetric}
\t\t\t\t\t<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 sm:grid-cols-3 sm:divide-x sm:divide-y-0">
\t\t\t\t\t\t<div class="p-5">
\t\t\t\t\t\t\t<p class="metric-label">CPU</p>
\t\t\t\t\t\t\t<p class="metric-value mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primaryMetric.cpu.toFixed(2)}%</p>
\t\t\t\t\t\t\t<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Current runtime sample</p>
\t\t\t\t\t\t</div>
\t\t\t\t\t\t<div class="p-5">
\t\t\t\t\t\t\t<p class="metric-label">Memory</p>
\t\t\t\t\t\t\t<p class="metric-value mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primaryMetric.memoryMb.toFixed(1)} <span class="text-sm font-medium text-gray-500 dark:text-gray-400">MB</span></p>
\t\t\t\t\t\t\t<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{primaryMetric.memoryLimitMb.toFixed(0)} MB limit</p>
\t\t\t\t\t\t</div>
\t\t\t\t\t\t<div class="p-5">
\t\t\t\t\t\t\t<p class="metric-label">Uptime</p>
\t\t\t\t\t\t\t<p class="metric-value mt-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">{primaryMetric.uptime}</p>
\t\t\t\t\t\t\t<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{primaryMetric.service}</p>
\t\t\t\t\t\t</div>
\t\t\t\t\t</div>
\t\t\t\t{:else}
\t\t\t\t\t<div class="px-5 py-8">
\t\t\t\t\t\t<EmptyState title="No active runtime metrics." description="CPU and memory samples are reported only while a container runtime is active." compact />
\t\t\t\t\t</div>
\t\t\t\t{/if}
\t\t\t</section>
\t\t{/if}
'''
p.write_text(text[:start] + replacement + text[end:])

replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    "\timport { api } from '$api';\n",
    "\timport { api } from '$api';\n\timport { appendProjectMetricHistory, type ProjectMetricHistory } from '$lib/utils/project-metric-history';\n",
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    "\tlet error = '';\n",
    "\tlet error = '';\n\tlet metricHistory: ProjectMetricHistory = {};\n",
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    "\t$: primary = metricItems.find((item) => item.service === selectedService) ?? metricItems[0] ?? null;\n",
    "\t$: primary = metricItems.find((item) => item.service === selectedService) ?? metricItems[0] ?? null;\n\t$: primaryHistory = primary ? (metricHistory[primary.service] ?? { cpu: [], memoryPercent: [] }) : { cpu: [], memoryPercent: [] };\n",
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    "\t\t\tconst result = await api.metrics.snapshot($page.params.id ?? '');\n\t\t\tconst nextServices = result.items.map((item) => item.service);",
    "\t\t\tconst result = await api.metrics.snapshot($page.params.id ?? '');\n\t\t\tmetricHistory = appendProjectMetricHistory(metricHistory, result.items);\n\t\t\tconst nextServices = result.items.map((item) => item.service);",
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    "\t\t\t\t{ label: 'Telemetry', value: 'statd preferred' },\n\t\t\t\t{ label: 'Persistent storage', value: 'Not measured' }\n",
    "\t\t\t\t{ label: 'Telemetry', value: 'statd preferred' }\n",
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    '<CapacityMetricChart label="CPU" value={`${primary.cpu.toFixed(2)}%`} detail="current runtime sample" percent={cpuPercent} resource="cpu" className="bg-white dark:bg-neutral-900" />',
    '<CapacityMetricChart label="CPU" value={`${primary.cpu.toFixed(2)}%`} detail="rolling samples · current runtime" percent={cpuPercent} series={primaryHistory.cpu} resource="cpu" className="bg-white dark:bg-neutral-900" />',
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    '<CapacityMetricChart label="Memory" value={`${primary.memoryMb.toFixed(1)} MB`} detail={`${primary.memoryLimitMb.toFixed(0)} MB limit`} percent={Math.min(memoryPercent, 100)} resource="memory" className="bg-white dark:bg-neutral-900" />',
    '<CapacityMetricChart label="Memory" value={`${primary.memoryMb.toFixed(1)} MB`} detail={`${primary.memoryLimitMb.toFixed(0)} MB limit · rolling usage`} percent={Math.min(memoryPercent, 100)} series={primaryHistory.memoryPercent} resource="memory" className="bg-white dark:bg-neutral-900" />',
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    '\n\t\t\t\t\t<p class="mt-3 text-[11px] leading-4 text-gray-500 dark:text-gray-400">Persistent storage means project-owned managed data. Host root-disk usage is intentionally not substituted here.</p>',
    "",
)
replace_once(
    "frontend/src/routes/projects/[id]/metrics/+page.svelte",
    '<EmptyState title="No metrics yet." description="Metrics appear after the project has a running container or service." compact />',
    '<EmptyState title="No active runtime metrics." description="Static projects and stopped runtimes do not expose container CPU or memory samples. Edge analytics remains available when Cloudflare is configured." compact />',
)
