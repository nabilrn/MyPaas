<script lang="ts">
	import { Box, Check, ChevronRight, CircleDot, GitBranch, MoreHorizontal, Rocket, Terminal } from '@lucide/svelte';
</script>

<div class="preview-window" role="img" aria-label="Placeholder preview of the MyPaas project dashboard showing running services and deployment activity">
	<div class="flex h-11 items-center justify-between border-b border-gray-800 px-4">
		<div class="flex items-center gap-2"><span class="h-2.5 w-2.5 rounded-full bg-red-400/70"></span><span class="h-2.5 w-2.5 rounded-full bg-yellow-300/70"></span><span class="h-2.5 w-2.5 rounded-full bg-brand-500"></span></div>
		<span class="font-mono text-[11px] text-gray-500">dashboard.nabilrn.space</span>
		<MoreHorizontal size={16} class="text-gray-600" aria-hidden="true" />
	</div>
	<div class="grid min-h-[27rem] sm:grid-cols-[10rem_minmax(0,1fr)]">
		<aside class="hidden border-r border-gray-800 bg-gray-950/80 p-3 sm:block">
			<p class="px-2 py-2 text-xs font-semibold text-white">MyPaas</p>
			<div class="mt-3 space-y-1">
				<div class="flex items-center gap-2 rounded-md bg-brand-500/10 px-2 py-2 text-xs font-medium text-brand-100"><Box size={14} /> Projects</div>
				<div class="flex items-center gap-2 px-2 py-2 text-xs text-gray-500"><Rocket size={14} /> Deployments</div>
				<div class="flex items-center gap-2 px-2 py-2 text-xs text-gray-500"><Terminal size={14} /> Audit logs</div>
			</div>
		</aside>
		<div class="bg-gray-950 p-4 sm:p-5">
			<div class="flex flex-wrap items-start justify-between gap-3">
				<div><div class="flex items-center gap-2"><h3 class="text-sm font-semibold text-white">personal-api</h3><span class="inline-flex items-center gap-1 rounded border border-brand-500/30 bg-brand-500/10 px-1.5 py-0.5 text-[10px] font-medium text-brand-100"><CircleDot size={9} /> Running</span></div><p class="mt-1 flex items-center gap-1.5 text-xs text-gray-500"><GitBranch size={12} /> nabilrn/personal-api · main</p></div>
				<button class="rounded-md bg-brand-500 px-3 py-1.5 text-xs font-semibold text-gray-950">Deploy latest</button>
			</div>
			<div class="mt-5 grid gap-2.5 sm:grid-cols-3">
				<div class="preview-stat"><span>Runtime</span><strong>Compose</strong></div>
				<div class="preview-stat"><span>Memory</span><strong>186 / 512 MB</strong></div>
				<div class="preview-stat"><span>Uptime</span><strong>4d 18h</strong></div>
			</div>
			<div class="mt-3 overflow-hidden rounded-lg border border-gray-800 bg-gray-900/45">
				<div class="flex items-center justify-between border-b border-gray-800 px-3 py-2.5"><span class="text-xs font-semibold text-gray-200">Deployment activity</span><span class="text-[10px] text-gray-500">Live</span></div>
				{#each [['Production deploy','Ready','2m 14s'], ['Docker image built','Complete','1m 48s'], ['Caddy route updated','Active','4s']] as row}
					<div class="grid grid-cols-[1fr_auto_auto] items-center gap-3 border-b border-gray-800/70 px-3 py-3 last:border-0"><span class="text-xs text-gray-300">{row[0]}</span><span class="flex items-center gap-1 text-[10px] text-brand-100"><Check size={11} /> {row[1]}</span><span class="text-[10px] text-gray-600">{row[2]}</span></div>
				{/each}
			</div>
			<div class="mt-3 flex items-center justify-between rounded-lg border border-gray-800 px-3 py-2.5"><span class="font-mono text-[11px] text-gray-500">personal-api.nabilrn.space</span><ChevronRight size={14} class="text-gray-600" /></div>
		</div>
	</div>
</div>
