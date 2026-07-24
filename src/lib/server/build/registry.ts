function requireEnv(name: string): string {
	const value = process.env[name];
	if (!value) throw new Error(`${name} is not set`);
	return value;
}

/**
 * `registry.<bakery-domain>/<organizationId>/<appId>:<commitSha>`, per task 08.
 *
 * `organizationId` is lowercased — Better Auth's default id generator draws
 * from `a-zA-Z0-9`, but every OCI registry requires an all-lowercase
 * repository path (found live: a build failed with podman's "repository
 * name must be lowercase" once an org happened to get an id with uppercase
 * characters in it). `appId` needs no equivalent handling: it's a Postgres
 * `gen_random_uuid()` value, which is always lowercase hex by definition.
 */
export function registryImageRef(organizationId: string, appId: string, commitSha: string): string {
	const host = requireEnv('BAKERY_REGISTRY_HOST');
	return `${host}/${organizationId.toLowerCase()}/${appId}:${commitSha}`;
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
