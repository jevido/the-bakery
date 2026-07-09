import { createSign } from 'node:crypto';

function base64url(input: Buffer | string): string {
	return (typeof input === 'string' ? Buffer.from(input) : input).toString('base64url');
}

function privateKey(): string {
	const encoded = process.env.GITHUB_APP_PRIVATE_KEY;
	if (!encoded) throw new Error('GITHUB_APP_PRIVATE_KEY is not set');
	return Buffer.from(encoded, 'base64').toString('utf-8');
}

/**
 * Short-lived JWT identifying the GitHub App itself (not an installation) —
 * see https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-json-web-token-jwt-for-a-github-app.
 * Signed with node:crypto directly rather than pulling in a JWT dependency,
 * since RS256 over two base64url segments is all this needs.
 */
export function createAppJwt(): string {
	const appId = process.env.GITHUB_APP_ID;
	if (!appId) throw new Error('GITHUB_APP_ID is not set');

	const now = Math.floor(Date.now() / 1000);
	const encodedHeader = base64url(JSON.stringify({ alg: 'RS256', typ: 'JWT' }));
	// iat backdated 60s to tolerate clock drift with GitHub's servers, per their docs.
	const encodedPayload = base64url(JSON.stringify({ iat: now - 60, exp: now + 9 * 60, iss: appId }));
	const signingInput = `${encodedHeader}.${encodedPayload}`;

	const signature = createSign('RSA-SHA256').update(signingInput).sign(privateKey());
	return `${signingInput}.${base64url(signature)}`;
}

interface InstallationTokenResponse {
	token: string;
}

/** Exchanges the App JWT for a ~1hr access token scoped to one installation. */
export async function getInstallationAccessToken(installationId: string): Promise<string> {
	const res = await fetch(`https://api.github.com/app/installations/${installationId}/access_tokens`, {
		method: 'POST',
		headers: {
			Authorization: `Bearer ${createAppJwt()}`,
			Accept: 'application/vnd.github+json',
			'X-GitHub-Api-Version': '2022-11-28'
		}
	});
	if (!res.ok) {
		throw new Error(`Failed to get installation access token: ${res.status} ${await res.text()}`);
	}
	const data = (await res.json()) as InstallationTokenResponse;
	return data.token;
}

export interface GithubRepo {
	full_name: string;
	default_branch: string;
}

/** Every repo the installation can see, paginated per GitHub's `per_page`/`page` convention. */
export async function listInstallationRepositories(installationId: string): Promise<GithubRepo[]> {
	const token = await getInstallationAccessToken(installationId);
	const repos: GithubRepo[] = [];

	for (let page = 1; ; page++) {
		const res = await fetch(
			`https://api.github.com/installation/repositories?per_page=100&page=${page}`,
			{
				headers: {
					Authorization: `Bearer ${token}`,
					Accept: 'application/vnd.github+json',
					'X-GitHub-Api-Version': '2022-11-28'
				}
			}
		);
		if (!res.ok) {
			throw new Error(`Failed to list installation repositories: ${res.status} ${await res.text()}`);
		}
		const data = (await res.json()) as { repositories: GithubRepo[] };
		repos.push(...data.repositories);
		if (data.repositories.length < 100) break;
	}

	return repos;
}
