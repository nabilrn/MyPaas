<script lang="ts">
  import {
    ArrowRightLeft,
    Bot,
    ClipboardList,
    Database,
    Settings2,
    Users,
  } from "@lucide/svelte";
  import { page } from "$app/stores";
  import {
    administrationNavGroups,
    isAdministrationNavItemActive,
    type AdministrationNavKey,
  } from "$lib/navigation/administration";

  const icons = {
    general: Settings2,
    users: Users,
    backup: Database,
    migration: ArrowRightLeft,
    mcp: Bot,
    "audit-logs": ClipboardList,
  } satisfies Record<AdministrationNavKey, typeof Settings2>;

  $: pathname = $page.url.pathname;
</script>

<nav aria-label="Administration" class="space-y-5 lg:sticky lg:top-4">
  {#each administrationNavGroups as group}
    <div>
      <p
        class="px-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-400 dark:text-gray-500"
      >
        {group.label}
      </p>
      <div class="mt-2 space-y-1">
        {#each group.items as item}
          <a
            href={item.href}
            aria-current={isAdministrationNavItemActive(item, pathname)
              ? "page"
              : undefined}
            class={`app-focus flex w-full items-center gap-2.5 border-l-2 bg-transparent px-2.5 py-2 text-left text-[13px] font-medium transition-colors ${isAdministrationNavItemActive(item, pathname) ? "border-gray-950 text-gray-950 dark:border-white dark:text-white" : "border-transparent text-gray-600 hover:text-gray-950 dark:text-gray-400 dark:hover:text-white"}`}
          >
            <svelte:component
              this={icons[item.key]}
              class="h-4 w-4 shrink-0"
              aria-hidden="true"
            />
            {item.label}
          </a>
        {/each}
      </div>
    </div>
  {/each}
</nav>
