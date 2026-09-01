// ─── Domain enums ────────────────────────────────────────────────────────────

export type ProjectStatus = 'pending' | 'running' | 'stopped' | 'crashed' | 'building';
export type DeployMode    = 'dockerfile' | 'compose' | 'static' | 'image';
export type ProjectSourceType = 'git' | 'registry';
export type DeployStatus  = 'queued' | 'cloning' | 'building' | 'starting' | 'running' | 'failed' | 'stopped' | 'rolled_back';
export type UserRole      = 'owner' | 'collaborator';
export type TriggeredBy   = 'manual' | 'webhook' | 'rollback';
export type ResourceProfile = 'static' | 'go-small' | 'node-python' | 'compose-main' | 'custom';

// ─── Domain models ───────────────────────────────────────────────────────────

export interface User {
	id:             string;
	email:          string;
	githubId:       string | null;
	githubUsername: string | null;
	avatarUrl:      string | null;
	role:           UserRole;
	createdAt:      string;
	lastLoginAt:    string | null;
}

export interface Project {
	id:                   string;
	userId:               string;
	name:                 string;
	sourceType:           ProjectSourceType;
	repoUrl:              string;
	imageRef:             string | null;
	branch:               string;
	subdomain:            string;
	deployMode:          DeployMode;
	resourceProfile:     ResourceProfile;
	mainService:         string | null;
	appPort:             number;
	webhookSecret:       string;
	allocatedPort:       number | null;
	memoryLimitMb:       number;
	cpuLimit:            number;
	status:              ProjectStatus;
	activeDeploymentId:  string | null;
	composeFilePath:      string | null;
	composeOverridePaths: string[];
	composeProfiles:      string[];
	composeWorkdir:       string | null;
	serviceResources:    Record<string, { memoryLimitMb: number; cpuLimit: number }>;
	staticFrontendPath:   string | null;
	baseDirectory:        string | null;
	createdAt:            string;
	updatedAt:            string;
}

export interface Deployment {
	id:                string;
	projectId:         string;
	commitSha:         string | null;
	commitMessage:     string | null;
	status:            DeployStatus;
	buildLog:          string | null;
	errorMsg:          string | null;
	imageTag:          string | null;
	triggeredBy:       TriggeredBy;
	triggeredByUserId: string | null;
	startedAt:         string;
	finishedAt:        string | null;
}

export interface EnvVar {
	id:        string;
	projectId: string;
	key:       string;
	createdAt: string;
	updatedAt: string;
}

export interface EnvVarConflictValue {
	value: string;
	sources: string[];
}

export interface EnvVarConflict {
	values: EnvVarConflictValue[];
}

export interface EnvVarDiscovery {
	key: string;
	source: string;
	sensitive: boolean;
	defaultValue?: string;
	services?: string[];
	conflict?: EnvVarConflict | null;
}

export interface RepoTreeEntry {
	name: string;
	path: string;
	type: 'file' | 'directory';
	depth: number;
}

export interface RepoInspection {
	branch: string;
	defaultBranch: string;
	branches: string[];
	tree: RepoTreeEntry[];
	treeTruncated: boolean;
}

export interface GitHubRepository {
	id: number;
	name: string;
	fullName: string;
	private: boolean;
	defaultBranch: string;
	cloneUrl: string;
	htmlUrl: string;
	description: string | null;
	updatedAt: string;
}

export interface GitHubRepositoryPage {
	repositories: GitHubRepository[];
	page: number;
	hasNextPage: boolean;
}

export interface ComposeIssue {
	severity: 'error' | 'warning' | 'info';
	code: string;
	service?: string;
	message: string;
}

export interface ComposePortPlan {
	target: number;
	published: string | null;
	protocol: string;
}

export interface ComposeServicePlan {
	name: string;
	role: 'public' | 'internal';
	buildContext: string | null;
	dockerfile: string | null;
	image: string | null;
	ports: ComposePortPlan[];
	expose: number[];
	dependsOn: string[];
}

export interface ComposePlan {
	recommendedMainService: string;
	recommendedAppPort: number;
	routeTarget: string;
	requiredEnvVars: string[];
	services: ComposeServicePlan[];
	issues: ComposeIssue[];
}

export interface ContainerMetrics {
	service:        string;
	cpu:            number;
	memoryMb:       number;
	memoryLimitMb:  number;
	uptime:         string;
}

export interface TimeseriesDataPoint {
	timestamp: string;
	requests:  number;
	bandwidth: number;
}

export interface CloudflareAnalytics {
	total_requests: number;
	bandwidth:      number;
	errors:         number;
	timeseries:     TimeseriesDataPoint[];
}

export interface MetricsSnapshot {
	items:       ContainerMetrics[];
	analytics?:  CloudflareAnalytics;
	collectedAt: string;
}

export interface ComposeResourceSummary {
	projectName: string;
	containers: number;
	volumes: number;
	networks: number;
}

export interface LogLine {
	service:   string;
	line:      string;
	timestamp: string;
}

export interface LogsResponse {
	lines: string[];
	items: Array<{
		service: string;
		line: string;
	}>;
}

export type DBStudioDriver = 'postgres' | 'mysql' | 'mariadb';

export interface DBStudioConnection {
	driver: DBStudioDriver;
	host: string;
	port: number;
	database: string;
	user: string;
	source: string;
}

export interface DBStudioWriteSession {
	id: string;
	expiresAt: string;
}

export interface DBStudioStatus {
	configured: boolean;
	connected: boolean;
	message: string;
	connection: DBStudioConnection | null;
	writeAccess: DBStudioWriteSession | null;
}

export interface DBStudioSchema {
	name: string;
}

export interface DBStudioTable {
	schema: string;
	name: string;
}

export interface DBStudioColumn {
	name: string;
	dataType: string;
	nullable: boolean;
	primaryKey: boolean;
	autoGenerated: boolean;
	enumValues?: string[];
}

export interface DBStudioForeignKey {
	name: string;
	column: string;
	referencedSchema: string;
	referencedTable: string;
	referencedColumn: string;
	onUpdate: string;
	onDelete: string;
}

export interface DBStudioIndex {
	name: string;
	columns: string[];
	unique: boolean;
	primary: boolean;
	method: string;
	definition?: string;
}

export interface DBStudioConstraint {
	name: string;
	type: string;
	columns: string[];
	definition?: string;
}

export interface DBStudioTableDetails {
	schema: string;
	name: string;
	columns: DBStudioColumn[];
	foreignKeys: DBStudioForeignKey[];
	indexes: DBStudioIndex[];
	constraints: DBStudioConstraint[];
}

export interface DBStudioRowFilters {
	search?: string;
	enumFilters?: Record<string, string>;
}

export interface DBStudioRowPage {
	columns: DBStudioColumn[];
	rows: Record<string, unknown>[];
	limit: number;
	offset: number;
	hasMore: boolean;
}

export interface QuotaUsage {
	memoryLimitMb: number;
	memoryUsedMb: number;
	memoryRuntimeMb: number;
	cpuLimit: number;
	cpuUsed: number;
	cpuRuntime: number;
	projectLimit: number;
	projectCount: number;
}

export interface ComposeCandidate {
	path: string;
	score: number;
	depth: number;
}

export interface ComposeFileDetection {
	branch:        string;
	defaultBranch: string;
	branches:      string[];
	candidates:    ComposeCandidate[];
}

export interface DeployModeDetection extends RepoInspection {
	deployMode: DeployMode;
	mainService: string | null;
	services: string[];
	composeFile: string | null;
	hasDockerfile: boolean;
	envVars: EnvVarDiscovery[];
	appPort: number;
	composePlan: ComposePlan | null;
	composeCandidates: ComposeCandidate[];
	staticFrontendCandidates?: string[];
}

export interface AuditLog {
	id: string;
	userId: string | null;
	action: string;
	resourceType: string | null;
	resourceId: string | null;
	metadata: Record<string, unknown>;
	ipAddress: string | null;
	userAgent: string | null;
	createdAt: string;
}

export interface ApiSuccess<T> {
	data: T;
}

export interface ApiError {
	error: {
		code:    string;
		message: string;
	};
}
