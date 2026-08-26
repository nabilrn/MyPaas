export type AppTemplateSource =
	| { type: 'registry'; imageRef: string }
	| { type: 'compose'; baseDirectory: string; composeFilePath: string; mainService: string };

export type AppTemplateEnvKind = 'text' | 'secret' | 'public-url';
export type AppTemplateSecretFormat = 'hex' | 'base64url';

export interface AppTemplateEnvField {
	key: string;
	label: string;
	kind: AppTemplateEnvKind;
	description: string;
	defaultValue?: string;
	bytes?: number;
	format?: AppTemplateSecretFormat;
}

export interface AppTemplate {
	id: string;
	name: string;
	description: string;
	category: string;
	appPort: number;
	memoryLimitMb: number;
	cpuLimit: number;
	source: AppTemplateSource;
	env: AppTemplateEnvField[];
	persistent: boolean;
	limitations: string[];
}

const templateRepo = 'https://github.com/nabilrn/MyPaas.git';

export const appTemplateRepository = {
	repoUrl: templateRepo,
	branch: 'main'
} as const;

export const appTemplates: AppTemplate[] = [
	{
		id: 'excalidraw',
		name: 'Excalidraw',
		description: 'Lightweight collaborative whiteboard client served from the official container image.',
		category: 'Diagramming',
		appPort: 80,
		memoryLimitMb: 512,
		cpuLimit: 0.5,
		source: { type: 'registry', imageRef: 'excalidraw/excalidraw:latest' },
		env: [],
		persistent: false,
		limitations: ['The official self-hosted client does not include Excalidraw sharing/collaboration services.']
	},
	{
		id: 'uptime-kuma',
		name: 'Uptime Kuma',
		description: 'Self-hosted uptime monitor with a durable Docker-managed data volume.',
		category: 'Monitoring',
		appPort: 3001,
		memoryLimitMb: 768,
		cpuLimit: 0.75,
		source: { type: 'compose', baseDirectory: 'templates/manifests/uptime-kuma', composeFilePath: 'compose.yml', mainService: 'uptime-kuma' },
		env: [],
		persistent: true,
		limitations: []
	},
	{
		id: 'n8n',
		name: 'n8n',
		description: 'Automation platform with persistent application state and a generated encryption key.',
		category: 'Automation',
		appPort: 5678,
		memoryLimitMb: 1536,
		cpuLimit: 1,
		source: { type: 'compose', baseDirectory: 'templates/manifests/n8n', composeFilePath: 'compose.yml', mainService: 'n8n' },
		env: [
			{ key: 'N8N_ENCRYPTION_KEY', label: 'Encryption key', kind: 'secret', bytes: 32, format: 'base64url', description: 'Generated locally before project creation and stored through MyPaas encrypted environment storage.' },
			{ key: 'GENERIC_TIMEZONE', label: 'Workflow timezone', kind: 'text', defaultValue: 'Asia/Jakarta', description: 'Timezone used by scheduled workflows.' },
			{ key: 'TZ', label: 'Container timezone', kind: 'text', defaultValue: 'Asia/Jakarta', description: 'Container timezone.' }
		],
		persistent: true,
		limitations: ['Docker-in-Docker / sandbox stacks are outside the current MyPaas security boundary.']
	},
	{
		id: 'umami',
		name: 'Umami',
		description: 'Web analytics application with PostgreSQL and generated application/database secrets.',
		category: 'Analytics',
		appPort: 3000,
		memoryLimitMb: 1024,
		cpuLimit: 1,
		source: { type: 'compose', baseDirectory: 'templates/manifests/umami', composeFilePath: 'compose.yml', mainService: 'umami' },
		env: [
			{ key: 'UMAMI_DB_PASSWORD', label: 'Database password', kind: 'secret', bytes: 24, format: 'hex', description: 'Shared by the Umami service and its project-local PostgreSQL service.' },
			{ key: 'APP_SECRET', label: 'Application secret', kind: 'secret', bytes: 32, format: 'base64url', description: 'Generated application secret.' },
			{ key: 'TWO_FACTOR_ENCRYPTION_KEY', label: '2FA encryption key', kind: 'secret', bytes: 32, format: 'hex', description: 'Generated 64-character encryption key.' }
		],
		persistent: true,
		limitations: []
	},
	{
		id: 'ghost',
		name: 'Ghost',
		description: 'Ghost CMS with MySQL, persistent content, and generated database credentials.',
		category: 'CMS',
		appPort: 2368,
		memoryLimitMb: 1024,
		cpuLimit: 1,
		source: { type: 'compose', baseDirectory: 'templates/manifests/ghost', composeFilePath: 'compose.yml', mainService: 'ghost' },
		env: [
			{ key: 'GHOST_URL', label: 'Public URL', kind: 'public-url', description: 'Generated from the MyPaas project hostname.' },
			{ key: 'GHOST_DB_PASSWORD', label: 'Database password', kind: 'secret', bytes: 24, format: 'hex', description: 'Credential used by Ghost to connect to MySQL.' },
			{ key: 'GHOST_DB_ROOT_PASSWORD', label: 'MySQL root password', kind: 'secret', bytes: 24, format: 'hex', description: 'Root credential for the project-local MySQL service.' }
		],
		persistent: true,
		limitations: ['This template covers the Ghost + MySQL baseline, not optional auxiliary services.']
	},
	{
		id: 'nocodb',
		name: 'NocoDB',
		description: 'Multi-service NocoDB stack with worker, PostgreSQL, Redis, and durable volumes.',
		category: 'Developer tool',
		appPort: 8080,
		memoryLimitMb: 1536,
		cpuLimit: 1.25,
		source: { type: 'compose', baseDirectory: 'templates/manifests/nocodb', composeFilePath: 'compose.yml', mainService: 'nocodb' },
		env: [
			{ key: 'NOCODB_DB_PASSWORD', label: 'Database password', kind: 'secret', bytes: 24, format: 'hex', description: 'Credential shared by NocoDB and its project-local PostgreSQL service.' }
		],
		persistent: true,
		limitations: []
	}
];

function randomBytes(length: number): Uint8Array {
	if (!globalThis.crypto?.getRandomValues) {
		throw new Error('Secure random generation is not available in this browser');
	}
	const out = new Uint8Array(length);
	globalThis.crypto.getRandomValues(out);
	return out;
}

function hex(bytes: Uint8Array): string {
	return Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('');
}

function base64url(bytes: Uint8Array): string {
	let binary = '';
	for (const value of bytes) binary += String.fromCharCode(value);
	return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

export function generateTemplateSecret(field: AppTemplateEnvField): string {
	if (field.kind !== 'secret') return field.defaultValue ?? '';
	const bytes = randomBytes(field.bytes ?? 32);
	return field.format === 'hex' ? hex(bytes) : base64url(bytes);
}

export function initialTemplateEnv(template: AppTemplate): Record<string, string> {
	return Object.fromEntries(template.env.map((field) => [
		field.key,
		field.kind === 'secret' ? generateTemplateSecret(field) : field.defaultValue ?? ''
	]));
}
