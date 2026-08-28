export type AppTemplateSource =
	| { type: 'registry'; imageRef: string }
	| { type: 'dockerfile'; repoUrl: string; branch: string; baseDirectory?: string }
	| { type: 'compose'; baseDirectory: string; composeFilePath: string; mainService: string };

export type AppTemplateEnvKind = 'text' | 'secret' | 'public-url' | 'public-host' | 'route-url';
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
	routeName?: string;
}

export interface AppTemplateCompatibility {
	catalogId: string;
	status: AppTemplateCompatibilityStatus;
}

export interface AppTemplateServiceResource {
	memoryLimitMb: number;
	cpuLimit: number;
}

export interface AppTemplateRoute {
	name: string;
	service: string;
	containerPort: number;
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
	additionalRoutes?: AppTemplateRoute[];
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
		serviceResources: { db: { memoryLimitMb: 512, cpuLimit: 0.5 } },
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
		serviceResources: { db: { memoryLimitMb: 768, cpuLimit: 0.5 } },
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
	},
	{
		id: 'forgejo',
		name: 'Forgejo',
		description: 'Self-hosted Git forge with persistent repository data and HTTP Git access.',
		category: 'Developer platform',
		appPort: 3000,
		memoryLimitMb: 1024,
		cpuLimit: 1,
		source: { type: 'compose', baseDirectory: 'templates/manifests/forgejo', composeFilePath: 'compose.yml', mainService: 'forgejo' },
		env: [
			{ key: 'FORGEJO_DOMAIN', label: 'Public host', kind: 'public-host', required: true, description: 'Managed from the MyPaas project hostname and used by Forgejo for generated HTTP clone URLs.' },
			{ key: 'FORGEJO_ROOT_URL', label: 'Public URL', kind: 'public-url', required: true, description: 'Managed from the MyPaas project URL.' }
		],
		persistent: true,
		limitations: ['HTTP UI and HTTP Git are covered. Forgejo SSH is disabled; MyPaas additional routes are HTTP-only.'],
		compatibility: { catalogId: 'forgejo', status: 'catalogued-pattern' }
	},
	{
		id: 'paperless-ngx',
		name: 'Paperless-ngx',
		description: 'Document management platform with PostgreSQL, Redis, durable document storage, and generated credentials.',
		category: 'Document platform',
		appPort: 8000,
		memoryLimitMb: 1536,
		cpuLimit: 1,
		serviceResources: {
			db: { memoryLimitMb: 768, cpuLimit: 0.5 },
			broker: { memoryLimitMb: 256, cpuLimit: 0.25 }
		},
		source: { type: 'compose', baseDirectory: 'templates/manifests/paperless-ngx', composeFilePath: 'compose.yml', mainService: 'webserver' },
		env: [
			{ key: 'PAPERLESS_DB_PASSWORD', label: 'Database password', kind: 'secret', bytes: 24, format: 'hex', required: true, description: 'Credential shared by Paperless-ngx and its project-local PostgreSQL service.' },
			{ key: 'PAPERLESS_SECRET_KEY', label: 'Application secret', kind: 'secret', bytes: 32, format: 'base64url', required: true, description: 'Generated Paperless-ngx application secret.' },
			{ key: 'PAPERLESS_URL', label: 'Public URL', kind: 'public-url', required: true, description: 'Managed from the MyPaas project URL.' },
			{ key: 'PAPERLESS_TIME_ZONE', label: 'Timezone', kind: 'text', defaultValue: 'Asia/Jakarta', description: 'Timezone used by Paperless-ngx.' }
		],
		persistent: true,
		limitations: ['Optional Tika/Gotenberg document-conversion services are not included in this baseline template.'],
		compatibility: { catalogId: 'paperless-ngx', status: 'catalogued-pattern' }
	},
	{
		id: 'openclaw',
		name: 'OpenClaw',
		description: 'Pre-built OpenClaw gateway with persistent state and a generated gateway token.',
		category: 'Agent gateway',
		appPort: 18789,
		memoryLimitMb: 1536,
		cpuLimit: 1,
		serviceResources: { 'openclaw-bootstrap': { memoryLimitMb: 512, cpuLimit: 0.5 } },
		source: { type: 'compose', baseDirectory: 'templates/manifests/openclaw', composeFilePath: 'compose.yml', mainService: 'openclaw-gateway' },
		env: [
			{ key: 'OPENCLAW_GATEWAY_TOKEN', label: 'Gateway token', kind: 'secret', bytes: 32, format: 'base64url', required: true, description: 'Generated token protecting access to the OpenClaw gateway.' },
			{ key: 'TZ', label: 'Container timezone', kind: 'text', defaultValue: 'Asia/Jakarta', description: 'Container timezone.' }
		],
		persistent: true,
		limitations: [
			'Uses the pre-built gateway image only. CLI network sharing, host bind mounts, extra gateway ports, and Docker-socket sandboxing remain outside the current MyPaas safety boundary.',
			'Initial remote Control UI access can require OpenClaw device pairing approval; that application-level approval is not bypassed by MyPaas.'
		],
		compatibility: { catalogId: 'openclaw', status: 'catalogued-pattern' }
	},
	{
		id: 'minio',
		name: 'MinIO',
		description: 'S3-compatible object storage with persistent data plus a separately routed web console.',
		category: 'Object storage',
		appPort: 9000,
		memoryLimitMb: 1024,
		cpuLimit: 1,
		additionalRoutes: [
			{ name: 'console', service: 'minio', containerPort: 9001 }
		],
		source: { type: 'compose', baseDirectory: 'templates/manifests/minio', composeFilePath: 'compose.yml', mainService: 'minio' },
		env: [
			{ key: 'MINIO_ROOT_USER', label: 'Root user', kind: 'secret', bytes: 12, format: 'base64url', required: true, description: 'Generated MinIO root username. Save it before first login.' },
			{ key: 'MINIO_ROOT_PASSWORD', label: 'Root password', kind: 'secret', bytes: 32, format: 'base64url', required: true, description: 'Generated MinIO root password.' },
			{ key: 'MINIO_BROWSER_REDIRECT_URL', label: 'Console URL', kind: 'route-url', routeName: 'console', required: true, description: 'Managed from the additional MyPaas console route used by MinIO behind the reverse proxy.' }
		],
		persistent: true,
		limitations: ['The template exposes only HTTP(S) S3 API and Console routes. Raw TCP protocols and arbitrary host-port publishing remain out of scope.'],
		compatibility: { catalogId: 'minio', status: 'catalogued-pattern' }
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

export function templateEnvValue(
	field: AppTemplateEnvField,
	values: Record<string, string>,
	publicURL = '',
	publicHost = '',
	routeURLs: Record<string, string> = {}
): string {
	if (field.kind === 'public-url') return publicURL;
	if (field.kind === 'public-host') return publicHost;
	if (field.kind === 'route-url') return field.routeName ? routeURLs[field.routeName] ?? '' : '';
	return values[field.key] ?? '';
}

export function missingRequiredTemplateEnv(
	template: AppTemplate,
	values: Record<string, string>,
	publicURL = '',
	publicHost = '',
	routeURLs: Record<string, string> = {}
): string[] {
	return template.env
		.filter((field) => field.required && templateEnvValue(field, values, publicURL, publicHost, routeURLs).trim().length === 0)
		.map((field) => field.key);
}
