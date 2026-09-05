import { json } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { readFile } from 'node:fs/promises';
import type { RequestHandler } from './$types';

type UpdateStatus = {
	state: 'idle' | 'checking' | 'updating' | 'succeeded' | 'failed' | 'rolled_back' | 'blocked';
	channel: 'release' | 'main' | 'unknown';
	currentSha: string;
	targetSha: string;
	targetVersion: string;
	message: string;
	updatedAt: string;
};

type ReleaseInfo = {
	tagName: string;
	name: string;
	targetSha: string;
	prerelease: boolean;
	publishedAt: string;
	htmlUrl: string;
};

type GitHubRelease = {
	draft?: boolean;
	prerelease?: boolean;
	tag_name?: string;
	name?: string | null;
	target_commitish?: string;
	published_at?: string | null;
	html_url?: string;
};

type GitHubComparison = {
	status?: string;
	ahead_by?: number;
};

const DEFAULT_STATUS: UpdateStatus = {
	state: 'idle',
	channel: 'unknown',
	currentSha: '',
	targetSha: '',
	targetVersion: '',
	message: 'Updater status is not available yet',
	updatedAt: ''
};
const GITHUB_CACHE_TTL_MS = 5 * 60_000;
const GITHUB_HEADERS = {
	Accept: 'application/vnd.github+json',
	'User-Agent': 'MyPaaS-dashboard',
	'X-GitHub-Api-Version': '2022-11-28'
};

let cachedRelease: { key: string; expiresAt: number; value: ReleaseInfo | null } | null = null;
let cachedComparison: { key: string; expiresAt: number; available: boolean } | null = null;

function parseBool(value: string | undefined) {
	return ['1', 'true', 'yes', 'on'].includes((value ?? '').trim().toLowerCase());
}

function parseValues(raw: string) {
	const values = new Map<string, string>();
	for (const line of raw.split(/\r?\n/)) {
		const separator = line.indexOf('=');
		if (separator <= 0 || line.trimStart().startsWith('#')) continue;
		values.set(line.slice(0, separator), line.slice(separator + 1));
	}
	return values;
}

function parseStatus(raw: string): UpdateStatus {
	const values = parseValues(raw);
	const candidate = values.get('state') ?? 'idle';
	const state = ['idle', 'checking', 'updating', 'succeeded', 'failed', 'rolled_back', 'blocked'].includes(candidate)
		? candidate as UpdateStatus['state']
		: 'idle';
	const channelValue = values.get('channel') ?? 'unknown';
	const channel = ['release', 'main'].includes(channelValue) ? channelValue as UpdateStatus['channel'] : 'unknown';
	return {
		state,
		channel,
		currentSha: values.get('current_sha') ?? '',
		targetSha: values.get('target_sha') ?? '',
		targetVersion: values.get('target_version') ?? '',
		message: values.get('message') ?? '',
		updatedAt: values.get('updated_at') ?? ''
	};
}

async function readStatus(): Promise<UpdateStatus> {
	try {
		const raw = await readFile(env.MYPAAS_UPDATE_STATUS_FILE || '/run/mypaas/update/status', 'utf8');
		return parseStatus(raw);
	} catch {
		return { ...DEFAULT_STATUS };
	}
}

async function readPolicy() {
	try {
		const raw = await readFile(env.MYPAAS_UPDATE_POLICY_FILE || '/etc/mypaas/update.env', 'utf8');
		const values = parseValues(raw);
		const candidate = (values.get('AUTO_UPDATE_CHANNEL') || 'release').toLowerCase();
		return {
			channel: candidate === 'main' ? 'main' as const : 'release' as const,
			includePrereleases: parseBool(values.get('AUTO_UPDATE_INCLUDE_PRERELEASES'))
		};
	} catch {
		return { channel: 'release' as const, includePrereleases: false };
	}
}

async function fetchLatestRelease(includePrereleases: boolean): Promise<ReleaseInfo | null> {
	const cacheKey = includePrereleases ? 'all' : 'stable';
	const now = Date.now();
	if (cachedRelease?.key === cacheKey && cachedRelease.expiresAt > now) return cachedRelease.value;

	const response = await fetch('https://api.github.com/repos/nabilrn/MyPaas/releases?per_page=20', {
		headers: GITHUB_HEADERS,
		signal: AbortSignal.timeout(5000)
	});
	if (!response.ok) throw new Error(`GitHub releases returned ${response.status}`);
	const releases = await response.json() as GitHubRelease[];
	const release = releases.find((item) => !item.draft && (includePrereleases || !item.prerelease));
	const value = release?.tag_name ? {
		tagName: release.tag_name,
		name: release.name || release.tag_name,
		targetSha: release.target_commitish || '',
		prerelease: Boolean(release.prerelease),
		publishedAt: release.published_at || '',
		htmlUrl: release.html_url || ''
	} : null;
	cachedRelease = { key: cacheKey, expiresAt: now + GITHUB_CACHE_TTL_MS, value };
	return value;
}

async function releaseIsAhead(currentSha: string, release: ReleaseInfo) {
	if (!/^[0-9a-f]{40}$/.test(currentSha)) return false;
	if (release.targetSha === currentSha) return false;

	const cacheKey = `${currentSha}:${release.tagName}`;
	const now = Date.now();
	if (cachedComparison?.key === cacheKey && cachedComparison.expiresAt > now) return cachedComparison.available;

	const compareUrl = `https://api.github.com/repos/nabilrn/MyPaas/compare/${encodeURIComponent(currentSha)}...${encodeURIComponent(release.tagName)}`;
	const response = await fetch(compareUrl, {
		headers: GITHUB_HEADERS,
		signal: AbortSignal.timeout(5000)
	});
	if (!response.ok) throw new Error(`GitHub comparison returned ${response.status}`);
	const comparison = await response.json() as GitHubComparison;
	const available = comparison.status === 'ahead' && (comparison.ahead_by ?? 0) > 0;
	cachedComparison = { key: cacheKey, expiresAt: now + GITHUB_CACHE_TTL_MS, available };
	return available;
}

async function apiRequest(path: string, cookie: string) {
	const base = (env.INTERNAL_API_URL || 'http://api:8080').replace(/\/$/, '');
	return fetch(`${base}${path}`, {
		headers: { cookie, Accept: 'application/json' },
		cache: 'no-store',
		signal: AbortSignal.timeout(5000)
	});
}

export const GET: RequestHandler = async ({ request }) => {
	const cookie = request.headers.get('cookie') || '';
	if (!cookie) return json({ error: 'authentication required' }, { status: 401 });

	let me: Response;
	try {
		me = await apiRequest('/auth/me', cookie);
	} catch {
		return json({ error: 'authentication service unavailable' }, { status: 503 });
	}
	if (!me.ok) return json({ error: 'authentication required' }, { status: me.status === 403 ? 403 : 401 });
	const user = await me.json() as { role?: string };
	if (user.role !== 'owner') return json({ error: 'owner access required' }, { status: 403 });

	const [status, policy] = await Promise.all([readStatus(), readPolicy()]);
	let currentSha = status.currentSha;
	try {
		const settingsResponse = await apiRequest('/settings', cookie);
		if (settingsResponse.ok) {
			const settings = await settingsResponse.json() as { build_sha?: string };
			currentSha = settings.build_sha || currentSha;
		}
	} catch {
		// Status remains useful after the API returns from an update restart.
	}

	const channel = policy.channel || status.channel || 'release';
	let release: ReleaseInfo | null = null;
	let releaseAvailable = false;
	if (channel === 'release') {
		try {
			release = await fetchLatestRelease(policy.includePrereleases);
			if (release) releaseAvailable = await releaseIsAhead(currentSha, release);
		} catch {
			// Fail closed: keep release actions disabled when GitHub cannot prove ancestry.
		}
	}

	return json({
		status: { ...status, channel, currentSha },
		release: release ? { ...release, available: releaseAvailable } : null
	}, {
		headers: { 'Cache-Control': 'no-store' }
	});
};
