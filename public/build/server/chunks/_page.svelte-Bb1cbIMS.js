import { c as store_get, h as head, f as unsubscribe_stores, n as attr, m as escape_html, j as ensure_array_like, x as attr_style, k as attr_class, l as stringify, r as fallback, t as bind_props } from './renderer-EjaZHhrY.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';
import { p as page } from './stores-BdUEhWdH.js';
import { A as ActionButton } from './ActionButton-WfeBQzx6.js';
import { B as Breadcrumbs } from './Breadcrumbs-s_0YAf0V.js';
import { I as IconButton } from './IconButton-V8WSTVXe.js';
import { P as PageHeader } from './PageHeader-CxbUKF9J.js';
import { S as SectionPanel } from './SectionPanel-DrTEgZNe.js';
import './toast-4wUWO2xn.js';
import { a as projectHost, p as projectURL } from './urls-CHcPIehM.js';
import { U as Upload } from './upload-BW7UXErS.js';
import { X } from './x-CB0QdrYJ.js';
import './Icon-9ZFIo3zS.js';
import './index-CjGMQA9M.js';

function SegmentedChoice($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let value = fallback($$props["value"], "");
    let options = fallback($$props["options"], () => [], true);
    let label = fallback($$props["label"], "");
    $$renderer2.push(`<div class="grid gap-2 [grid-template-columns:repeat(auto-fit,minmax(10rem,1fr))]" role="group"${attr("aria-label", label || void 0)}><!--[-->`);
    const each_array = ensure_array_like(options);
    for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
      let option = each_array[$$index];
      const selected = option.value === value;
      $$renderer2.push(`<button type="button"${attr("disabled", option.disabled, true)}${attr("aria-pressed", selected)}${attr_class(`min-h-16 rounded-md border p-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-600 focus-visible:ring-offset-2 focus-visible:ring-offset-white disabled:cursor-not-allowed disabled:opacity-50 dark:focus-visible:ring-brand-500 dark:focus-visible:ring-offset-gray-950 ${stringify(selected ? "border-brand-500 bg-brand-50 text-brand-900 dark:border-brand-500/50 dark:bg-brand-500/10 dark:text-brand-100" : "border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950/80 dark:text-gray-300 dark:hover:border-gray-700 dark:hover:bg-gray-900")}`)}><span class="block text-sm font-semibold">${escape_html(option.label)}</span> `);
      if (option.description) {
        $$renderer2.push("<!--[0-->");
        $$renderer2.push(`<span class="mt-1 block text-xs opacity-75">${escape_html(option.description)}</span>`);
      } else {
        $$renderer2.push("<!--[-1-->");
      }
      $$renderer2.push(`<!--]--></button>`);
    }
    $$renderer2.push(`<!--]--></div>`);
    bind_props($$props, { value, options, label });
  });
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    var $$store_subs;
    let previewHost, selectedProfile, managedDatabaseUrl, effectiveAppPort, deployModeOptions, portStateLabel, composeBlockingIssues, envDraftValueByKey, normalizedComposeRequiredEnvKeys, missingRequiredEnvKeys, composeDisabledReason, canSubmit, createDisabledReason, reviewStateLabel, detectionStateLabel, detectionStateBody;
    const DEFAULT_APP_PORT = "3000";
    const breadcrumbs = [
      { label: "Projects", href: "/projects" },
      { label: "New project" }
    ];
    let submitting = false;
    let detecting = false;
    let inspectingRepo = false;
    let branchOptions = [];
    let defaultBranch = "";
    let repoTree = [];
    let envDrafts = [];
    let newEnvKey = "";
    let form = {
      name: "",
      repoUrl: "",
      branch: "",
      deployMode: "auto",
      appPort: "",
      resourceProfile: "node-python",
      memoryMb: "256",
      cpuLimit: "0.35",
      sharedPostgres: false
    };
    const deployModes = [
      { id: "auto", title: "Auto", body: "Detect" },
      { id: "dockerfile", title: "Dockerfile", body: "Single app" },
      { id: "compose", title: "Compose", body: "Multi-service" },
      { id: "static", title: "Static", body: "File server" }
    ];
    const resourceProfiles = [
      {
        id: "node-python",
        title: "Node/Python",
        memoryMb: "256",
        cpuLimit: "0.35"
      },
      {
        id: "go-small",
        title: "Go small",
        memoryMb: "128",
        cpuLimit: "0.2"
      },
      {
        id: "compose-main",
        title: "Compose main",
        memoryMb: "256",
        cpuLimit: "0.35"
      },
      {
        id: "static",
        title: "Static/no-runtime",
        memoryMb: "64",
        cpuLimit: "0.1"
      },
      {
        id: "custom",
        title: "Custom",
        memoryMb: "512",
        cpuLimit: "0.5"
      }
    ];
    function normalizeEnvKey(value) {
      return value.trim().toUpperCase().replace(/[^A-Z0-9_]/g, "_");
    }
    previewHost = projectHost("your-app", store_get($$store_subs ??= {}, "$page", page).url.hostname);
    projectURL("your-app", store_get($$store_subs ??= {}, "$page", page).url.protocol, store_get($$store_subs ??= {}, "$page", page).url.hostname);
    selectedProfile = resourceProfiles.find((profile) => profile.id === form.resourceProfile);
    managedDatabaseUrl = form.sharedPostgres;
    effectiveAppPort = DEFAULT_APP_PORT;
    deployModeOptions = deployModes.map((mode) => ({ value: mode.id, label: mode.title, description: mode.body }));
    portStateLabel = "Fallback if detection finds no port";
    composeBlockingIssues = [];
    envDraftValueByKey = new Map(envDrafts.map((item) => [normalizeEnvKey(item.key), item.value]).filter(([key]) => Boolean(key)));
    normalizedComposeRequiredEnvKeys = Array.from(new Set([].map(normalizeEnvKey).filter(Boolean)));
    missingRequiredEnvKeys = normalizedComposeRequiredEnvKeys.filter((key) => !(managedDatabaseUrl && key === "DATABASE_URL")).filter((key) => !((envDraftValueByKey.get(key)?.trim()?.length ?? 0) > 0));
    composeDisabledReason = composeBlockingIssues[0]?.message ?? (missingRequiredEnvKeys.length > 0 ? `Fill required env values: ${missingRequiredEnvKeys.slice(0, 3).join(", ")}${missingRequiredEnvKeys.length > 3 ? "..." : ""}` : "");
    canSubmit = Boolean(form.name.trim() && form.repoUrl.trim() && form.branch.trim() && !composeDisabledReason && !submitting && !detecting && !inspectingRepo);
    createDisabledReason = !form.name.trim() ? "Project name is required" : !form.repoUrl.trim() ? "Repository URL is required" : !form.branch.trim() ? "Branch is required" : composeDisabledReason ? composeDisabledReason : "";
    reviewStateLabel = canSubmit ? "Ready to create" : createDisabledReason || "Complete required fields";
    detectionStateLabel = form.repoUrl.trim() ? form.branch.trim() ? "Ready for detection" : "Select a branch" : "Waiting for repository URL";
    detectionStateBody = form.repoUrl.trim() ? form.branch.trim() ? "Run detection to fill runtime, port, service, and discovered environment defaults." : "Branches load automatically after the repository URL is entered." : "Paste a repository URL before running detection.";
    head("1ytgd2c", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>New project · MyPaas</title>`);
      });
    });
    $$renderer2.push(`<div class="page-shell py-6">`);
    Breadcrumbs($$renderer2, { items: breadcrumbs });
    $$renderer2.push(`<!----> `);
    PageHeader($$renderer2, {
      title: "New project",
      description: "Create a routable deployment target from a Git repository."
    });
    $$renderer2.push(`<!----> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_24rem]"><form class="space-y-5">`);
    SectionPanel($$renderer2, {
      title: "Repository source",
      description: "Name the route, load repository branches, and preview the selected branch structure.",
      children: ($$renderer3) => {
        $$renderer3.push(`<div class="grid gap-4"><div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="name">Project name</label> <input id="name" type="text"${attr("value", form.name)} placeholder="my-app" class="field w-full"/></div> <div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="repo">Repository URL</label> <input id="repo" type="text"${attr("value", form.repoUrl)} placeholder="https://github.com/username/repo" class="field w-full font-mono"/></div> <div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="branch">Branch</label> <div class="flex flex-col gap-2 sm:flex-row">`);
        $$renderer3.select(
          {
            id: "branch",
            value: form.branch,
            class: "field min-w-0 flex-1 font-mono",
            disabled: !branchOptions.length && true
          },
          ($$renderer4) => {
            $$renderer4.option({ value: "", disabled: true }, ($$renderer5) => {
              $$renderer5.push(`${escape_html("Select branch")}`);
            });
            $$renderer4.push(`<!--[-->`);
            const each_array = ensure_array_like(branchOptions);
            for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
              let branch = each_array[$$index];
              $$renderer4.option({ value: branch }, ($$renderer5) => {
                $$renderer5.push(`${escape_html(branch)}${escape_html(branch === defaultBranch ? " (default)" : "")}`);
              });
            }
            $$renderer4.push(`<!--]-->`);
          }
        );
        $$renderer3.push(` `);
        ActionButton($$renderer3, {
          variant: "secondary",
          type: "button",
          disabled: !form.repoUrl.trim(),
          loading: inspectingRepo,
          loadingLabel: "Loading...",
          children: ($$renderer4) => {
            $$renderer4.push(`<!---->Refresh`);
          },
          $$slots: { default: true }
        });
        $$renderer3.push(`<!----></div></div></div> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> <div class="mt-4"><div class="mb-2 flex items-center justify-between gap-3"><p class="text-xs font-medium text-gray-600 dark:text-gray-300">Repository structure</p> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--></div> <div class="max-h-72 overflow-auto rounded-md border border-gray-200 bg-white text-xs dark:border-gray-800 dark:bg-gray-950">`);
        if (repoTree.length > 0) {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<!--[-->`);
          const each_array_1 = ensure_array_like(repoTree);
          for (let $$index_1 = 0, $$length = each_array_1.length; $$index_1 < $$length; $$index_1++) {
            let item = each_array_1[$$index_1];
            $$renderer3.push(`<div class="grid grid-cols-[2.75rem_minmax(0,1fr)] items-center gap-2 border-b border-gray-100 px-3 py-1.5 last:border-b-0 dark:border-gray-900"${attr_style(`padding-left: ${0.75 + item.depth * 0.9}rem;`)}><span class="rounded border border-gray-200 px-1.5 py-0.5 text-[10px] uppercase text-gray-500 dark:border-gray-800 dark:text-gray-400">${escape_html(item.type === "directory" ? "dir" : "file")}</span> <span${attr_class(`truncate font-mono ${stringify(item.type === "directory" ? "font-medium text-gray-950 dark:text-white" : "text-gray-600 dark:text-gray-300")}`)}>${escape_html(item.path)}</span></div>`);
          }
          $$renderer3.push(`<!--]-->`);
        } else {
          $$renderer3.push("<!--[-1-->");
          $$renderer3.push(`<p class="px-3 py-4 text-sm text-gray-500 dark:text-gray-400">${escape_html("Repository structure appears after branches load.")}</p>`);
        }
        $$renderer3.push(`<!--]--></div></div>`);
      },
      $$slots: { default: true }
    });
    $$renderer2.push(`<!----> `);
    SectionPanel($$renderer2, {
      title: "Runtime and entrypoint",
      description: "Use detection for repository defaults, then override only the values that need to be explicit.",
      children: ($$renderer3) => {
        $$renderer3.push(`<div class="space-y-4"><div class="rounded-md border border-gray-200 bg-gray-50 px-3 py-3 text-sm dark:border-gray-800 dark:bg-gray-950/60" aria-live="polite"><div class="flex gap-3"><span${attr_class(`mt-1 h-2.5 w-2.5 shrink-0 rounded-full ${stringify("bg-gray-400 dark:bg-gray-600")}`)}></span> <div class="min-w-0"><p class="font-medium text-gray-950 dark:text-white">${escape_html(detectionStateLabel)}</p> <p class="mt-0.5 text-xs leading-5 text-gray-500 dark:text-gray-400">${escape_html(detectionStateBody)}</p></div></div></div> `);
        SegmentedChoice($$renderer3, {
          label: "Deployment mode",
          value: form.deployMode,
          options: deployModeOptions
        });
        $$renderer3.push(`<!----> <div class="grid gap-4 sm:grid-cols-2">`);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> `);
        {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="appPort">App port</label> <input id="appPort" type="number" min="1" max="65535"${attr("value", form.appPort)}${attr("placeholder", DEFAULT_APP_PORT)} class="field w-full font-mono"/> <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">${escape_html(portStateLabel)}</p></div>`);
        }
        $$renderer3.push(`<!--]--></div> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--></div>`);
      },
      $$slots: {
        default: true,
        actions: ($$renderer3) => {
          {
            ActionButton($$renderer3, {
              variant: "secondary",
              type: "button",
              disabled: !form.repoUrl.trim() || !form.branch.trim(),
              loading: detecting,
              loadingLabel: "Detecting...",
              children: ($$renderer4) => {
                $$renderer4.push(`<!---->Detect runtime`);
              },
              $$slots: { default: true }
            });
          }
        }
      }
    });
    $$renderer2.push(`<!----> `);
    SectionPanel($$renderer2, {
      title: "Resources",
      description: "Keep defaults small for the self-hosted VM quota, or switch to custom values when needed.",
      children: ($$renderer3) => {
        $$renderer3.push(`<div class="grid gap-4 sm:grid-cols-3"><div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="profile">Profile</label> `);
        $$renderer3.select(
          {
            id: "profile",
            value: form.resourceProfile,
            class: "field w-full"
          },
          ($$renderer4) => {
            $$renderer4.push(`<!--[-->`);
            const each_array_5 = ensure_array_like(resourceProfiles);
            for (let $$index_5 = 0, $$length = each_array_5.length; $$index_5 < $$length; $$index_5++) {
              let profile = each_array_5[$$index_5];
              $$renderer4.option({ value: profile.id }, ($$renderer5) => {
                $$renderer5.push(`${escape_html(profile.title)}`);
              });
            }
            $$renderer4.push(`<!--]-->`);
          }
        );
        $$renderer3.push(`</div> <div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="memory">Memory</label> `);
        $$renderer3.select({ id: "memory", value: form.memoryMb, class: "field w-full" }, ($$renderer4) => {
          $$renderer4.option({ value: "64" }, ($$renderer5) => {
            $$renderer5.push(`64 MB`);
          });
          $$renderer4.option({ value: "128" }, ($$renderer5) => {
            $$renderer5.push(`128 MB`);
          });
          $$renderer4.option({ value: "256" }, ($$renderer5) => {
            $$renderer5.push(`256 MB`);
          });
          $$renderer4.option({ value: "512" }, ($$renderer5) => {
            $$renderer5.push(`512 MB`);
          });
          $$renderer4.option({ value: "1024" }, ($$renderer5) => {
            $$renderer5.push(`1024 MB`);
          });
          $$renderer4.option({ value: "2048" }, ($$renderer5) => {
            $$renderer5.push(`2048 MB`);
          });
        });
        $$renderer3.push(`</div> <div><label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="cpu">CPU</label> `);
        $$renderer3.select({ id: "cpu", value: form.cpuLimit, class: "field w-full" }, ($$renderer4) => {
          $$renderer4.option({ value: "0.1" }, ($$renderer5) => {
            $$renderer5.push(`0.10`);
          });
          $$renderer4.option({ value: "0.2" }, ($$renderer5) => {
            $$renderer5.push(`0.20`);
          });
          $$renderer4.option({ value: "0.25" }, ($$renderer5) => {
            $$renderer5.push(`0.25`);
          });
          $$renderer4.option({ value: "0.35" }, ($$renderer5) => {
            $$renderer5.push(`0.35`);
          });
          $$renderer4.option({ value: "0.5" }, ($$renderer5) => {
            $$renderer5.push(`0.50`);
          });
          $$renderer4.option({ value: "1" }, ($$renderer5) => {
            $$renderer5.push(`1.00`);
          });
          $$renderer4.option({ value: "2" }, ($$renderer5) => {
            $$renderer5.push(`2.00`);
          });
        });
        $$renderer3.push(`</div></div> <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">Changing memory or CPU directly switches the profile to Custom.</p>`);
      },
      $$slots: { default: true }
    });
    $$renderer2.push(`<!----> `);
    SectionPanel($$renderer2, {
      title: "Environment",
      description: "Add only the variables this project needs. Keys are normalized before create.",
      children: ($$renderer3) => {
        $$renderer3.push(`<div><div class="overflow-hidden rounded-md border border-gray-200 dark:border-gray-800"><div class="hidden gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 text-[11px] font-medium text-gray-500 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400 lg:grid lg:grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem]"><span>Key</span> <span>Value</span> <span>Source</span> <span></span></div> `);
        if (managedDatabaseUrl) {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<div class="grid gap-2 border-b border-gray-100 px-3 py-3 dark:border-gray-800 lg:grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem] lg:items-center"><p class="truncate font-mono text-sm font-medium text-gray-950 dark:text-white">DATABASE_URL</p> <input value="Generated on create" disabled="" class="field w-full opacity-70"/> <span class="truncate text-xs text-gray-500 dark:text-gray-400"><span class="lg:hidden">Source:</span>managed</span> <span></span></div>`);
        } else {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> <!--[-->`);
        const each_array_6 = ensure_array_like(envDrafts);
        for (let index = 0, $$length = each_array_6.length; index < $$length; index++) {
          let draft = each_array_6[index];
          $$renderer3.push(`<div class="grid gap-2 border-b border-gray-100 px-3 py-3 last:border-b-0 dark:border-gray-800 lg:grid-cols-[minmax(8rem,1fr)_minmax(10rem,1.4fr)_6rem_2rem] lg:items-center"><input${attr("value", draft.key)} class="field w-full font-mono uppercase"/> <input${attr("type", draft.sensitive ? "password" : "text")}${attr("value", draft.value)}${attr("placeholder", draft.defaultValue ? `sample: ${draft.defaultValue}` : "")} class="field w-full font-mono"/> <span class="truncate text-xs text-gray-500 dark:text-gray-400"${attr("title", draft.source)}><span class="lg:hidden">Source:</span>${escape_html(draft.source)}</span> `);
          IconButton($$renderer3, {
            label: `Remove ${draft.key || "environment variable"}`,
            variant: "ghost",
            type: "button",
            children: ($$renderer4) => {
              X($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
            },
            $$slots: { default: true }
          });
          $$renderer3.push(`<!----></div>`);
        }
        $$renderer3.push(`<!--]--> `);
        if (envDrafts.length === 0 && !managedDatabaseUrl) {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<p class="px-3 py-4 text-sm text-gray-500 dark:text-gray-400">No project environment variables configured.</p>`);
        } else {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--></div> <div class="mt-3 flex gap-2"><input${attr("value", newEnvKey)} placeholder="ENV_KEY" class="field min-w-0 flex-1 font-mono uppercase"/> `);
        ActionButton($$renderer3, {
          type: "button",
          variant: "secondary",
          children: ($$renderer4) => {
            $$renderer4.push(`<!---->Add`);
          },
          $$slots: { default: true }
        });
        $$renderer3.push(`<!----></div></div>`);
      },
      $$slots: {
        default: true,
        actions: ($$renderer3) => {
          {
            $$renderer3.push(`<div class="flex flex-wrap items-center gap-2"><input type="file" accept=".env,text/plain" class="hidden"/> `);
            ActionButton($$renderer3, {
              type: "button",
              variant: "secondary",
              size: "xs",
              children: ($$renderer4) => {
                $$renderer4.push(`<span class="inline-flex items-center gap-1.5">`);
                Upload($$renderer4, { class: "h-3.5 w-3.5", "aria-hidden": "true" });
                $$renderer4.push(`<!----> Import .env</span>`);
              },
              $$slots: { default: true }
            });
            $$renderer3.push(`<!----> `);
            {
              $$renderer3.push("<!--[0-->");
              $$renderer3.push(`<label class="inline-flex min-h-8 items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input type="checkbox"${attr("checked", form.sharedPostgres, true)} class="h-4 w-4 rounded border-gray-300 text-gray-950 focus:ring-gray-950 dark:border-gray-700"/> Shared PostgreSQL</label>`);
            }
            $$renderer3.push(`<!--]--></div>`);
          }
        }
      }
    });
    $$renderer2.push(`<!----></form> <aside class="lg:sticky lg:top-6 lg:self-start">`);
    SectionPanel($$renderer2, {
      title: "Review",
      description: "Confirm route, runtime, and quota before create.",
      contentClass: "p-0",
      children: ($$renderer3) => {
        $$renderer3.push(`<div class="border-b border-gray-100 px-5 py-4 dark:border-gray-800" aria-live="polite"><span${attr_class(`inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium ${stringify(canSubmit ? "border-brand-500/30 bg-brand-50 text-brand-900 dark:border-brand-500/40 dark:bg-brand-500/10 dark:text-brand-100" : "border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300")}`)}>${escape_html(reviewStateLabel)}</span> `);
        if (createDisabledReason) {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">${escape_html(createDisabledReason)}</p>`);
        } else {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--></div> <dl class="divide-y divide-gray-100 text-sm dark:divide-gray-800"><div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Subdomain</dt> <dd class="mt-1 truncate font-mono font-medium text-gray-950 dark:text-white">${escape_html(previewHost)}</dd></div> <div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Repository</dt> <dd class="mt-1 truncate font-mono text-gray-950 dark:text-white">${escape_html("-")}</dd></div> <div class="grid grid-cols-2 divide-x divide-gray-100 dark:divide-gray-800"><div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Branch</dt> <dd class="mt-1 font-mono text-gray-950 dark:text-white">${escape_html("-")}</dd></div> <div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Runtime</dt> <dd class="mt-1 font-mono text-gray-950 dark:text-white">${escape_html(form.deployMode)}</dd></div></div> <div class="grid grid-cols-3 divide-x divide-gray-100 dark:divide-gray-800"><div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Port</dt> <dd class="mt-1"><span class="font-mono text-gray-950 dark:text-white">${escape_html(effectiveAppPort)}</span> `);
        {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<span class="mt-0.5 block text-[11px] text-gray-500 dark:text-gray-400">${escape_html(portStateLabel)}</span>`);
        }
        $$renderer3.push(`<!--]--></dd></div> <div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Memory</dt> <dd class="mt-1 font-mono text-gray-950 dark:text-white">${escape_html(form.memoryMb)} MB</dd></div> <div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">CPU</dt> <dd class="mt-1 font-mono text-gray-950 dark:text-white">${escape_html(form.cpuLimit)}</dd></div></div> <div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Profile</dt> <dd class="mt-1 text-gray-950 dark:text-white">${escape_html(selectedProfile?.title ?? form.resourceProfile)}</dd></div> `);
        {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<div class="px-5 py-3"><dt class="text-xs text-gray-500 dark:text-gray-400">Database</dt> <dd class="mt-1 text-gray-950 dark:text-white">${escape_html("-")}</dd></div>`);
        }
        $$renderer3.push(`<!--]--></dl> <div class="border-t border-gray-100 p-5 dark:border-gray-800">`);
        ActionButton($$renderer3, {
          variant: "primary",
          size: "md",
          type: "button",
          full: true,
          loading: submitting,
          loadingLabel: "Detecting...",
          disabled: !canSubmit,
          children: ($$renderer4) => {
            $$renderer4.push(`<!---->Create project`);
          },
          $$slots: { default: true }
        });
        $$renderer3.push(`<!----> `);
        {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">Auto mode runs detection before the project is created.</p>`);
        }
        $$renderer3.push(`<!--]--></div>`);
      },
      $$slots: { default: true }
    });
    $$renderer2.push(`<!----></aside></div></div>`);
    if ($$store_subs) unsubscribe_stores($$store_subs);
  });
}

export { _page as default };
//# sourceMappingURL=_page.svelte-Bb1cbIMS.js.map
