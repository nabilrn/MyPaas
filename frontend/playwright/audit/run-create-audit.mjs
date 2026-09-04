import { runAudit } from './create-project-audit.mjs';

const modeArg = process.argv.find((arg) => arg.startsWith('--mode='));
const mode = modeArg?.split('=')[1] || 'mock';

if (!['mock', 'production'].includes(mode)) {
	console.error(`Unsupported mode: ${mode}`);
	process.exit(1);
}

if (!process.env.MYPAAS_AUDIT_BASE_URL) {
	if (mode === 'production') {
		console.error('MYPAAS_AUDIT_BASE_URL is required for production UI audits.');
		process.exit(1);
	}
	process.env.MYPAAS_AUDIT_BASE_URL = 'http://127.0.0.1:4173';
}

await runAudit({ mode });
