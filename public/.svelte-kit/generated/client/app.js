export { matchers } from './matchers.js';

export const nodes = [
	() => import('./nodes/0'),
	() => import('./nodes/1'),
	() => import('./nodes/2'),
	() => import('./nodes/3'),
	() => import('./nodes/4'),
	() => import('./nodes/5'),
	() => import('./nodes/6'),
	() => import('./nodes/7'),
	() => import('./nodes/8'),
	() => import('./nodes/9'),
	() => import('./nodes/10'),
	() => import('./nodes/11'),
	() => import('./nodes/12'),
	() => import('./nodes/13'),
	() => import('./nodes/14'),
	() => import('./nodes/15'),
	() => import('./nodes/16')
];

export const server_loads = [];

export const dictionary = {
		"/": [3],
		"/admin/audit-logs": [4],
		"/admin/users": [5],
		"/docs": [6],
		"/login": [7],
		"/projects": [8],
		"/projects/new": [9],
		"/projects/[id]": [10,[2]],
		"/projects/[id]/database": [11,[2]],
		"/projects/[id]/deployments": [12,[2]],
		"/projects/[id]/env": [13,[2]],
		"/projects/[id]/logs": [14,[2]],
		"/projects/[id]/metrics": [15,[2]],
		"/projects/[id]/settings": [16,[2]]
	};

export const hooks = {
	handleError: (({ error }) => { console.error(error) }),
	
	reroute: (() => {}),
	transport: {}
};

export const decoders = Object.fromEntries(Object.entries(hooks.transport).map(([k, v]) => [k, v.decode]));
export const encoders = Object.fromEntries(Object.entries(hooks.transport).map(([k, v]) => [k, v.encode]));

export const hash = false;

export const decode = (type, value) => decoders[type](value);

export { default as root } from '../root.js';