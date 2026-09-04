<script lang="ts">
	import {
		CircleAlert,
		Database,
		FileText,
		History,
		KeyRound,
		LayoutDashboard,
		Settings2,
		Webhook
	} from '@lucide/svelte';
	import { page } from '$app/stores';
	import ProjectSecondaryNavItem from './ProjectSecondaryNavItem.svelte';

	export let projectId = '';

	type NavItem = {
		label: string;
		href: string;
		icon: any;
		exact?: boolean;
		danger?: boolean;
	};

	type NavGroup = {
		label: string;
		items: NavItem[];
	};

	let groups: NavGroup[] = [];

	$: base = `/projects/${projectId}`;
	$: pathname = $page.url.pathname;
	$: groups = [
		{
			label: 'Project',
			items: [
				{ label: 'Overview', href: base, icon: LayoutDashboard, exact: true },
				{ label: 'Deployments', href: `${base}/deployments`, icon: History },
				{ label: 'Logs', href: `${base}/logs`, icon: FileText }
			]
		},
		{
			label: 'Data',
			items: [
				{ label: 'Environment', href: `${base}/env`, icon: KeyRound },
				{ label: 'Database', href: `${base}/database`, icon: Database }
			]
		},
		{
			label: 'Configuration',
			items: [
				{ label: 'Settings', href: `${base}/settings`, icon: Settings2, exact: true },
				{ label: 'Webhook', href: `${base}/settings/webhook`, icon: Webhook }
			]
		},
		{
			label: 'Advanced',
			items: [{ label: 'Danger zone', href: `${base}/settings/danger`, icon: CircleAlert, danger: true }]
		}
	];

	function isActive(item: NavItem, currentPath: string) {
		if (item.exact) return currentPath === item.href;
		return currentPath === item.href || currentPath.startsWith(`${item.href}/`);
	}
</script>

<nav aria-label="Project navigation" class="space-y-5 lg:sticky lg:top-4">
	{#each groups as group}
		<div>
			<p class="px-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500">{group.label}</p>
			<div class="mt-2 space-y-1">
				{#each group.items as item}
					<ProjectSecondaryNavItem
						active={isActive(item, pathname)}
						danger={item.danger ?? false}
						href={item.href}
						label={item.label}
						icon={item.icon}
					/>
				{/each}
			</div>
		</div>
	{/each}
</nav>
