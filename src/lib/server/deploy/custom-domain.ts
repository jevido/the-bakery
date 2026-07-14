import { resolveCname } from 'node:dns/promises';

/**
 * Confirms a customer's custom domain has a CNAME record pointing at the
 * app's default subdomain (task 02). Any DNS failure — NXDOMAIN, no CNAME
 * record at all, an A record instead, propagation still in flight — is
 * treated the same way as "not verified yet"; there's no useful distinction
 * to surface beyond "check again once DNS has propagated".
 */
export async function verifyCnameTarget(
	hostname: string,
	expectedTarget: string
): Promise<boolean> {
	try {
		const records = await resolveCname(hostname);
		const expected = normalizeHostname(expectedTarget);
		return records.some((record) => normalizeHostname(record) === expected);
	} catch {
		return false;
	}
}

function normalizeHostname(hostname: string): string {
	return hostname.trim().toLowerCase().replace(/\.$/, '');
}
