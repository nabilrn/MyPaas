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

function visibleColumnCount(node: ERDNode, density: ERDDensity) {
	return density === 'compact' ? Math.min(5, node.detail.columns.length) : Math.min(7, node.detail.columns.length);
}

function columnY(node: ERDNode, column: string, density: ERDDensity) {
	const rows = visibleColumnCount(node, density);
	const rawIndex = node.detail.columns.findIndex((item) => item.name === column);
	const index = rawIndex < 0 ? 0 : Math.min(rawIndex, Math.max(rows - 1, 0));
	const headerHeight = density === 'compact' ? 55 : 58;
	const rowHeight = density === 'compact' ? 18 : 20;
	return node.y + headerHeight + index * rowHeight + rowHeight / 2;
}

export function relationPath(edge: ERDEdge, graph: ERDGraph, direction: 'LR' | 'TB', density: ERDDensity) {
	const edgeIndex = Math.max(0, graph.edges.findIndex((item) => item.id === edge.id));
	const lane = ((edgeIndex % 7) - 3) * 8;
	const fromCenterX = edge.from.x + edge.from.width / 2;
	const toCenterX = edge.to.x + edge.to.width / 2;

	if (direction === 'LR') {
		const toRight = toCenterX >= fromCenterX;
		const fromX = toRight ? edge.from.x + edge.from.width : edge.from.x;
		const toX = toRight ? edge.to.x : edge.to.x + edge.to.width;
		const fromY = columnY(edge.from, edge.foreignKey.column, density);
		const toY = columnY(edge.to, edge.foreignKey.referencedColumn, density);
		const middleX = (fromX + toX) / 2 + lane;
		return { path: `M ${fromX} ${fromY} H ${middleX} V ${toY} H ${toX}`, labelX: middleX, labelY: (fromY + toY) / 2 };
	}

	const toBelow = edge.to.y + edge.to.height / 2 >= edge.from.y + edge.from.height / 2;
	const fromY = toBelow ? edge.from.y + edge.from.height : edge.from.y;
	const toY = toBelow ? edge.to.y : edge.to.y + edge.to.height;
	const fromX = fromCenterX;
	const toX = toCenterX;
	const middleY = (fromY + toY) / 2 + lane;
	return { path: `M ${fromX} ${fromY} V ${middleY} H ${toX} V ${toY}`, labelX: (fromX + toX) / 2, labelY: middleY };
}

export function buildERDSVG(graph: ERDGraph, direction: 'LR' | 'TB', options: ERDExportOptions) {
	const ratio = presetRatio(options.preset);
	const margin = 48;
	const naturalWidth = Math.max(960, graph.width + margin * 2);
	const naturalHeight = Math.max(600, graph.height + margin * 2);
	const pageWidth = ratio ? 1600 : naturalWidth;
	const pageHeight = ratio ? Math.round(pageWidth / ratio) : naturalHeight;
	const scale = Math.min((pageWidth - margin * 2) / Math.max(graph.width, 1), (pageHeight - margin * 2) / Math.max(graph.height, 1), 1);
	const offsetX = (pageWidth - graph.width * scale) / 2;
	const offsetY = (pageHeight - graph.height * scale) / 2;
	const rows = (node: ERDNode) => visibleColumnCount(node, options.density);

	const edges = graph.edges.map((edge) => {
		const relation = relationPath(edge, graph, direction, options.density);
		const label = `${edge.foreignKey.column} → ${edge.foreignKey.referencedColumn}`;
		return `<path d="${relation.path}" fill="none" stroke="#6b7280" stroke-width="2" marker-end="url(#arrow)"/>${options.showRelationLabels ? `<text x="${relation.labelX}" y="${relation.labelY - 6}" text-anchor="middle" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="11" fill="#374151" paint-order="stroke" stroke="#ffffff" stroke-width="4" stroke-linejoin="round">${escapeXML(label)}</text>` : ''}`;
	}).join('');

	const nodes = graph.nodes.map((node) => {
		const visible = node.detail.columns.slice(0, rows(node));
		const columns = visible.map((column, index) => {
			const y = node.y + 68 + index * (options.density === 'compact' ? 18 : 20);
			const type = options.showDataTypes ? `<text x="${node.x + node.width - 12}" y="${y}" text-anchor="end" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="10" fill="#6b7280">${escapeXML(column.dataType)}</text>` : '';
			return `<text x="${node.x + 12}" y="${y}" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="10.5" fill="#1f2937">${column.primaryKey ? 'PK ' : ''}${escapeXML(column.name)}</text>${type}`;
		}).join('');
		const more = node.detail.columns.length > rows(node)
			? `<text x="${node.x + 12}" y="${node.y + node.height - 12}" font-family="Inter, Arial, sans-serif" font-size="10" fill="#9ca3af">+${node.detail.columns.length - rows(node)} more</text>`
			: '';
		return `<g><rect x="${node.x}" y="${node.y}" width="${node.width}" height="${node.height}" rx="8" fill="#ffffff" stroke="#9ca3af" stroke-width="1.25"/><line x1="${node.x}" y1="${node.y + 50}" x2="${node.x + node.width}" y2="${node.y + 50}" stroke="#e5e7eb"/><text x="${node.x + 12}" y="${node.y + 17}" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="9" fill="#9ca3af">${escapeXML(node.detail.schema)}</text><text x="${node.x + 12}" y="${node.y + 37}" font-family="Inter, Arial, sans-serif" font-size="14" font-weight="600" fill="#111827">${escapeXML(node.detail.name)}</text>${columns}${more}</g>`;
	}).join('');

	const title = options.title
		? `<text x="${margin}" y="${Math.max(28, offsetY - 16)}" font-family="Inter, Arial, sans-serif" font-size="16" font-weight="600" fill="#111827">${escapeXML(options.title)}</text>`
		: '';

	return `<svg xmlns="http://www.w3.org/2000/svg" width="${pageWidth}" height="${pageHeight}" viewBox="0 0 ${pageWidth} ${pageHeight}"><rect width="100%" height="100%" fill="#ffffff"/>${title}<defs><marker id="arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="#4b5563"/></marker></defs><g transform="translate(${offsetX} ${offsetY}) scale(${scale})">${edges}${nodes}</g></svg>`;
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

async function svgCanvas(svg: string, width = 2200) {
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
		const ratio = image.naturalWidth / Math.max(image.naturalHeight, 1);
		const canvas = document.createElement('canvas');
		canvas.width = width;
		canvas.height = Math.max(1, Math.round(width / ratio));
		const context = canvas.getContext('2d');
		if (!context) throw new Error('Canvas is not available');
		context.fillStyle = '#ffffff';
		context.fillRect(0, 0, canvas.width, canvas.height);
		context.drawImage(image, 0, 0, canvas.width, canvas.height);
		return canvas;
	} finally {
		URL.revokeObjectURL(url);
	}
}

export async function downloadPNG(svg: string, filename: string) {
	const canvas = await svgCanvas(svg);
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
	const pageWidth = 841.89;
	const pageHeight = preset === 'a4-landscape' ? 595.28 : pageWidth / ratio;
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
	const canvas = await svgCanvas(svg, 2400);
	const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', 0.94));
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
