import { h as head, j as ensure_array_like, n as attr, m as escape_html, k as attr_class, l as stringify } from './renderer-EjaZHhrY.js';
import { A as ActionButton } from './ActionButton-WfeBQzx6.js';
import { I as IconButton } from './IconButton-V8WSTVXe.js';
import { P as PageHeader } from './PageHeader-CxbUKF9J.js';
import { T as TableShell, P as Pagination } from './TableShell-CE6dcEMn.js';
import './toast-4wUWO2xn.js';
import { R as Refresh_cw } from './refresh-cw-JzODzqwL.js';
import { P as Plus } from './plus-BgLmOiR6.js';
import { T as Trash_2 } from './trash-2-KJN1o-8B.js';
import './EmptyState-DcYSSZfa.js';
import './Icon-9ZFIo3zS.js';
import './index-CjGMQA9M.js';

function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let pageStart, visibleUsers, hasNext;
    const pageSize = 10;
    let users = [];
    let loading = true;
    let error = "";
    let currentPage = 0;
    let adding = false;
    let removingUserId = "";
    let confirmRemoveUserId = "";
    function initial(email) {
      return email.trim().slice(0, 1).toUpperCase() || "?";
    }
    function formatDate(value) {
      return value ? new Date(value).toLocaleDateString() : "-";
    }
    pageStart = currentPage * pageSize;
    visibleUsers = users.slice(pageStart, pageStart + pageSize);
    hasNext = pageStart + pageSize < users.length;
    let $$settled = true;
    let $$inner_renderer;
    function $$render_inner($$renderer3) {
      head("1p497kv", $$renderer3, ($$renderer4) => {
        $$renderer4.title(($$renderer5) => {
          $$renderer5.push(`<title>Users · MyPaas Admin</title>`);
        });
      });
      $$renderer3.push(`<div class="page-shell py-6">`);
      PageHeader($$renderer3, {
        title: "User whitelist",
        description: "Only listed users can sign in via GitHub OAuth.",
        $$slots: {
          actions: ($$renderer4) => {
            {
              IconButton($$renderer4, {
                label: "Refresh users",
                variant: "brand",
                loading,
                children: ($$renderer5) => {
                  Refresh_cw($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                },
                $$slots: { default: true }
              });
              $$renderer4.push(`<!----> `);
              IconButton($$renderer4, {
                label: "Add user",
                variant: "primary",
                disabled: adding,
                children: ($$renderer5) => {
                  Plus($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                },
                $$slots: { default: true }
              });
              $$renderer4.push(`<!---->`);
            }
          }
        }
      });
      $$renderer3.push(`<!----> `);
      {
        $$renderer3.push("<!--[-1-->");
      }
      $$renderer3.push(`<!--]--> `);
      TableShell($$renderer3, {
        title: "Whitelisted users",
        description: "Manage who can access the deployment control plane.",
        loading,
        loadingRows: 3,
        error,
        empty: users.length === 0,
        emptyTitle: "No users are whitelisted yet.",
        emptyDescription: "Add a collaborator or owner to allow GitHub OAuth sign-in.",
        children: ($$renderer4) => {
          $$renderer4.push(`<table class="min-w-full divide-y divide-gray-100 dark:divide-gray-800"><thead><tr class="bg-gray-50/70 dark:bg-gray-900/70"><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">User</th><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Role</th><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Last login</th><th class="px-5 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Added</th><th class="px-5 py-3"></th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-gray-800"><!--[-->`);
          const each_array = ensure_array_like(visibleUsers);
          for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
            let user = each_array[$$index];
            $$renderer4.push(`<tr class="hover:bg-gray-50/80 dark:hover:bg-gray-900/70"><td class="px-5 py-4"><div class="flex items-center gap-3">`);
            if (user.avatarUrl) {
              $$renderer4.push("<!--[0-->");
              $$renderer4.push(`<img${attr("src", user.avatarUrl)} alt="" class="h-8 w-8 rounded-full object-cover"/>`);
            } else {
              $$renderer4.push("<!--[-1-->");
              $$renderer4.push(`<div class="flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 bg-gray-50 text-xs font-semibold text-gray-500 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300">${escape_html(initial(user.email))}</div>`);
            }
            $$renderer4.push(`<!--]--> <div><p class="text-sm font-medium text-gray-950 dark:text-white">${escape_html(user.githubUsername ?? "Not logged in yet")}</p> <p class="text-xs text-gray-500 dark:text-gray-400">${escape_html(user.email)}</p></div></div></td><td class="px-5 py-4"><span${attr_class(`inline-flex rounded-md border px-2 py-1 text-xs font-medium capitalize ${stringify(user.role === "owner" ? "border-brand-500/30 bg-brand-50 text-brand-900 dark:border-brand-500/40 dark:bg-brand-500/10 dark:text-brand-100" : "border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300")}`)}>${escape_html(user.role)}</span></td><td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">${escape_html(formatDate(user.lastLoginAt))}</td><td class="px-5 py-4 text-sm text-gray-500 dark:text-gray-400">${escape_html(formatDate(user.createdAt))}</td><td class="px-5 py-4 text-right"><div class="flex justify-end gap-2">`);
            if (confirmRemoveUserId === user.id) {
              $$renderer4.push("<!--[0-->");
              ActionButton($$renderer4, {
                variant: "ghost",
                size: "xs",
                children: ($$renderer5) => {
                  $$renderer5.push(`<!---->Cancel`);
                },
                $$slots: { default: true }
              });
              $$renderer4.push(`<!----> `);
              ActionButton($$renderer4, {
                variant: "danger",
                size: "xs",
                disabled: removingUserId !== "",
                loading: removingUserId === user.id,
                loadingLabel: "Removing...",
                children: ($$renderer5) => {
                  $$renderer5.push(`<!---->Confirm`);
                },
                $$slots: { default: true }
              });
              $$renderer4.push(`<!---->`);
            } else {
              $$renderer4.push("<!--[-1-->");
              IconButton($$renderer4, {
                label: `Remove ${user.email}`,
                variant: "danger",
                disabled: removingUserId !== "",
                children: ($$renderer5) => {
                  Trash_2($$renderer5, { class: "h-4 w-4", "aria-hidden": "true" });
                },
                $$slots: { default: true }
              });
            }
            $$renderer4.push(`<!--]--></div></td></tr>`);
          }
          $$renderer4.push(`<!--]--></tbody></table>`);
        },
        $$slots: {
          default: true,
          footer: ($$renderer4) => {
            {
              Pagination($$renderer4, {
                pageSize,
                totalShown: visibleUsers.length,
                hasNext,
                loading,
                label: "Users",
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
  });
}

export { _page as default };
//# sourceMappingURL=_page.svelte-BpYB-tez.js.map
