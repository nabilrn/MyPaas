import { runAudit } from './create-project-audit.mjs';

const modeArg = process.argv.find((arg) => arg.startsWith('--mode='));
const mode = modeArg?.split('=')[1] || 'mock';

if (!['mock', 'production'].includes(mode)) {
	console.error(`Unsupported mode: ${mode}`);
	process.exit(1);
}

await runAudit({ mode });
