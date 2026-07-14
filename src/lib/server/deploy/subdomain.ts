/**
 * Bare hostname suffix apps are provisioned under, e.g. `bakery.example.com`
 * — distinct from `ORIGIN` (a full URL with scheme, used for redirects and
 * the install command), since this gets concatenated into subdomains.
 * Falls back to a `.localhost` suffix so default-subdomain rows can still be
 * created in local dev without a real domain configured.
 */
export function bakeryDomain(): string {
	return process.env.BAKERY_DOMAIN ?? 'bakery.localhost';
}

/**
 * DNS labels can't contain arbitrary characters (spaces, underscores,
 * uppercase is merely non-canonical) the way `app.name` or a guild slug are
 * otherwise allowed to — this is the one place that distinction actually
 * matters, so it's normalized here rather than by tightening those fields'
 * validation everywhere else they're used.
 */
function toHostnameLabel(value: string): string {
	return value
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '');
}

/**
 * `<app>.<guild-slug>.<bakery-domain>` — the default subdomain every app
 * gets on creation (task 02). The guild slug is what keeps same-named apps
 * in different guilds from colliding on the same hostname.
 */
export function defaultSubdomain(appName: string, guildSlug: string): string {
	return `${toHostnameLabel(appName)}.${toHostnameLabel(guildSlug)}.${bakeryDomain()}`;
}
