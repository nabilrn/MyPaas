export type EnvParseStatus = 'valid' | 'invalid' | 'duplicate';

export interface ParsedEnvEntry {
	line: number;
	raw: string;
	rawKey: string;
	key: string;
	value: string;
	status: EnvParseStatus;
	error: string;
}

const KEY_PATTERN = /^[A-Z_][A-Z0-9_]*$/;

export function normalizeEnvKey(value: string) {
	return value.trim().toUpperCase();
}

export function isValidEnvKey(value: string) {
	return KEY_PATTERN.test(normalizeEnvKey(value));
}

export function parseEnvContent(content: string): ParsedEnvEntry[] {
	const entries = content.replace(/^\uFEFF/, '').split(/\r?\n/).flatMap((line, index) => parseLine(line, index + 1));
	const counts = new Map<string, number>();

	for (const entry of entries) {
		if (entry.status === 'valid') {
			counts.set(entry.key, (counts.get(entry.key) ?? 0) + 1);
		}
	}

	return entries.map((entry) => {
		if (entry.status === 'valid' && (counts.get(entry.key) ?? 0) > 1) {
			return { ...entry, status: 'duplicate', error: 'Duplicate key in import' };
		}
		return entry;
	});
}

function parseLine(line: string, lineNumber: number): ParsedEnvEntry[] {
	const trimmed = line.trim();
	if (!trimmed || trimmed.startsWith('#')) {
		return [];
	}

	const source = trimmed.startsWith('export ') ? trimmed.slice('export '.length).trimStart() : trimmed;
	const equalsIndex = source.indexOf('=');
	if (equalsIndex === -1) {
		return [invalidEntry(lineNumber, line, source, 'Missing = separator')];
	}

	const rawKey = source.slice(0, equalsIndex).trim();
	const key = normalizeEnvKey(rawKey);
	if (!KEY_PATTERN.test(key)) {
		return [invalidEntry(lineNumber, line, rawKey, 'Invalid key')];
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
		status: 'valid',
		error: ''
	}];
}

function parseValue(rawValue: string): { value: string; error: string } {
	const value = rawValue.trimStart();
	if (!value) {
		return { value: '', error: '' };
	}

	if (value.startsWith('"')) {
		return parseQuotedValue(value, '"');
	}
	if (value.startsWith("'")) {
		return parseQuotedValue(value, "'");
	}

	return { value: stripInlineComment(value).trim(), error: '' };
}

function parseQuotedValue(value: string, quote: '"' | "'"): { value: string; error: string } {
	let out = '';
	let escaped = false;

	for (let i = 1; i < value.length; i += 1) {
		const char = value[i];

		if (quote === '"' && escaped) {
			out += unescapeDoubleQuotedChar(char);
			escaped = false;
			continue;
		}
		if (quote === '"' && char === '\\') {
			escaped = true;
			continue;
		}
		if (char === quote) {
			const rest = value.slice(i + 1).trim();
			if (rest && !rest.startsWith('#')) {
				return { value: '', error: 'Unexpected text after quoted value' };
			}
			return { value: out, error: '' };
		}
		out += char;
	}

	return { value: '', error: 'Unterminated quoted value' };
}

function unescapeDoubleQuotedChar(char: string) {
	if (char === 'n') return '\n';
	if (char === 'r') return '\r';
	if (char === 't') return '\t';
	return char;
}

function stripInlineComment(value: string) {
	for (let i = 0; i < value.length; i += 1) {
		if (value[i] === '#' && i > 0 && /\s/.test(value[i - 1])) {
			return value.slice(0, i);
		}
	}
	return value;
}

function invalidEntry(line: number, raw: string, rawKey: string, error: string): ParsedEnvEntry {
	return {
		line,
		raw,
		rawKey,
		key: normalizeEnvKey(rawKey),
		value: '',
		status: 'invalid',
		error
	};
}
