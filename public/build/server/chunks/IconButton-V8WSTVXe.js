import { r as fallback, n as attr, k as attr_class, w as clsx, d as slot, t as bind_props } from './renderer-EjaZHhrY.js';

/* empty css                                         */
function IconButton($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let variantClass, isUnavailable, effectiveHref, disabledClass, controlClass;
    let label = $$props["label"];
    let href = fallback($$props["href"], "");
    let type = fallback($$props["type"], "button");
    let variant = fallback($$props["variant"], "default");
    let disabled = fallback($$props["disabled"], false);
    let loading = fallback($$props["loading"], false);
    let external = fallback($$props["external"], false);
    let className = fallback($$props["className"], "");
    variantClass = {
      default: "border-gray-200 bg-white text-gray-600 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-950 dark:border-gray-800 dark:bg-gray-950/80 dark:text-gray-300 dark:hover:border-gray-700 dark:hover:bg-gray-900 dark:hover:text-white",
      primary: "border-brand-700 bg-brand-700 text-white hover:border-brand-900 hover:bg-brand-900 dark:border-brand-500 dark:bg-brand-500 dark:text-gray-950 dark:hover:border-brand-100 dark:hover:bg-brand-100",
      brand: "border-brand-100 bg-brand-50 text-brand-700 hover:border-brand-500/40 hover:bg-brand-100 hover:text-brand-900 dark:border-brand-500/35 dark:bg-brand-500/10 dark:text-brand-500 dark:hover:border-brand-500/50 dark:hover:bg-brand-500/15 dark:hover:text-brand-100",
      danger: "border-red-200 bg-white text-red-600 hover:border-red-300 hover:bg-red-50 hover:text-red-700 dark:border-red-900/70 dark:bg-gray-950 dark:text-red-300 dark:hover:bg-red-950/30",
      ghost: "border-transparent bg-transparent text-gray-500 hover:border-gray-200 hover:bg-gray-100 hover:text-gray-950 dark:text-gray-400 dark:hover:border-gray-800 dark:hover:bg-gray-900 dark:hover:text-white"
    }[variant];
    isUnavailable = disabled || loading;
    effectiveHref = isUnavailable ? void 0 : href;
    disabledClass = isUnavailable ? "cursor-not-allowed opacity-50" : "";
    controlClass = `inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white active:translate-y-px disabled:translate-y-0 dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950 ${variantClass} ${disabledClass} ${className}`;
    if (href) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<a${attr("href", effectiveHref)}${attr_class(clsx(controlClass), "svelte-11linj7")} data-icon-button=""${attr("aria-label", label)}${attr("aria-disabled", isUnavailable)}${attr("aria-busy", loading)}${attr("tabindex", isUnavailable ? -1 : void 0)}${attr("title", label)}${attr("target", external && !isUnavailable ? "_blank" : void 0)}${attr("rel", external && !isUnavailable ? "noopener" : void 0)}>`);
      if (loading) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>`);
      } else {
        $$renderer2.push("<!--[-1-->");
        $$renderer2.push(`<!--[-->`);
        slot($$renderer2, $$props, "default", {});
        $$renderer2.push(`<!--]-->`);
      }
      $$renderer2.push(`<!--]--></a>`);
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<button${attr("type", type)}${attr_class(clsx(controlClass), "svelte-11linj7")} data-icon-button=""${attr("aria-label", label)}${attr("aria-busy", loading)}${attr("title", label)}${attr("disabled", isUnavailable, true)}>`);
      if (loading) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>`);
      } else {
        $$renderer2.push("<!--[-1-->");
        $$renderer2.push(`<!--[-->`);
        slot($$renderer2, $$props, "default", {});
        $$renderer2.push(`<!--]-->`);
      }
      $$renderer2.push(`<!--]--></button>`);
    }
    $$renderer2.push(`<!--]-->`);
    bind_props($$props, {
      label,
      href,
      type,
      variant,
      disabled,
      loading,
      external,
      className
    });
  });
}

export { IconButton as I };
//# sourceMappingURL=IconButton-V8WSTVXe.js.map
