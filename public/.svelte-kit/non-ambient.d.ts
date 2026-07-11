
// this file is generated — do not edit it


declare module "svelte/elements" {
	export interface HTMLAttributes<T> {
		'data-sveltekit-keepfocus'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-noscroll'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-preload-code'?:
			| true
			| ''
			| 'eager'
			| 'viewport'
			| 'hover'
			| 'tap'
			| 'off'
			| undefined
			| null;
		'data-sveltekit-preload-data'?: true | '' | 'hover' | 'tap' | 'off' | undefined | null;
		'data-sveltekit-reload'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-replacestate'?: true | '' | 'off' | undefined | null;
	}
}

export {};


declare module "$app/types" {
	type MatcherParam<M> = M extends (param : string) => param is (infer U extends string) ? U : string;

	export interface AppTypes {
		RouteId(): "/" | "/admin" | "/admin/audit-logs" | "/admin/users" | "/docs" | "/login" | "/projects" | "/projects/new" | "/projects/[id]" | "/projects/[id]/database" | "/projects/[id]/deployments" | "/projects/[id]/env" | "/projects/[id]/logs" | "/projects/[id]/metrics" | "/projects/[id]/settings";
		RouteParams(): {
			"/projects/[id]": { id: string };
			"/projects/[id]/database": { id: string };
			"/projects/[id]/deployments": { id: string };
			"/projects/[id]/env": { id: string };
			"/projects/[id]/logs": { id: string };
			"/projects/[id]/metrics": { id: string };
			"/projects/[id]/settings": { id: string }
		};
		LayoutParams(): {
			"/": { id?: string };
			"/admin": Record<string, never>;
			"/admin/audit-logs": Record<string, never>;
			"/admin/users": Record<string, never>;
			"/docs": Record<string, never>;
			"/login": Record<string, never>;
			"/projects": { id?: string };
			"/projects/new": Record<string, never>;
			"/projects/[id]": { id: string };
			"/projects/[id]/database": { id: string };
			"/projects/[id]/deployments": { id: string };
			"/projects/[id]/env": { id: string };
			"/projects/[id]/logs": { id: string };
			"/projects/[id]/metrics": { id: string };
			"/projects/[id]/settings": { id: string }
		};
		Pathname(): "/" | "/admin/audit-logs" | "/admin/users" | "/docs" | "/login" | "/projects" | "/projects/new" | `/projects/${string}` & {} | `/projects/${string}/database` & {} | `/projects/${string}/deployments` & {} | `/projects/${string}/env` & {} | `/projects/${string}/logs` & {} | `/projects/${string}/metrics` & {} | `/projects/${string}/settings` & {};
		ResolvedPathname(): `${"" | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): "/firebase-messaging-sw.js" | string & {};
	}
}