<script lang="ts">
  import AdminSidebar from "$components/AdminSidebar.svelte";
</script>

<div class="grid min-h-[calc(100vh-3.5rem)] lg:grid-cols-[12rem_minmax(0,1fr)]">
  <aside
    class="border-b border-gray-100 px-3 py-4 dark:border-neutral-900 lg:border-b-0 lg:border-r"
  >
    <AdminSidebar />
  </aside>

  <main class="min-w-0">
    <slot />
  </main>
</div>
