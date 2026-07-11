import { i as spread_props, h as head, f as attr, s as store_get, u as unsubscribe_stores } from "../../../chunks/renderer.js";
import { t as theme } from "../../../chunks/theme.js";
import { I as Icon } from "../../../chunks/Icon.js";
function Moon($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    [
      "path",
      {
        "d": "M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401"
      }
    ]
  ];
  Icon($$renderer, spread_props([{ name: "moon" }, props, { iconNode }]));
}
function Sun($$renderer, $$props) {
  let { $$slots, $$events, ...props } = $$props;
  const iconNode = [
    ["circle", { "cx": "12", "cy": "12", "r": "4" }],
    ["path", { "d": "M12 2v2" }],
    ["path", { "d": "M12 20v2" }],
    ["path", { "d": "m4.93 4.93 1.41 1.41" }],
    ["path", { "d": "m17.66 17.66 1.41 1.41" }],
    ["path", { "d": "M2 12h2" }],
    ["path", { "d": "M20 12h2" }],
    ["path", { "d": "m6.34 17.66-1.41 1.41" }],
    ["path", { "d": "m19.07 4.93-1.41 1.41" }]
  ];
  Icon($$renderer, spread_props([{ name: "sun" }, props, { iconNode }]));
}
const logoGreen = "/_app/immutable/assets/mypaas-horizontal-transparent-green.HhLLh6MW.png";
const logoWhite = "/_app/immutable/assets/mypaas-horizontal-transparent-white.BBmvvEIA.png";
const circuitBgLight = "/_app/immutable/assets/mypaas-circuit-background.C9hLxEwf.svg";
const circuitBgDark = "/_app/immutable/assets/mypaas-circuit-background-dark.BmFLHx0I.svg";
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    head("1x05zx6", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>Sign in · MyPaas</title>`);
      });
    });
    $$renderer2.push(`<div class="login-page svelte-1x05zx6"><img${attr("src", circuitBgLight)} alt="" aria-hidden="true" class="circuit-bg pointer-events-none dark:hidden svelte-1x05zx6"/> <img${attr("src", circuitBgDark)} alt="" aria-hidden="true" class="circuit-bg pointer-events-none hidden dark:block svelte-1x05zx6"/> <button type="button" class="theme-toggle svelte-1x05zx6" aria-label="Toggle dark mode">`);
    if (store_get($$store_subs ??= {}, "$theme", theme) === "dark") {
      $$renderer2.push("<!--[0-->");
      Sun($$renderer2, { class: "h-4 w-4", "aria-hidden": "true" });
    } else {
      $$renderer2.push("<!--[-1-->");
      Moon($$renderer2, { class: "h-4 w-4", "aria-hidden": "true" });
    }
    $$renderer2.push(`<!--]--></button> <main class="login-content svelte-1x05zx6"><div class="login-inner svelte-1x05zx6"><div class="mb-6 flex flex-col items-center text-center"><div class="flex h-14 w-[200px] items-center justify-center"><img${attr("src", logoGreen)} alt="MyPaas" class="h-14 w-[200px] object-contain dark:hidden"/> <img${attr("src", logoWhite)} alt="MyPaas" class="hidden h-14 w-[200px] object-contain dark:block"/></div> <p class="mt-2 text-sm" style="color: var(--app-muted);">Self-hosted Git-based deployments.</p> <h1 class="sr-only">Sign in to MyPaas</h1></div> <div class="login-card svelte-1x05zx6"><a href="/api/auth/github/login" id="login-github-btn" class="github-btn svelte-1x05zx6"><svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"></path></svg> Continue with GitHub</a></div></div></main></div>`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}
export {
  _page as default
};
