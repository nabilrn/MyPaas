// ─── Domain enums ────────────────────────────────────────────────────────────

export type ProjectStatus = 'pending' | 'running' | 'stopped' | 'crashed' | 'building';
export type DeployMode    = 'dockerfile' | 'compose' | 'static';
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
	id:                  string;
	userId:              string;
	name:                string;
	repoUrl:             string;
	branch:              string;
	subdomain:           string;
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
	createdAt:           string;
	updatedAt:           string;
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

export interface EnvVarDiscovery {
	key: string;
	source: string;
	sensitive: boolean;
	defaultValue?: string;
}

export interface ContainerMetrics {
	service:        string;
	cpu:            number;   // percent
	memoryMb:       number;
	memoryLimitMb:  number;
	uptime:         string;   // e.g. "2h 14m"
}

export interface MetricsSnapshot {
	items:       ContainerMetrics[];
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

export interface DeployModeDetection {
	deployMode: DeployMode;
	branch: string;
	mainService: string | null;
	services: string[];
	composeFile: string | null;
	hasDockerfile: boolean;
	envVars: EnvVarDiscovery[];
	appPort: number;
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

// ─── API response wrappers ────────────────────────────────────────────────────

export interface ApiSuccess<T> {
	data: T;
}

export interface ApiError {
	error: {
		code:    string;
		message: string;
	};
}
