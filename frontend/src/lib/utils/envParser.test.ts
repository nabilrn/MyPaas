import { describe, expect, it } from 'vitest';

import { parseEnvContent } from './envParser';

describe('parseEnvContent', () => {
	it('parses common dotenv values', () => {
		const entries = parseEnvContent(`
# comment
app_port=3000
DATABASE_URL="postgres://user:pass@localhost:5432/app"
EMPTY=
SECRET='value with # hash'
PUBLIC_URL=https://example.com/#anchor
export FEATURE_FLAG=true # inline comment
`);

		expect(entries).toMatchObject([
			{ key: 'APP_PORT', value: '3000', status: 'valid' },
			{ key: 'DATABASE_URL', value: 'postgres://user:pass@localhost:5432/app', status: 'valid' },
			{ key: 'EMPTY', value: '', status: 'valid' },
			{ key: 'SECRET', value: 'value with # hash', status: 'valid' },
			{ key: 'PUBLIC_URL', value: 'https://example.com/#anchor', status: 'valid' },
			{ key: 'FEATURE_FLAG', value: 'true', status: 'valid' }
		]);
	});

	it('unescapes double quoted values', () => {
		const [entry] = parseEnvContent(String.raw`PRIVATE_KEY="line1\nline2\t\"quoted\""`);

		expect(entry.value).toBe('line1\nline2\t"quoted"');
	});

	it('reports invalid lines', () => {
		const entries = parseEnvContent(`
MISSING_SEPARATOR
BAD-KEY=value
BROKEN="unterminated
`);

		expect(entries).toMatchObject([
			{ line: 2, key: 'MISSING_SEPARATOR', status: 'invalid', error: 'Missing = separator' },
			{ line: 3, key: 'BAD-KEY', status: 'invalid', error: 'Invalid key' },
			{ line: 4, key: 'BROKEN', status: 'invalid', error: 'Unterminated quoted value' }
		]);
	});

	it('marks duplicate normalized keys', () => {
		const entries = parseEnvContent(`
token=first
TOKEN=second
OTHER=value
`);

		expect(entries).toMatchObject([
			{ key: 'TOKEN', status: 'duplicate' },
			{ key: 'TOKEN', status: 'duplicate' },
			{ key: 'OTHER', status: 'valid' }
		]);
	});
});
