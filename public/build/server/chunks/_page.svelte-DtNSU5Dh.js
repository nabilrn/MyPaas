import { h as head, m as escape_html, j as ensure_array_like } from './renderer-EjaZHhrY.js';
import './root-d39-B9S_.js';
import './state.svelte-xyT85yFW.js';
import { A as ActionButton } from './ActionButton-WfeBQzx6.js';
import { I as IconButton } from './IconButton-V8WSTVXe.js';
import { S as SectionPanel } from './SectionPanel-DrTEgZNe.js';
import './toast-4wUWO2xn.js';
import { P as Plus } from './plus-BgLmOiR6.js';
import './index-CjGMQA9M.js';
import './Icon-9ZFIo3zS.js';

const KEY_PATTERN = /^[A-Z_][A-Z0-9_]*$/;
function normalizeEnvKey(value) {
  return value.trim().toUpperCase();
}
function parseEnvContent(content) {
  const entries = content.replace(/^\uFEFF/, "").split(/\r?\n/).flatMap((line, index) => parseLine(line, index + 1));
  const counts = /* @__PURE__ */ new Map();
  for (const entry of entries) {
    if (entry.status === "valid") {
      counts.set(entry.key, (counts.get(entry.key) ?? 0) + 1);
    }
  }
  return entries.map((entry) => {
    if (entry.status === "valid" && (counts.get(entry.key) ?? 0) > 1) {
      return { ...entry, status: "duplicate", error: "Duplicate key in import" };
    }
    return entry;
  });
}
function parseLine(line, lineNumber) {
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith("#")) {
    return [];
  }
  const source = trimmed.startsWith("export ") ? trimmed.slice("export ".length).trimStart() : trimmed;
  const equalsIndex = source.indexOf("=");
  if (equalsIndex === -1) {
    return [invalidEntry(lineNumber, line, source, "Missing = separator")];
  }
  const rawKey = source.slice(0, equalsIndex).trim();
  const key = normalizeEnvKey(rawKey);
  if (!KEY_PATTERN.test(key)) {
    return [invalidEntry(lineNumber, line, rawKey, "Invalid key")];
  }
  const rawValue = source.slice(equalsIndex + 1);
  const parsed = parseValue(rawValue);
  if (parsed.error) {
    return [invalidEntry(lineNumber, line, rawKey, parsed.error)];
  }
  return [{
    line: lineNumber,
    raw: line,
    rawKey,
    key,
    value: parsed.value,
    status: "valid",
    error: ""
  }];
}
function parseValue(rawValue) {
  const value = rawValue.trimStart();
  if (!value) {
    return { value: "", error: "" };
  }
  if (value.startsWith('"')) {
    return parseQuotedValue(value, '"');
  }
  if (value.startsWith("'")) {
    return parseQuotedValue(value, "'");
  }
  return { value: stripInlineComment(value).trim(), error: "" };
}
function parseQuotedValue(value, quote) {
  let out = "";
  let escaped = false;
  for (let i = 1; i < value.length; i += 1) {
    const char = value[i];
    if (quote === '"' && escaped) {
      out += unescapeDoubleQuotedChar(char);
      escaped = false;
      continue;
    }
    if (quote === '"' && char === "\\") {
      escaped = true;
      continue;
    }
    if (char === quote) {
      const rest = value.slice(i + 1).trim();
      if (rest && !rest.startsWith("#")) {
        return { value: "", error: "Unexpected text after quoted value" };
      }
      return { value: out, error: "" };
    }
    out += char;
  }
  return { value: "", error: "Unterminated quoted value" };
}
function unescapeDoubleQuotedChar(char) {
  if (char === "n") return "\n";
  if (char === "r") return "\r";
  if (char === "t") return "	";
  return char;
}
function stripInlineComment(value) {
  for (let i = 0; i < value.length; i += 1) {
    if (value[i] === "#" && i > 0 && /\s/.test(value[i - 1])) {
      return value.slice(0, i);
    }
  }
  return value;
}
function invalidEntry(line, raw, rawKey, error) {
  return {
    line,
    raw,
    rawKey,
    key: normalizeEnvKey(rawKey),
    value: "",
    status: "invalid",
    error
  };
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let dirtyCount, hasDirty, existingKeys, importRows, importReadyRows;
    let vars = [];
    let loading = true;
    let savingChanges = false;
    let importText = "";
    let confirmOverwrite = false;
    function buildImportRows(content, keys) {
      if (!content.trim()) return [];
      return parseEnvContent(content).map((entry) => ({
        ...entry,
        importStatus: entry.status === "invalid" ? "invalid" : entry.status === "duplicate" ? "duplicate" : keys.has(entry.key) ? "overwrite" : "new"
      }));
    }
    function countImportRows(rows) {
      return rows.reduce(
        (counts, row) => {
          counts.total += 1;
          if (row.importStatus === "new") counts.newCount += 1;
          if (row.importStatus === "overwrite") counts.overwrite += 1;
          if (row.importStatus === "duplicate") counts.duplicate += 1;
          if (row.importStatus === "invalid") counts.invalid += 1;
          return counts;
        },
        {
          total: 0,
          newCount: 0,
          overwrite: 0,
          duplicate: 0,
          invalid: 0
        }
      );
    }
    dirtyCount = vars.filter((v) => v.dirty).length;
    hasDirty = dirtyCount > 0;
    existingKeys = new Set(vars.map((v) => v.key));
    importRows = buildImportRows(importText, existingKeys);
    countImportRows(importRows);
    importReadyRows = importRows.filter((row) => row.importStatus === "new" || row.importStatus === "overwrite" && confirmOverwrite);
    importReadyRows.length > 0 && true && !hasDirty;
    head("1fdiw6z", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>Environment · MyPaas</title>`);
      });
    });
    SectionPanel($$renderer2, {
      title: "Environment variables",
      description: "Encrypted at rest. Reveal only when you need to inspect a stored value.",
      contentClass: "p-0",
      children: ($$renderer3) => {
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]--> `);
        {
          $$renderer3.push("<!--[0-->");
          $$renderer3.push(`<div class="space-y-3 p-5"><!--[-->`);
          const each_array_1 = ensure_array_like([1, 2, 3]);
          for (let $$index_1 = 0, $$length = each_array_1.length; $$index_1 < $$length; $$index_1++) {
            each_array_1[$$index_1];
            $$renderer3.push(`<div class="h-11 animate-pulse rounded-md bg-gray-100 dark:bg-gray-800"></div>`);
          }
          $$renderer3.push(`<!--]--></div>`);
        }
        $$renderer3.push(`<!--]--> `);
        {
          $$renderer3.push("<!--[-1-->");
        }
        $$renderer3.push(`<!--]-->`);
      },
      $$slots: {
        default: true,
        actions: ($$renderer3) => {
          {
            $$renderer3.push(`<div class="flex flex-wrap items-center gap-2">`);
            if (hasDirty) {
              $$renderer3.push("<!--[0-->");
              $$renderer3.push(`<span class="rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">${escape_html(dirtyCount)} unsaved</span> `);
              ActionButton($$renderer3, {
                variant: "primary",
                loading: savingChanges,
                loadingLabel: "Saving...",
                children: ($$renderer4) => {
                  $$renderer4.push(`<!---->Save`);
                },
                $$slots: { default: true }
              });
              $$renderer3.push(`<!---->`);
            } else {
              $$renderer3.push("<!--[-1-->");
            }
            $$renderer3.push(`<!--]--> `);
            {
              $$renderer3.push("<!--[0-->");
              ActionButton($$renderer3, {
                variant: "secondary",
                disabled: loading,
                children: ($$renderer4) => {
                  $$renderer4.push(`<!---->Import .env`);
                },
                $$slots: { default: true }
              });
            }
            $$renderer3.push(`<!--]--> `);
            {
              $$renderer3.push("<!--[0-->");
              IconButton($$renderer3, {
                label: "Add variable",
                variant: "primary",
                children: ($$renderer4) => {
                  Plus($$renderer4, { class: "h-4 w-4", "aria-hidden": "true" });
                },
                $$slots: { default: true }
              });
            }
            $$renderer3.push(`<!--]--></div>`);
          }
        }
      }
    });
  });
}

export { _page as default };
//# sourceMappingURL=_page.svelte-DtNSU5Dh.js.map
