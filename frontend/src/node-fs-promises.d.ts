// The production dashboard runs on @sveltejs/adapter-node, but the frontend
// intentionally does not depend on the full Node type surface. Keep the one
// host-file API used by the owner-only update status route explicit and narrow.
declare module 'node:fs/promises' {
	export function readFile(path: string, encoding: 'utf8'): Promise<string>;
}
