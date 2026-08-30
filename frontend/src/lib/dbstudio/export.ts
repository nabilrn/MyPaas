import type { ERDDensity, ERDEdge, ERDGraph, ERDNode } from './erd';
import type { DBStudioDriver, DBStudioTableDetails } from '$types';

export type ERDPagePreset = 'auto' | '1:1' | '4:3' | '3:2' | '16:9' | 'a4-landscape';

export interface ERDExportOptions {
	preset: ERDPagePreset;
	density: ERDDensity;
	showDataTypes: boolean;
	showRelationLabels: boolean;
	title?: string;
}

const encoder = new TextEncoder();
const parallelEdgeSpacing = 12;

export function presetRatio(preset: ERDPagePreset): number | null {
	switch (preset) {
		case '1:1': return 1;
		case '4:3': return 4 / 3;
		case '3:2': return 3 / 2;
		case '16:9': return 16 / 9;
		case 'a4-landscape': return 297 / 210;
		default: return null;
	}
}

function escapeXML(value: string) {
	return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function truncateERDText(value: string, maxCharacters: number) {
	if (value.length <= maxCharacters) return value;
	return `${value.slice(0, Math.max(1, maxCharacters - 1))}…`;
}

function columnY(node: ERDNode, column: string, density: ERDDensity) {
	const rawIndex = node.displayColumns.findIndex((item) => item.name === column);
	const index = rawIndex < 0 ? 0 : rawIndex;
	const headerHeight = density === 'compact' ? 52 : 56;
	const rowHeight = density === 'compact' ? 18 : 20;
	return node.y + headerHeight + index * rowHeight + rowHeight / 2;
}

function edgeLane(edge: ERDEdge, graph: ERDGraph) {
	const low = Math.min(edge.from.level, edge.to.level);
	const high = Math.max(edge.from.level, edge.to.level);
	const siblings = graph.edges
		.filter((item) => item.from.group === edge.from.group && Math.min(item.from.level, item.to.level) === low && Math.max(item.from.level, item.to.level) === high)
		.sort((a, b) => a.id.localeCompare(b.id));
	const index = Math.max(0, siblings.findIndex((item) => item.id === edge.id));
	return (index - (siblings.length - 1) / 2) * parallelEdgeSpacing;
}

export function relationPath(edge: ERDEdge, graph: ERDGraph, direction: 'LR' | 'TB', density: ERDDensity) {
	const lane = edgeLane(edge, graph);
	const fromCenterX = edge.from.x + edge.from.width / 2;
	const toCenterX = edge.to.x + edge.to.width / 2;
	const fromCenterY = edge.from.y + edge.from.height / 2;
	const toCenterY = edge.to.y + edge.to.height / 2;
	const fromRowY = columnY(edge.from, edge.foreignKey.column, density);
	const toRowY = columnY(edge.to, edge.foreignKey.referencedColumn, density);
	const sameRank = edge.from.level === edge.to.level;

	if (direction === 'LR' && !sameRank) {
		const toRight = toCenterX >= fromCenterX;
		const fromX = toRight ? edge.from.x + edge.from.width : edge.from.x;
		const toX = toRight ? edge.to.x : edge.to.x + edge.to.width;
		const middleX = (fromX + toX) / 2 + lane;
		return {
			path: `M ${fromX} ${fromRowY} H ${middleX} V ${toRowY} H ${toX}`,
			labelX: middleX,
			labelY: (fromRowY + toRowY) / 2
		};
	}

	if (direction === 'TB' && !sameRank) {
		const targetToRight = toCenterX >= fromCenterX;
		const fromX = targetToRight ? edge.from.x + edge.from.width : edge.from.x;
		const toX = targetToRight ? edge.to.x : edge.to.x + edge.to.width;
		const fromStubX = fromX + (targetToRight ? 18 : -18);
		const toStubX = toX + (targetToRight ? -18 : 18);
		const toBelow = toCenterY >= fromCenterY;
		const fromBoundary = toBelow ? edge.from.y + edge.from.height : edge.from.y;
		const toBoundary = toBelow ? edge.to.y : edge.to.y + edge.to.height;
		const middleY = (fromBoundary + toBoundary) / 2 + lane;
		return {
			path: `M ${fromX} ${fromRowY} H ${fromStubX} V ${middleY} H ${toStubX} V ${toRowY} H ${toX}`,
			labelX: (fromStubX + toStubX) / 2,
			labelY: middleY
		};
	}

	const routeRight = fromCenterX + toCenterX >= graph.width;
	const outsideX = routeRight
		? Math.max(edge.from.x + edge.from.width, edge.to.x + edge.to.width) + 48 + Math.abs(lane)
		: Math.min(edge.from.x, edge.to.x) - 48 - Math.abs(lane);
	const fromX = routeRight ? edge.from.x + edge.from.width : edge.from.x;
	const toX = routeRight ? edge.to.x + edge.to.width : edge.to.x;
	return {
		path: `M ${fromX} ${fromRowY} H ${outsideX} V ${toRowY} H ${toX}`,
		labelX: outsideX,
		labelY: (fromRowY + toRowY) / 2
	};
}

function exportDimensions(graph: ERDGraph, preset: ERDPagePreset, hasTitle: boolean) {
	const margin = 56;
	const titleHeight = hasTitle ? 36 : 0;
	const contentWidth = graph.width + margin * 2;
	const contentHeight = graph.height + margin * 2 + titleHeight;
	const ratio = presetRatio(preset);
	if (!ratio) return { width: contentWidth, height: contentHeight, margin, titleHeight };

	let width = contentWidth;
	let height = contentHeight;
	if (width / height < ratio) width = height * ratio;
	else height = width / ratio;
	return { width: Math.ceil(width), height: Math.ceil(height), margin, titleHeight };
}

export function buildERDSVG(graph: ERDGraph, direction: 'LR' | 'TB', options: ERDExportOptions) {
	const page = exportDimensions(graph, options.preset, Boolean(options.title));
	const offsetX = (page.width - graph.width) / 2;
	const freeHeight = page.height - page.titleHeight;
	const offsetY = page.titleHeight + (freeHeight - graph.height) / 2;
	const rowHeight = options.density === 'compact' ? 18 : 20;

	const edges = graph.edges.map((edge) => {
		const relation = relationPath(edge, graph, direction, options.density);
		const label = truncateERDText(`${edge.foreignKey.column} → ${edge.foreignKey.referencedColumn}`, 38);
		return `<path d="${relation.path}" fill="none" stroke="#6b7280" stroke-width="1.75" stroke-linejoin="round" marker-end="url(#arrow)"/>${options.showRelationLabels ? `<text x="${relation.labelX}" y="${relation.labelY - 6}" text-anchor="middle" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="10.5" fill="#374151" paint-order="stroke" stroke="#ffffff" stroke-width="4" stroke-linejoin="round">${escapeXML(label)}</text>` : ''}`;
	}).join('');

	const nodes = graph.nodes.map((node, nodeIndex) => {
		const clipID = `erd-node-${nodeIndex}`;
		const columns = node.displayColumns.map((column, index) => {
			const y = node.y + (options.density === 'compact' ? 65 : 69) + index * rowHeight;
			const isForeignKey = node.detail.foreignKeys.some((foreignKey) => foreignKey.column === column.name);
			const prefix = column.primaryKey ? 'PK ' : isForeignKey ? 'FK ' : '';
			const displayName = truncateERDText(`${prefix}${column.name}`, options.showDataTypes ? 22 : 36);
			const displayType = truncateERDText(column.dataType, 18);
			const type = options.showDataTypes ? `<text x="${node.x + node.width - 12}" y="${y}" text-anchor="end" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="9.5" fill="#6b7280">${escapeXML(displayType)}</text>` : '';
			return `<text x="${node.x + 12}" y="${y}" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="10.5" fill="#1f2937">${escapeXML(displayName)}</text>${type}`;
		}).join('');
		const more = node.hiddenColumnCount > 0
			? `<text x="${node.x + 12}" y="${node.y + node.height - 10}" font-family="Inter, Arial, sans-serif" font-size="10" fill="#9ca3af">+${node.hiddenColumnCount} more</text>`
			: '';
		return `<g><defs><clipPath id="${clipID}"><rect x="${node.x + 1}" y="${node.y + 1}" width="${Math.max(0, node.width - 2)}" height="${Math.max(0, node.height - 2)}" rx="6"/></clipPath></defs><rect x="${node.x}" y="${node.y}" width="${node.width}" height="${node.height}" rx="7" fill="#ffffff" stroke="#9ca3af" stroke-width="1.2"/><g clip-path="url(#${clipID})"><line x1="${node.x}" y1="${node.y + 48}" x2="${node.x + node.width}" y2="${node.y + 48}" stroke="#e5e7eb"/><text x="${node.x + 12}" y="${node.y + 16}" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="8.5" fill="#9ca3af">${escapeXML(truncateERDText(node.detail.schema, 28))}</text><text x="${node.x + 12}" y="${node.y + 36}" font-family="Inter, Arial, sans-serif" font-size="13" font-weight="600" fill="#111827">${escapeXML(truncateERDText(node.detail.name, 28))}</text>${columns}${more}</g></g>`;
	}).join('');

	const title = options.title
		? `<text x="${page.margin}" y="${page.margin - 12}" font-family="Inter, Arial, sans-serif" font-size="17" font-weight="600" fill="#111827">${escapeXML(options.title)}</text>`
		: '';

	return `<svg xmlns="http://www.w3.org/2000/svg" width="${page.width}" height="${page.height}" viewBox="0 0 ${page.width} ${page.height}"><rect width="100%" height="100%" fill="#ffffff"/>${title}<defs><marker id="arrow" markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto"><path d="M0,0 L7,3.5 L0,7 z" fill="#4b5563"/></marker></defs><g transform="translate(${offsetX} ${offsetY})">${edges}${nodes}</g></svg>`;
}

function downloadBlob(blob: Blob, filename: string) {
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement('a');
	anchor.href = url;
	anchor.download = filename;
	anchor.click();
	setTimeout(() => URL.revokeObjectURL(url), 0);
}

export function downloadSVG(svg: string, filename: string) {
	downloadBlob(new Blob([svg], { type: 'image/svg+xml;charset=utf-8' }), filename);
}

async function svgCanvas(svg: string, requestedScale = 2.25) {
	const blob = new Blob([svg], { type: 'image/svg+xml;charset=utf-8' });
	const url = URL.createObjectURL(blob);
	try {
		const image = new Image();
		image.decoding = 'async';
		await new Promise<void>((resolve, reject) => {
			image.onload = () => resolve();
			image.onerror = () => reject(new Error('Could not render ERD image'));
			image.src = url;
		});
		const naturalWidth = Math.max(1, image.naturalWidth);
		const naturalHeight = Math.max(1, image.naturalHeight);
		const maxSide = 12000;
		const maxPixels = 48_000_000;
		const scale = Math.max(0.5, Math.min(
			requestedScale,
			maxSide / naturalWidth,
			maxSide / naturalHeight,
			Math.sqrt(maxPixels / (naturalWidth * naturalHeight))
		));
		const canvas = document.createElement('canvas');
		canvas.width = Math.max(1, Math.round(naturalWidth * scale));
		canvas.height = Math.max(1, Math.round(naturalHeight * scale));
		const context = canvas.getContext('2d');
		if (!context) throw new Error('Canvas is not available');
		context.imageSmoothingEnabled = true;
		context.imageSmoothingQuality = 'high';
		context.fillStyle = '#ffffff';
		context.fillRect(0, 0, canvas.width, canvas.height);
		context.drawImage(image, 0, 0, canvas.width, canvas.height);
		return canvas;
	} finally {
		URL.revokeObjectURL(url);
	}
}

export async function downloadPNG(svg: string, filename: string) {
	const canvas = await svgCanvas(svg, 2.5);
	const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/png'));
	if (!blob) throw new Error('Could not encode PNG');
	downloadBlob(blob, filename);
}

function concatBytes(parts: Uint8Array[]) {
	const size = parts.reduce((sum, part) => sum + part.length, 0);
	const out = new Uint8Array(size);
	let offset = 0;
	for (const part of parts) {
		out.set(part, offset);
		offset += part.length;
	}
	return out;
}

function singleImagePDF(jpeg: Uint8Array, imageWidth: number, imageHeight: number, preset: ERDPagePreset) {
	const ratio = presetRatio(preset) ?? imageWidth / Math.max(imageHeight, 1);
	const pageWidth = preset === 'a4-landscape' ? 841.89 : 1000;
	const pageHeight = pageWidth / ratio;
	const content = `q ${pageWidth.toFixed(2)} 0 0 ${pageHeight.toFixed(2)} 0 0 cm /Im0 Do Q`;
	const objects: Uint8Array[] = [
		encoder.encode('<< /Type /Catalog /Pages 2 0 R >>'),
		encoder.encode('<< /Type /Pages /Kids [3 0 R] /Count 1 >>'),
		encoder.encode(`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${pageWidth.toFixed(2)} ${pageHeight.toFixed(2)}] /Resources << /XObject << /Im0 4 0 R >> >> /Contents 5 0 R >>`),
		concatBytes([encoder.encode(`<< /Type /XObject /Subtype /Image /Width ${imageWidth} /Height ${imageHeight} /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ${jpeg.length} >>\nstream\n`), jpeg, encoder.encode('\nendstream')]),
		encoder.encode(`<< /Length ${content.length} >>\nstream\n${content}\nendstream`)
	];
	const parts: Uint8Array[] = [encoder.encode('%PDF-1.4\n%âãÏÓ\n')];
	const offsets = [0];
	let cursor = parts[0].length;
	objects.forEach((object, index) => {
		offsets.push(cursor);
		const wrapped = concatBytes([encoder.encode(`${index + 1} 0 obj\n`), object, encoder.encode('\nendobj\n')]);
		parts.push(wrapped);
		cursor += wrapped.length;
	});
	const xrefOffset = cursor;
	let xref = `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
	for (let index = 1; index < offsets.length; index += 1) {
		xref += `${String(offsets[index]).padStart(10, '0')} 00000 n \n`;
	}
	xref += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xrefOffset}\n%%EOF`;
	parts.push(encoder.encode(xref));
	return concatBytes(parts);
}

export async function downloadPDF(svg: string, filename: string, preset: ERDPagePreset) {
	const canvas = await svgCanvas(svg, 2.25);
	const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.98));
	if (!blob) throw new Error('Could not encode PDF image');
	const jpeg = new Uint8Array(await blob.arrayBuffer());
	const pdf = singleImagePDF(jpeg, canvas.width, canvas.height, preset);
	downloadBlob(new Blob([pdf], { type: 'application/pdf' }), filename);
}

function quoteIdentifier(value: string, driver: DBStudioDriver) {
	const quote = driver === 'postgres' ? '"' : '`';
	return `${quote}${value.replaceAll(quote, quote + quote)}${quote}`;
}

export function buildSchemaSQL(details: DBStudioTableDetails[], driver: DBStudioDriver) {
	const lines: string[] = [
		'-- MyPaaS DB Studio structural schema export',
		'-- Generated from introspected columns, keys, indexes, and constraints.',
		'-- Column defaults and engine-specific table options are omitted when not exposed by DB Studio metadata.',
		''
	];
	const q = (value: string) => quoteIdentifier(value, driver);

	for (const table of details) {
		const primary = table.columns.filter((column) => column.primaryKey).map((column) => column.name);
		const columnLines = table.columns.map((column) => `  ${q(column.name)} ${column.dataType}${column.nullable ? '' : ' NOT NULL'}`);
		if (primary.length > 0) columnLines.push(`  PRIMARY KEY (${primary.map(q).join(', ')})`);
		lines.push(`CREATE TABLE ${q(table.schema)}.${q(table.name)} (`, columnLines.join(',\n'), ');', '');
	}

	for (const table of details) {
		for (const foreignKey of table.foreignKeys) {
			lines.push(`ALTER TABLE ${q(table.schema)}.${q(table.name)} ADD CONSTRAINT ${q(foreignKey.name)} FOREIGN KEY (${q(foreignKey.column)}) REFERENCES ${q(foreignKey.referencedSchema)}.${q(foreignKey.referencedTable)} (${q(foreignKey.referencedColumn)}) ON UPDATE ${foreignKey.onUpdate} ON DELETE ${foreignKey.onDelete};`);
		}
		for (const index of table.indexes) {
			if (index.primary || index.columns.length === 0) continue;
			lines.push(`CREATE ${index.unique ? 'UNIQUE ' : ''}INDEX ${q(index.name)} ON ${q(table.schema)}.${q(table.name)} (${index.columns.map(q).join(', ')});`);
		}
		lines.push('');
	}
	return lines.join('\n');
}

export function downloadSQL(sql: string, filename: string) {
	downloadBlob(new Blob([sql], { type: 'text/sql;charset=utf-8' }), filename);
}
