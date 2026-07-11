import { j as fallback, f as attr, a as attr_class, k as clsx, c as escape_html, d as slot, m as bind_props } from "./renderer.js";
/* empty css                                           */
function ActionButton($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let baseClass, sizeClass, variantClass, classes;
    let type = fallback($$props["type"], "button");
    let variant = fallback($$props["variant"], "secondary");
    let size = fallback($$props["size"], "sm");
    let loading = fallback($$props["loading"], false);
    let disabled = fallback($$props["disabled"], false);
    let full = fallback($$props["full"], false);
    let loadingLabel = fallback($$props["loadingLabel"], "");
    let ariaLabel = fallback($$props["ariaLabel"], void 0);
    let className = fallback($$props["className"], "");
    baseClass = "inline-flex min-w-0 items-center justify-center gap-2 whitespace-nowrap font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white active:translate-y-px disabled:cursor-not-allowed disabled:translate-y-0 disabled:opacity-55 dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950";
    sizeClass = {
      xs: "min-h-8 rounded-md px-2.5 py-1.5 text-xs",
      sm: "min-h-9 rounded-md px-3 py-1.5 text-sm",
      md: "min-h-10 rounded-md px-4 py-2 text-sm"
    }[size];
    variantClass = {
      primary: "bg-brand-700 text-white hover:bg-brand-900 dark:bg-brand-500 dark:text-gray-950 dark:hover:bg-brand-100",
      secondary: "border border-gray-300 bg-white text-gray-800 hover:border-gray-400 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-950/80 dark:text-gray-200 dark:hover:border-gray-600 dark:hover:bg-gray-900",
      danger: "bg-red-600 text-white hover:bg-red-700 focus-visible:ring-red-500 dark:bg-red-500 dark:text-white dark:hover:bg-red-400",
      ghost: "text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
      ghostDanger: "text-red-600 hover:bg-red-50 hover:text-red-700 focus-visible:ring-red-500 dark:text-red-300 dark:hover:bg-red-950/30 dark:hover:text-red-200"
    }[variant];
    classes = `${baseClass} ${sizeClass} ${variantClass} ${full ? "w-full" : ""} ${className}`.trim();
    $$renderer2.push(`<button${attr("type", type)}${attr_class(clsx(classes), "svelte-fm8pxm")} data-action-button=""${attr("disabled", disabled || loading, true)}${attr("aria-busy", loading)}${attr("aria-label", ariaLabel)}>`);
    if (loading) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>`);
    } else {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> <span class="min-w-0 truncate">`);
    if (loading && loadingLabel) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`${escape_html(loadingLabel)}`);
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<!--[-->`);
      slot($$renderer2, $$props, "default", {});
      $$renderer2.push(`<!--]-->`);
    }
    $$renderer2.push(`<!--]--></span></button>`);
    bind_props($$props, {
      type,
      variant,
      size,
      loading,
      disabled,
      full,
      loadingLabel,
      ariaLabel,
      className
    });
  });
}
export {
  ActionButton as A
};
