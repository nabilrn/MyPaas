import type { DBStudioForeignKey, DBStudioTableDetails } from '$types';

export interface ERDNode {
	id: string;
	detail: DBStudioTableDetails;
	x: number;
	y: number;
	width: number;
	height: number;
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
}

const nodeWidth = 250;
const nodeHeight = 168;
const horizontalGap = 72;
const verticalGap = 72;
const padding = 28;

export function tableID(schema: string, table: string) {
	return `${schema}.${table}`;
}

export function layoutERD(details: DBStudioTableDetails[], columns = 3): ERDGraph {
	const columnCount = Math.max(1, columns);
	const nodes = details.map((detail, index) => ({
		id: tableID(detail.schema, detail.name),
		detail,
		x: padding + (index % columnCount) * (nodeWidth + horizontalGap),
		y: padding + Math.floor(index / columnCount) * (nodeHeight + verticalGap),
		width: nodeWidth,
		height: nodeHeight
	}));
	const byID = new Map(nodes.map((node) => [node.id, node]));
	const edges: ERDEdge[] = [];

	for (const from of nodes) {
		from.detail.foreignKeys.forEach((foreignKey, index) => {
			const to = byID.get(tableID(foreignKey.referencedSchema, foreignKey.referencedTable));
			if (!to) return;
			edges.push({
				id: `${from.id}:${foreignKey.name}:${foreignKey.column}:${index}`,
				from,
				to,
				foreignKey
			});
		});
	}

	const rows = Math.ceil(Math.max(1, nodes.length) / columnCount);
	const usedColumns = Math.min(columnCount, Math.max(1, nodes.length));
	return {
		nodes,
		edges,
		width: padding * 2 + usedColumns * nodeWidth + Math.max(0, usedColumns - 1) * horizontalGap,
		height: padding * 2 + rows * nodeHeight + Math.max(0, rows - 1) * verticalGap
	};
}
