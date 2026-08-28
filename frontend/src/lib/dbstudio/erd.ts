import type { DBStudioForeignKey, DBStudioTableDetails } from '$types';

export type ERDDirection = 'LR' | 'TB';
export type ERDDensity = 'compact' | 'comfortable';

export interface ERDLayoutOptions {
	direction?: ERDDirection;
	density?: ERDDensity;
}

export interface ERDNode {
	id: string;
	detail: DBStudioTableDetails;
	x: number;
	y: number;
	width: number;
	height: number;
	group: number;
	level: number;
}

export interface ERDEdge {
	id: string;
	from: ERDNode;
	to: ERDNode;
	foreignKey: DBStudioForeignKey;
}

export interface ERDGraph {
	nodes: ERDNode[];
	edges: ERDEdge[];
	width: number;
	height: number;
	groups: number;
}

const padding = 28;
const componentGap = 96;

export function tableID(schema: string, table: string) {
	return `${schema}.${table}`;
}

function dimensions(detail: DBStudioTableDetails, density: ERDDensity) {
	const width = density === 'compact' ? 228 : 260;
	const visibleRows = Math.min(detail.columns.length, density === 'compact' ? 5 : 7);
	const rowHeight = density === 'compact' ? 18 : 20;
	const height = 66 + visibleRows * rowHeight + (detail.columns.length > visibleRows ? 18 : 0);
	return { width, height: Math.max(density === 'compact' ? 132 : 148, height) };
}

function connectedGroups(ids: string[], edges: Array<{ from: string; to: string }>) {
	const adjacency = new Map(ids.map((id) => [id, new Set<string>()]));
	for (const edge of edges) {
		adjacency.get(edge.from)?.add(edge.to);
		adjacency.get(edge.to)?.add(edge.from);
	}

	const groupByID = new Map<string, number>();
	let group = 0;
	for (const id of [...ids].sort()) {
		if (groupByID.has(id)) continue;
		const queue = [id];
		groupByID.set(id, group);
		while (queue.length > 0) {
			const current = queue.shift();
			if (!current) continue;
			for (const next of adjacency.get(current) ?? []) {
				if (groupByID.has(next)) continue;
				groupByID.set(next, group);
				queue.push(next);
			}
		}
		group += 1;
	}
	return { groupByID, count: group };
}

function assignLevels(ids: string[], edges: Array<{ child: string; parent: string }>, groupByID: Map<string, number>) {
	const children = new Map(ids.map((id) => [id, new Set<string>()]));
	const dependencyCount = new Map(ids.map((id) => [id, 0]));
	for (const edge of edges) {
		children.get(edge.parent)?.add(edge.child);
		dependencyCount.set(edge.child, (dependencyCount.get(edge.child) ?? 0) + 1);
	}

	const levels = new Map<string, number>();
	const groups = new Map<number, string[]>();
	for (const id of ids) {
		const group = groupByID.get(id) ?? 0;
		groups.set(group, [...(groups.get(group) ?? []), id]);
	}

	for (const members of groups.values()) {
		const ordered = [...members].sort();
		const roots = ordered.filter((id) => (dependencyCount.get(id) ?? 0) === 0);
		const queue = (roots.length > 0 ? roots : ordered.slice(0, 1)).map((id) => ({ id, level: 0 }));
		const visited = new Set<string>();

		while (queue.length > 0) {
			const current = queue.shift();
			if (!current || visited.has(current.id)) continue;
			visited.add(current.id);
			levels.set(current.id, current.level);
			for (const child of children.get(current.id) ?? []) {
				if (!visited.has(child)) queue.push({ id: child, level: current.level + 1 });
			}
		}

		for (const id of ordered) {
			if (!levels.has(id)) levels.set(id, 0);
		}
	}
	return levels;
}

export function layoutERD(details: DBStudioTableDetails[], options: ERDLayoutOptions = {}): ERDGraph {
	const direction = options.direction ?? 'LR';
	const density = options.density ?? 'comfortable';
	const horizontalGap = density === 'compact' ? 64 : 92;
	const verticalGap = density === 'compact' ? 42 : 64;
	const ids = details.map((detail) => tableID(detail.schema, detail.name));
	const detailsByID = new Map(details.map((detail) => [tableID(detail.schema, detail.name), detail]));

	const edgeSpecs: Array<{ id: string; child: string; parent: string; foreignKey: DBStudioForeignKey }> = [];
	for (const detail of details) {
		const child = tableID(detail.schema, detail.name);
		detail.foreignKeys.forEach((foreignKey, index) => {
			const parent = tableID(foreignKey.referencedSchema, foreignKey.referencedTable);
			if (!detailsByID.has(parent)) return;
			edgeSpecs.push({
				id: `${child}:${foreignKey.name}:${foreignKey.column}:${index}`,
				child,
				parent,
				foreignKey
			});
		});
	}

	const { groupByID, count: groups } = connectedGroups(ids, edgeSpecs.map((edge) => ({ from: edge.child, to: edge.parent })));
	const levels = assignLevels(ids, edgeSpecs, groupByID);
	const nodes: ERDNode[] = [];
	let groupOffset = padding;
	let maxX = padding;
	let maxY = padding;

	for (let group = 0; group < groups; group += 1) {
		const members = ids
			.filter((id) => (groupByID.get(id) ?? 0) === group)
			.sort((a, b) => (levels.get(a) ?? 0) - (levels.get(b) ?? 0) || a.localeCompare(b));
		const levelBuckets = new Map<number, string[]>();
		for (const id of members) {
			const level = levels.get(id) ?? 0;
			levelBuckets.set(level, [...(levelBuckets.get(level) ?? []), id]);
		}

		let groupExtent = 0;
		for (const [level, levelIDs] of [...levelBuckets.entries()].sort((a, b) => a[0] - b[0])) {
			let crossOffset = groupOffset;
			let levelExtent = 0;
			for (const id of levelIDs) {
				const detail = detailsByID.get(id);
				if (!detail) continue;
				const { width, height } = dimensions(detail, density);
				const x = direction === 'LR' ? padding + level * (width + horizontalGap) : crossOffset;
				const y = direction === 'LR' ? crossOffset : padding + level * (height + verticalGap);
				nodes.push({ id, detail, x, y, width, height, group, level });
				crossOffset += (direction === 'LR' ? height : width) + verticalGap;
				levelExtent = Math.max(levelExtent, crossOffset - groupOffset - verticalGap);
				maxX = Math.max(maxX, x + width);
				maxY = Math.max(maxY, y + height);
			}
			groupExtent = Math.max(groupExtent, levelExtent);
		}
		groupOffset += groupExtent + componentGap;
	}

	const byID = new Map(nodes.map((node) => [node.id, node]));
	const edges: ERDEdge[] = [];
	for (const spec of edgeSpecs) {
		const from = byID.get(spec.child);
		const to = byID.get(spec.parent);
		if (!from || !to) continue;
		edges.push({ id: spec.id, from, to, foreignKey: spec.foreignKey });
	}

	return {
		nodes,
		edges,
		width: Math.max(720, maxX + padding),
		height: Math.max(360, maxY + padding),
		groups
	};
}
