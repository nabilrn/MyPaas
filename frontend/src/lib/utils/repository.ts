export type RepositoryHost = 'github' | 'gitlab' | 'git' | 'registry';

export interface RepositoryDisplay {
	host: RepositoryHost;
	/** Short human label, e.g. "nabilrn/MyPaas" or "grafana/grafana:10.4.2". */
	label: string;
	/** Clickable external URL for git sources; null for registry images. */
	href: string | null;
}

const SSH_PREFIX = /^git@([^:]+):/;

/**
 * Present a project source as a compact, scannable repository reference.
 * Never exposes the raw clone URL in the UI; registry projects show the image ref.
 */
export function describeProjectSource(project: {
	sourceType: string;
	repoUrl: string;
	imageRef: string | null;
}): RepositoryDisplay {
	if (project.sourceType === 'registry') {
		return { host: 'registry', label: project.imageRef ?? 'container image', href: null };
	}
	return describeRepoUrl(project.repoUrl);
}

export function describeRepoUrl(repoUrl: string): RepositoryDisplay {
	const trimmed = repoUrl.trim();
	if (!trimmed) return { host: 'git', label: 'repository', href: null };

	// Normalize SCP-style SSH (git@host:owner/repo.git) to a URL-like shape.
	const sshMatch = trimmed.match(SSH_PREFIX);
	const normalized = sshMatch ? `ssh://${sshMatch[1]}/${trimmed.slice(sshMatch[0].length)}` : trimmed;

	let url: URL | null = null;
	try {
		url = new URL(normalized);
	} catch {
		url = null;
	}
	if (!url) return { host: 'git', label: trimmed, href: null };

	const hostname = url.hostname.toLowerCase();
	const path = url.pathname.replace(/^\/+|\/+$/g, '').replace(/\.git$/i, '');
	const label = path || hostname;
	const httpsHref = path ? `https://${hostname}/${path}` : `https://${hostname}`;

	if (hostname === 'github.com') return { host: 'github', label, href: httpsHref };
	if (hostname === 'gitlab.com') return { host: 'gitlab', label, href: httpsHref };
	return { host: 'git', label: path ? `${hostname}/${path}` : hostname, href: httpsHref };
}
