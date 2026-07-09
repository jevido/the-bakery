/**
 * Generates registry-auth/htpasswd for the Bakery-hosted registry (task 08).
 * Docker's registry:2 htpasswd auth handler only accepts bcrypt hashes, and
 * there's no `htpasswd` CLI dependency in this repo — Bun's built-in
 * `Bun.password` already does bcrypt natively, so this is simpler than
 * pulling in an external tool. Run with `bun scripts/generate-registry-htpasswd.ts`
 * after setting BAKERY_REGISTRY_*_USERNAME/PASSWORD in .env.
 */
import { writeFile, mkdir } from 'node:fs/promises';

interface Credential {
	label: string;
	usernameEnv: string;
	passwordEnv: string;
}

const CREDENTIALS: Credential[] = [
	{
		label: 'push (build worker)',
		usernameEnv: 'BAKERY_REGISTRY_PUSH_USERNAME',
		passwordEnv: 'BAKERY_REGISTRY_PUSH_PASSWORD'
	},
	{
		label: 'pull (host agents)',
		usernameEnv: 'BAKERY_REGISTRY_PULL_USERNAME',
		passwordEnv: 'BAKERY_REGISTRY_PULL_PASSWORD'
	}
];

async function main() {
	const lines: string[] = [];
	for (const { label, usernameEnv, passwordEnv } of CREDENTIALS) {
		const username = process.env[usernameEnv];
		const password = process.env[passwordEnv];
		if (!username || !password) {
			throw new Error(`${usernameEnv} and ${passwordEnv} must be set (missing for ${label})`);
		}
		const hash = await Bun.password.hash(password, { algorithm: 'bcrypt', cost: 10 });
		lines.push(`${username}:${hash}`);
	}

	await mkdir('registry-auth', { recursive: true });
	await writeFile('registry-auth/htpasswd', lines.join('\n') + '\n');
	console.log(`Wrote registry-auth/htpasswd with ${lines.length} user(s).`);
}

main().catch((err) => {
	console.error(err.message);
	process.exit(1);
});
