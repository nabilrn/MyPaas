<script lang="ts">
  import { page } from "$app/stores";
  import AdminSidebar from "$components/AdminSidebar.svelte";
  import { administrationNavItemForPath } from "$lib/navigation/administration";

  $: currentSection = administrationNavItemForPath($page.url.pathname);
</script>

<div class="grid min-h-[calc(100vh-3.5rem)] lg:grid-cols-[12rem_minmax(0,1fr)]">
  <aside
    class="border-b border-[color:var(--workspace-divider)] px-3 py-4 lg:border-b-0 lg:border-r"
  >
    <AdminSidebar />
  </aside>

  <main class="min-w-0 px-3.5 py-3">
    <div class="w-full space-y-3">
      <header class="border-b border-[color:var(--workspace-divider)] px-5 pb-3">
        <h1 class="text-lg font-semibold text-gray-950 dark:text-white">
          {currentSection.title}
        </h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {currentSection.description}
        </p>
      </header>

      <div class="admin-content min-w-0">
        <slot />
      </div>
    </div>
  </main>
</div>

<style>
  :global(.admin-content > .page-shell) {
    width: 100%;
    max-width: none;
    margin-inline: 0;
    padding-inline: 0;
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 0;
  }

  :global(.admin-content > .page-shell > section.rounded-lg) {
    border-left: 0 !important;
    border-right: 0 !important;
    border-radius: 0 !important;
  }

  /*
   * Administration follows the same horizontal rhythm as project Overview,
   * Deployments, and Logs: the route surface/dividers stay full width while
   * readable headings use the 20px header gutter and row content uses 16px.
   */
  :global(.admin-content > .page-shell > section > h2) {
    padding-inline: 1.25rem;
  }

  :global(.admin-content > .page-shell > section.rounded-lg > .divide-y > div),
  :global(.admin-content > .page-shell > section > .border-y > .grid),
  :global(.admin-content > .page-shell > section > .border-y > .flex),
  :global(.admin-content > .page-shell > section > .border-y > .border-t),
  :global(.admin-content > .page-shell > section > .border-y.grid) {
    padding-left: 1rem !important;
    padding-right: 1rem !important;
  }

  :global(.admin-content > .page-shell > details) {
    padding-left: 1rem;
    padding-right: 1rem;
  }

  :global(.admin-content > .page-shell > section + section),
  :global(.admin-content > .page-shell > section + details),
  :global(.admin-content > .page-shell > details + section),
  :global(.admin-content > .page-shell > details + details) {
    margin-top: 1rem;
  }
</style>
