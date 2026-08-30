import type { DBStudioForeignKey, DBStudioTableDetails } from '$types';

export type ERDDirection = 'LR' | 'TB';
export type ERDDensity = 'compact' | 'comfortable';

export interface ERDLayoutOptions {
	direction?: ERDDirection;
	density?: ERDDensity;
	targetRatio?: number;
}

export interface ERDNode {
	id: string;
	detail: DBStudioTableDetails;
	displayColumns: DBStudioTableDetails['columns'];
	hiddenColumnCount: number;
	x: number;
	y: number;
	width: number;
	height: number;
	group: number;
	level: number;
	order: number;
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

// Mirrors the layout concepts used by draw.io's hierarchical layout without
// pulling the full diagram editor into the dashboard bundle.
const pagePadding = 40;
const intraCellSpacing = 44;
const interRankCellSpacing = 128;
const interHierarchySpacing = 104;

export function tableID(schema: string, table: string) {
	return `${schema}.${table}`;
}

function buildDisplayColumns(
	detail: DBStudioTableDetails,
	incomingReferences: Set<string>,
	density: ERDDensity
) {
	const baseCount = density === 'compact' ? 6 : 8;
	const required = new Set<string>(incomingReferences);
	for (const column of detail.columns) {
		if (column.primaryKey) required.add(column.name);
	}
	for (const foreignKey of detail.foreignKeys) required.add(foreignKey.column);

	const visible = new Set(detail.columns.slice(0, baseCount).map((column) => column.name));
	for (const name of required) visible.add(name);

	return detail.columns.filter((column) => visible.has(column.name));
}

function dimensions(displayRows: number, hiddenRows: number, density: ERDDensity) {
	const width = density === 'compact' ? 224 : 248;
	const rowHeight = density === 'compact' ? 18 : 20;
	const headerHeight = density === 'compact' ? 52 : 56;
	const footerHeight = hiddenRows > 0 ? 24 : 10;
	const height = headerHeight + Math.max(displayRows, 1) * rowHeight + footerHeight;
	return { width, height: Math.max(density === 'compact' ? 126 : 142, height) };
}

function connectedGroups(ids: string[], edges: Array<{ child: string; parent: string }>) {
	const adjacency = new Map(ids.map((id) => [id, new Set<string>()]));
	for (const edge of edges) {
		adjacency.get(edge.child)?.add(edge.parent);
		adjacency.get(edge.parent)?.add(edge.child);
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

function assignLevels(members: string[], edges: Array<{ child: string; parent: string }>) {
	const memberSet = new Set(members);
	const children = new Map(members.map((id) => [id, new Set<string>()]));
	const parentCount = new Map(members.map((id) => [id, 0]));
	for (const edge of edges) {
		if (!memberSet.has(edge.child) || !memberSet.has(edge.parent)) continue;
		children.get(edge.parent)?.add(edge.child);
		parentCount.set(edge.child, (parentCount.get(edge.child) ?? 0) + 1);
	}

	const levels = new Map<string, number>();
	const roots = members.filter((id) => (parentCount.get(id) ?? 0) === 0).sort();
	const seeds = roots.length > 0 ? roots : [...members].sort().slice(0, 1);
	const queue = seeds.map((id) => ({ id, level: 0 }));

	while (queue.length > 0) {
		const current = queue.shift();
		if (!current) continue;
		const previous = levels.get(current.id);
		if (previous !== undefined && previous <= current.level) continue;
		levels.set(current.id, current.level);
		for (const child of children.get(current.id) ?? []) {
			if (!levels.has(child)) queue.push({ id: child, level: current.level + 1 });
		}
	}

	for (const id of [...members].sort()) {
		if (!levels.has(id)) levels.set(id, 0);
	}
	return levels;
}

function orderedBuckets(
	members: string[],
	levels: Map<string, number>,
	edges: Array<{ child: string; parent: string }>
) {
	const buckets = new Map<number, string[]>();
	for (const id of members) {
		const level = levels.get(id) ?? 0;
		buckets.set(level, [...(buckets.get(level) ?? []), id]);
	}
	for (const [level, ids] of buckets) buckets.set(level, [...ids].sort());

	const adjacency = new Map(members.map((id) => [id, new Set<string>()]));
	for (const edge of edges) {
		if (!adjacency.has(edge.child) || !adjacency.has(edge.parent)) continue;
		adjacency.get(edge.child)?.add(edge.parent);
		adjacency.get(edge.parent)?.add(edge.child);
	}

	const levelsSorted = [...buckets.keys()].sort((a, b) => a - b);
	const reorder = (level: number, neighborLevel: number) => {
		const current = buckets.get(level);
		const neighbors = buckets.get(neighborLevel);
		if (!current || !neighbors || current.length < 2) return;
		const positions = new Map(neighbors.map((id, index) => [id, index]));
		const previousOrder = new Map(current.map((id, index) => [id, index]));
		current.sort((a, b) => {
			const score = (id: string) => {
				const values = [...(adjacency.get(id) ?? [])]
					.map((neighbor) => positions.get(neighbor))
					.filter((value): value is number => value !== undefined);
				return values.length > 0 ? values.reduce((sum, value) => sum + value, 0) / values.length : Number.POSITIVE_INFINITY;
			};
			const diff = score(a) - score(b);
			if (Number.isFinite(diff) && Math.abs(diff) > 0.001) return diff;
			return (previousOrder.get(a) ?? 0) - (previousOrder.get(b) ?? 0);
		});
	};

	// Draw.io's hierarchical layout performs crossing-reduction sweeps between
	// ranks. A few deterministic barycentric passes give ERDs the same useful
	// property without embedding the editor runtime.
	for (let pass = 0; pass < 4; pass += 1) {
		for (let index = 1; index < levelsSorted.length; index += 1) {
			reorder(levelsSorted[index], levelsSorted[index - 1]);
		}
		for (let index = levelsSorted.length - 2; index >= 0; index -= 1) {
			reorder(levelsSorted[index], levelsSorted[index + 1]);
		}
	}
	return buckets;
}

interface ComponentLayout {
	group: number;
	nodes: ERDNode[];
	width: number;
	height: number;
}

export function layoutERD(details: DBStudioTableDetails[], options: ERDLayoutOptions = {}): ERDGraph {
	const direction = options.direction ?? 'TB';
	const density = options.density ?? 'comfortable';
	const targetRatio = Math.max(0.55, options.targetRatio ?? (direction === 'TB' ? 16 / 9 : 3 / 4));
	const ids = details.map((detail) => tableID(detail.schema, detail.name));
	const detailsByID = new Map(details.map((detail) => [tableID(detail.schema, detail.name), detail]));

	const edgeSpecs: Array<{ id: string; child: string; parent: string; foreignKey: DBStudioForeignKey }> = [];
	const incomingColumns = new Map(ids.map((id) => [id, new Set<string>()]));
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
			incomingColumns.get(parent)?.add(foreignKey.referencedColumn);
		});
	}

	const { groupByID, count: groups } = connectedGroups(ids, edgeSpecs);
	const components: ComponentLayout[] = [];

	for (let group = 0; group < groups; group += 1) {
		const members = ids.filter((id) => (groupByID.get(id) ?? 0) === group);
		const groupEdges = edgeSpecs.filter((edge) => members.includes(edge.child) && members.includes(edge.parent));
		const levels = assignLevels(members, groupEdges);
		const buckets = orderedBuckets(members, levels, groupEdges);
		const levelNumbers = [...buckets.keys()].sort((a, b) => a - b);
		const nodeSpecs = new Map<string, { width: number; height: number; displayColumns: DBStudioTableDetails['columns']; hiddenColumnCount: number }>();

		for (const id of members) {
			const detail = detailsByID.get(id);
			if (!detail) continue;
			const displayColumns = buildDisplayColumns(detail, incomingColumns.get(id) ?? new Set<string>(), density);
			const hiddenColumnCount = Math.max(0, detail.columns.length - displayColumns.length);
			nodeSpecs.set(id, { ...dimensions(displayColumns.length, hiddenColumnCount, density), displayColumns, hiddenColumnCount });
		}

		const rankPrimarySizes = new Map<number, number>();
		const rankCrossSizes = new Map<number, number>();
		for (const level of levelNumbers) {
			const bucket = buckets.get(level) ?? [];
			const primary = Math.max(0, ...bucket.map((id) => {
				const spec = nodeSpecs.get(id);
				return direction === 'LR' ? spec?.width ?? 0 : spec?.height ?? 0;
			}));
			const cross = bucket.reduce((sum, id, index) => {
				const spec = nodeSpecs.get(id);
				const size = direction === 'LR' ? spec?.height ?? 0 : spec?.width ?? 0;
				return sum + size + (index > 0 ? intraCellSpacing : 0);
			}, 0);
			rankPrimarySizes.set(level, primary);
			rankCrossSizes.set(level, cross);
		}

		const componentCross = Math.max(0, ...rankCrossSizes.values());
		const primaryOffsets = new Map<number, number>();
		let primaryCursor = 0;
		for (const level of levelNumbers) {
			primaryOffsets.set(level, primaryCursor);
			primaryCursor += (rankPrimarySizes.get(level) ?? 0) + interRankCellSpacing;
		}
		const componentPrimary = Math.max(0, primaryCursor - (levelNumbers.length > 0 ? interRankCellSpacing : 0));
		const nodes: ERDNode[] = [];

		for (const level of levelNumbers) {
			const bucket = buckets.get(level) ?? [];
			let crossCursor = Math.max(0, (componentCross - (rankCrossSizes.get(level) ?? 0)) / 2);
			bucket.forEach((id, order) => {
				const detail = detailsByID.get(id);
				const spec = nodeSpecs.get(id);
				if (!detail || !spec) return;
				const primary = primaryOffsets.get(level) ?? 0;
				const x = direction === 'LR' ? primary : crossCursor;
				const y = direction === 'LR' ? crossCursor : primary;
				nodes.push({
					id,
					detail,
					displayColumns: spec.displayColumns,
					hiddenColumnCount: spec.hiddenColumnCount,
					x,
					y,
					width: spec.width,
					height: spec.height,
					group,
					level,
					order
				});
				crossCursor += (direction === 'LR' ? spec.height : spec.width) + intraCellSpacing;
			});
		}

		components.push({
			group,
			nodes,
			width: direction === 'LR' ? componentPrimary : componentCross,
			height: direction === 'LR' ? componentCross : componentPrimary
		});
	}

	const totalArea = components.reduce((sum, component) => sum + (component.width + interHierarchySpacing) * (component.height + interHierarchySpacing), 0);
	const targetRowWidth = Math.max(720, Math.sqrt(Math.max(totalArea, 1) * targetRatio));
	let cursorX = pagePadding;
	let cursorY = pagePadding;
	let rowHeight = 0;
	let maxX = pagePadding;
	let maxY = pagePadding;
	const nodes: ERDNode[] = [];

	for (const component of components) {
		if (cursorX > pagePadding && cursorX + component.width > targetRowWidth + pagePadding) {
			cursorX = pagePadding;
			cursorY += rowHeight + interHierarchySpacing;
			rowHeight = 0;
		}
		for (const node of component.nodes) {
			node.x += cursorX;
			node.y += cursorY;
			nodes.push(node);
		}
		maxX = Math.max(maxX, cursorX + component.width);
		maxY = Math.max(maxY, cursorY + component.height);
		rowHeight = Math.max(rowHeight, component.height);
		cursorX += component.width + interHierarchySpacing;
	}

	const byID = new Map(nodes.map((node) => [node.id, node]));
	const edges: ERDEdge[] = edgeSpecs.flatMap((spec) => {
		const from = byID.get(spec.child);
		const to = byID.get(spec.parent);
		return from && to ? [{ id: spec.id, from, to, foreignKey: spec.foreignKey }] : [];
	});

	return {
		nodes,
		edges,
		width: Math.max(720, maxX + pagePadding),
		height: Math.max(420, maxY + pagePadding),
		groups
	};
}
