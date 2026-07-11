const manifest = (() => {
function __memo(fn) {
	let value;
	return () => value ??= (value = fn());
}

return {
	appDir: "_app",
	appPath: "_app",
	assets: new Set(["firebase-messaging-sw.js"]),
	mimeTypes: {".js":"text/javascript"},
	_: {
		client: {start:"_app/immutable/entry/start.VRkp71_5.js",app:"_app/immutable/entry/app.C8ippmtc.js",imports:["_app/immutable/entry/start.VRkp71_5.js","_app/immutable/chunks/DOS1i2qW.js","_app/immutable/chunks/DLHw1tiy.js","_app/immutable/chunks/BHzOHchT.js","_app/immutable/chunks/CWeFt6jb.js","_app/immutable/entry/app.C8ippmtc.js","_app/immutable/chunks/BHzOHchT.js","_app/immutable/chunks/Bzak7iHL.js","_app/immutable/chunks/DLHw1tiy.js","_app/immutable/chunks/CDMxTIWt.js","_app/immutable/chunks/Bveajtgm.js","_app/immutable/chunks/aH1CfaRe.js"],stylesheets:[],fonts:[],uses_env_dynamic_public:false},
		nodes: [
			__memo(() => import('./chunks/0-BAd7Lq8W.js')),
			__memo(() => import('./chunks/1-D13VBbzH.js')),
			__memo(() => import('./chunks/2-D3yW7wGW.js')),
			__memo(() => import('./chunks/3-Dvh8ydFS.js')),
			__memo(() => import('./chunks/4-C03x-DEw.js')),
			__memo(() => import('./chunks/5-DZYk5EFm.js')),
			__memo(() => import('./chunks/6-B4mgtrMI.js')),
			__memo(() => import('./chunks/7-DhD9Ua8k.js')),
			__memo(() => import('./chunks/8-Dd1lj2c1.js')),
			__memo(() => import('./chunks/9-askMthNX.js')),
			__memo(() => import('./chunks/10-C7PLnOF9.js')),
			__memo(() => import('./chunks/11-C3lHIbLk.js')),
			__memo(() => import('./chunks/12-C12OKgC5.js')),
			__memo(() => import('./chunks/13-BjRAPXfH.js')),
			__memo(() => import('./chunks/14-BYa57I4C.js')),
			__memo(() => import('./chunks/15-ECNNOugJ.js'))
		],
		remotes: {
			
		},
		routes: [
			{
				id: "/",
				pattern: /^\/$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 3 },
				endpoint: null
			},
			{
				id: "/admin/audit-logs",
				pattern: /^\/admin\/audit-logs\/?$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 4 },
				endpoint: null
			},
			{
				id: "/admin/users",
				pattern: /^\/admin\/users\/?$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 5 },
				endpoint: null
			},
			{
				id: "/login",
				pattern: /^\/login\/?$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 6 },
				endpoint: null
			},
			{
				id: "/projects",
				pattern: /^\/projects\/?$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 7 },
				endpoint: null
			},
			{
				id: "/projects/new",
				pattern: /^\/projects\/new\/?$/,
				params: [],
				page: { layouts: [0,], errors: [1,], leaf: 8 },
				endpoint: null
			},
			{
				id: "/projects/[id]",
				pattern: /^\/projects\/([^/]+?)\/?$/,
				params: [{"name":"id","optional":false,"rest":false,"chained":false}],
				page: { layouts: [0,2,], errors: [1,,], leaf: 9 },
				endpoint: null
			},
			{
				id: "/projects/[id]/database",
				pattern: /^\/projects\/([^/]+?)\/database\/?$/,
				params: [{"name":"id","optional":false,"rest":false,"chained":false}],
				page: { layouts: [0,2,], errors: [1,,], leaf: 10 },
				endpoint: null
			},
			{
				id: "/projects/[id]/deployments",
				pattern: /^\/projects\/([^/]+?)\/deployments\/?$/,
				params: [{"name":"id","optional":false,"rest":false,"chained":false}],
				page: { layouts: [0,2,], errors: [1,,], leaf: 11 },
				endpoint: null
			},
			{
				id: "/projects/[id]/env",
				pattern: /^\/projects\/([^/]+?)\/env\/?$/,
				params: [{"name":"id","optional":false,"rest":false,"chained":false}],
				page: { layouts: [0,2,], errors: [1,,], leaf: 12 },
				endpoint: null
			},
			{
				id: "/projects/[id]/logs",
				pattern: /^\/projects\/([^/]+?)\/logs\/?$/,
				params: [{"name":"id","optional":false,"rest":false,"chained":false}],
				page: { layouts: [0,2,], errors: [1,,], leaf: 13 },
				endpoint: null
			},
			{
				id: "/projects/[id]/metrics",
				pattern: /^\/projects\/([^/]+?)\/metrics\/?$/,
				params: [{"name":"id","optional":false,"rest":false,"chained":false}],
				page: { layouts: [0,2,], errors: [1,,], leaf: 14 },
				endpoint: null
			},
			{
				id: "/projects/[id]/settings",
				pattern: /^\/projects\/([^/]+?)\/settings\/?$/,
				params: [{"name":"id","optional":false,"rest":false,"chained":false}],
				page: { layouts: [0,2,], errors: [1,,], leaf: 15 },
				endpoint: null
			}
		],
		prerendered_routes: new Set([]),
		matchers: async () => {
			
			return {  };
		},
		server_assets: {}
	}
}
})();

const prerendered = new Set([]);

const base = "";

export { base, manifest, prerendered };
//# sourceMappingURL=manifest.js.map
