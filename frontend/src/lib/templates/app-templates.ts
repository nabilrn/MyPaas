export type AppTemplateSource =
	| { type: 'registry'; imageRef: string }
	| { type: 'dockerfile'; repoUrl: string; branch: string; baseDirectory?: string }
	| { type: 'compose'; baseDirectory: string; composeFilePath: string; mainService: string };

export type AppTemplateEnvKind = 'text' | 'secret' | 'public-url';
export type AppTemplateSecretFormat = 'hex' | 'base64url';
export type AppTemplateCompatibilityStatus = 'catalogued-pattern';

export interface AppTemplateEnvField {
	key: string;
	label: string;
	kind: AppTemplateEnvKind;
	description: string;
	defaultValue?: string;
	bytes?: number;
	format?: AppTemplateSecretFormat;
	required?: boolean;
}

export interface AppTemplateCompatibility {
	catalogId: string;
	status: AppTemplateCompatibilityStatus;
}

export interface AppTemplateServiceResource {
	memoryLimitMb: number;
	cpuLimit: number;
}

export interface AppTemplate {
	id: string;
	name: string;
	description: string;
	category: string;
	appPort: number;
	memoryLimitMb: number;
	cpuLimit: number;
	serviceResources?: Record<string, AppTemplateServiceResource>;
	source: AppTemplateSource;
	env: AppTemplateEnvField[];
	persistent: boolean;
	limitations: string[];
	compatibility: AppTemplateCompatibility;
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
		limitations: ['The official self-hosted client does not include Excalidraw sharing/collaboration services.'],
		compatibility: { catalogId: 'excalidraw', status: 'catalogued-pattern' }
	},
	{
		id: 'drawdb',
		name: 'drawDB',
		description: 'Database diagram editor built directly from the upstream Dockerfile.',
		category: 'Diagramming',
		appPort: 80,
		memoryLimitMb: 512,
		cpuLimit: 0.5,
		source: { type: 'dockerfile', repoUrl: 'https://github.com/drawdb-io/drawdb.git', branch: 'main' },
		env: [],
		persistent: false,
		limitations: ['This template covers the base editor; optional sharing services are outside the catalogued pattern.'],
		compatibility: { catalogId: 'drawdb', status: 'catalogued-pattern' }
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
		limitations: [],
		compatibility: { catalogId: 'uptime-kuma', status: 'catalogued-pattern' }
	},
	{
		id: 'meilisearch',
		name: 'Meilisearch',
		description: 'Search engine with persistent index data and a generated production master key.',
		category: 'Search',
		appPort: 7700,
		memoryLimitMb: 768,
		cpuLimit: 0.75,
		source: { type: 'compose', baseDirectory: 'templates/manifests/meilisearch', composeFilePath: 'compose.yml', mainService: 'meilisearch' },
		env: [
			{ key: 'MEILI_MASTER_KEY', label: 'Master key', kind: 'secret', bytes: 32, format: 'base64url', required: true, description: 'Generated access key used by the production Meilisearch instance.' }
		],
		persistent: true,
		limitations: [],
		compatibility: { catalogId: 'meilisearch', status: 'catalogued-pattern' }
	},
	{
		id: 'directus',
		name: 'Directus',
		description: 'Realtime data platform using SQLite-backed persistence for a compact single-host install.',
		category: 'Data platform',
		appPort: 8055,
		memoryLimitMb: 1024,
		cpuLimit: 1,
		source: { type: 'compose', baseDirectory: 'templates/manifests/directus', composeFilePath: 'compose.yml', mainService: 'directus' },
		env: [
			{ key: 'SECRET', label: 'Application secret', kind: 'secret', bytes: 32, format: 'base64url', required: true, description: 'Generated Directus application secret.' },
			{ key: 'ADMIN_EMAIL', label: 'Admin email', kind: 'text', required: true, description: 'Email address for the initial Directus administrator.' },
			{ key: 'ADMIN_PASSWORD', label: 'Admin password', kind: 'secret', bytes: 24, format: 'base64url', required: true, description: 'Generated password for the initial Directus administrator. Replace it here if you prefer your own value.' }
		],
		persistent: true,
		limitations: ['This template uses SQLite for the compact single-host baseline qualified by the compatibility catalog.'],
		compatibility: { catalogId: 'directus', status: 'catalogued-pattern' }
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
			{ key: 'N8N_ENCRYPTION_KEY', label: 'Encryption key', kind: 'secret', bytes: 32, format: 'base64url', required: true, description: 'Generated locally before project creation and stored through MyPaas encrypted environment storage.' },
			{ key: 'GENERIC_TIMEZONE', label: 'Workflow timezone', kind: 'text', defaultValue: 'Asia/Jakarta', description: 'Timezone used by scheduled workflows.' },
			{ key: 'TZ', label: 'Container timezone', kind: 'text', defaultValue: 'Asia/Jakarta', description: 'Container timezone.' }
		],
		persistent: true,
		limitations: ['Docker-in-Docker / sandbox stacks are outside the current MyPaas security boundary.'],
		compatibility: { catalogId: 'n8n', status: 'catalogued-pattern' }
	},
	{
		id: 'umami',
		name: 'Umami',
		description: 'Web analytics application with PostgreSQL and generated application/database secrets.',
		category: 'Analytics',
		appPort: 3000,
		memoryLimitMb: 1024,
		cpuLimit: 1,
		serviceResources: {
			db: { memoryLimitMb: 512, cpuLimit: 0.5 }
		},
		source: { type: 'compose', baseDirectory: 'templates/manifests/umami', composeFilePath: 'compose.yml', mainService: 'umami' },
		env: [
			{ key: 'UMAMI_DB_PASSWORD', label: 'Database password', kind: 'secret', bytes: 24, format: 'hex', required: true, description: 'Shared by the Umami service and its project-local PostgreSQL service.' },
			{ key: 'APP_SECRET', label: 'Application secret', kind: 'secret', bytes: 32, format: 'base64url', required: true, description: 'Generated application secret.' },
			{ key: 'TWO_FACTOR_ENCRYPTION_KEY', label: '2FA encryption key', kind: 'secret', bytes: 32, format: 'hex', required: true, description: 'Generated 64-character encryption key.' }
		],
		persistent: true,
		limitations: [],
		compatibility: { catalogId: 'umami', status: 'catalogued-pattern' }
	},
	{
		id: 'ghost',
		name: 'Ghost',
		description: 'Ghost CMS with MySQL, persistent content, and generated database credentials.',
		category: 'CMS',
		appPort: 2368,
		memoryLimitMb: 1024,
		cpuLimit: 1,
		serviceResources: {
			db: { memoryLimitMb: 768, cpuLimit: 0.5 }
		},
		source: { type: 'compose', baseDirectory: 'templates/manifests/ghost', composeFilePath: 'compose.yml', mainService: 'ghost' },
		env: [
			{ key: 'GHOST_URL', label: 'Public URL', kind: 'public-url', required: true, description: 'Generated from the MyPaas project hostname.' },
			{ key: 'GHOST_DB_PASSWORD', label: 'Database password', kind: 'secret', bytes: 24, format: 'hex', required: true, description: 'Credential used by Ghost to connect to MySQL.' },
			{ key: 'GHOST_DB_ROOT_PASSWORD', label: 'MySQL root password', kind: 'secret', bytes: 24, format: 'hex', required: true, description: 'Root credential for the project-local MySQL service.' }
		],
		persistent: true,
		limitations: ['This template covers the Ghost + MySQL baseline, not optional auxiliary services.'],
		compatibility: { catalogId: 'ghost', status: 'catalogued-pattern' }
	},
	{
		id: 'nocodb',
		name: 'NocoDB',
		description: 'Multi-service NocoDB stack with worker, PostgreSQL, Redis, and durable volumes.',
		category: 'Developer tool',
		appPort: 8080,
		memoryLimitMb: 1536,
		cpuLimit: 1.25,
		serviceResources: {
			worker: { memoryLimitMb: 768, cpuLimit: 0.75 },
			db: { memoryLimitMb: 768, cpuLimit: 0.5 },
			redis: { memoryLimitMb: 256, cpuLimit: 0.25 }
		},
		source: { type: 'compose', baseDirectory: 'templates/manifests/nocodb', composeFilePath: 'compose.yml', mainService: 'nocodb' },
		env: [
			{ key: 'NOCODB_DB_PASSWORD', label: 'Database password', kind: 'secret', bytes: 24, format: 'hex', required: true, description: 'Credential shared by NocoDB and its project-local PostgreSQL service.' }
		],
		persistent: true,
		limitations: [],
		compatibility: { catalogId: 'nocodb', status: 'catalogued-pattern' }
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

export function missingRequiredTemplateEnv(template: AppTemplate, values: Record<string, string>, publicURL = ''): string[] {
	return template.env
		.filter((field) => {
			if (!field.required) return false;
			const value = field.kind === 'public-url' ? publicURL : (values[field.key] ?? '');
			return value.trim().length === 0;
		})
		.map((field) => field.key);
}
