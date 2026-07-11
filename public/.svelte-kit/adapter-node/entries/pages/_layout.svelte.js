import { e as ensure_array_like, s as store_get, a as attr_class, b as stringify, c as escape_html, u as unsubscribe_stores, h as head, d as slot, f as attr } from "../../chunks/renderer.js";
import "@sveltejs/kit/internal";
import "../../chunks/exports.js";
import "../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../chunks/root.js";
import "../../chunks/state.svelte.js";
import { p as page } from "../../chunks/stores.js";
/* empty css                                                      */
/* empty css                                                    */
import { w as writable } from "../../chunks/index.js";
import { t as theme } from "../../chunks/theme.js";
import { t as toast } from "../../chunks/toast.js";
import { X } from "../../chunks/x.js";
function initialValue() {
  return false;
}
function createSidebarStore() {
  const { subscribe, set, update } = writable(initialValue());
  return {
    subscribe,
    set(value) {
      set(value);
    },
    toggle() {
      update((value) => {
        const next = !value;
        return next;
      });
    }
  };
}
createSidebarStore();
function Toast($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    const icons = { success: "✓", error: "✕", warning: "⚠", info: "ℹ" };
    const styles = {
      success: "bg-green-50  border-green-200  text-green-800  dark:bg-green-900/30  dark:border-green-700  dark:text-green-300",
      error: "bg-red-50    border-red-200    text-red-800    dark:bg-red-900/30    dark:border-red-700    dark:text-red-300",
      warning: "bg-yellow-50 border-yellow-200 text-yellow-800 dark:bg-yellow-900/30 dark:border-yellow-700 dark:text-yellow-300",
      info: "bg-blue-50   border-blue-200   text-blue-800   dark:bg-blue-900/30   dark:border-blue-700   dark:text-blue-300"
    };
    $$renderer2.push(`<div class="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2"><!--[-->`);
    const each_array = ensure_array_like(store_get($$store_subs ??= {}, "$toast", toast));
    for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
      let t = each_array[$$index];
      $$renderer2.push(`<div${attr_class(`pointer-events-auto flex max-w-sm items-start gap-3 rounded-lg border px-4 py-3 shadow-lg ${stringify(styles[t.kind])}`)}><span class="mt-0.5 shrink-0 font-bold">${escape_html(icons[t.kind])}</span> <p class="text-sm">${escape_html(t.message)}</p> <button class="ml-auto shrink-0 opacity-60 hover:opacity-100" aria-label="Dismiss">`);
      X($$renderer2, { class: "h-4 w-4", "aria-hidden": "true" });
      $$renderer2.push(`<!----></button></div>`);
    }
    $$renderer2.push(`<!--]--></div>`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}
const faviconGreen = "/_app/immutable/assets/mypaas-icon-transparent-green.BiMig4EB.png";
const faviconWhite = "/_app/immutable/assets/mypaas-icon-transparent-white.CStI-b0H.png";
function _layout($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let isLogin;
    isLogin = store_get($$store_subs ??= {}, "$page", page).url.pathname === "/login";
    head("12qhfyh", $$renderer2, ($$renderer3) => {
      $$renderer3.push(`<link rel="icon" type="image/png"${attr("href", store_get($$store_subs ??= {}, "$theme", theme) === "dark" ? faviconWhite : faviconGreen)}/>`);
    });
    if (isLogin) {
      $$renderer2.push("<!--[0-->");
      {
        $$renderer2.push("<!--[-1-->");
        $$renderer2.push(`<main class="min-h-screen"><!--[-->`);
        slot($$renderer2, $$props, "default", {});
        $$renderer2.push(`<!--]--></main>`);
      }
      $$renderer2.push(`<!--]-->`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> `);
    Toast($$renderer2);
    $$renderer2.push(`<!---->`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}
export {
  _layout as default
};
