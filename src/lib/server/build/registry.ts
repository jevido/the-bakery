function requireEnv(name: string): string {
	const value = process.env[name];
	if (!value) throw new Error(`${name} is not set`);
	return value;
}

/** `registry.<bakery-domain>/<organizationId>/<appId>:<commitSha>`, per task 08. */
export function registryImageRef(organizationId: string, appId: string, commitSha: string): string {
	const host = requireEnv('BAKERY_REGISTRY_HOST');
	return `${host}/${organizationId}/${appId}:${commitSha}`;
}

/**
 * Push credentials as a `podman push --creds user:pass` arg rather than a
 * persistent `podman login` — several worker instances (or builds within
 * one) can run concurrently, and `podman login` writes to a single shared
 * auth.json, so per-invocation `--creds` avoids them racing on that file.
 */
export function pushCredentials(): string {
	const username = requireEnv('BAKERY_REGISTRY_PUSH_USERNAME');
	const password = requireEnv('BAKERY_REGISTRY_PUSH_PASSWORD');
	return `${username}:${password}`;
}
