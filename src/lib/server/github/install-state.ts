import { createHmac, timingSafeEqual } from 'node:crypto';

const MAX_STATE_AGE_MS = 10 * 60 * 1000;

interface InstallStatePayload {
	organizationId: string;
	issuedAt: number;
}

function secret(): string {
	const value = process.env.GITHUB_WEBHOOK_SECRET;
	if (!value) throw new Error('GITHUB_WEBHOOK_SECRET is not set');
	return value;
}

function sign(encodedPayload: string): string {
	return createHmac('sha256', secret()).update(encodedPayload).digest('base64url');
}

/**
 * HMAC-signed `state` param for the GitHub App install redirect, so the
 * callback (task 03) can recover which guild initiated the install and
 * reject a tampered/forged `organizationId` rather than trusting it blindly.
 */
export function createGithubInstallState(organizationId: string): string {
	const encodedPayload = Buffer.from(
		JSON.stringify({ organizationId, issuedAt: Date.now() } satisfies InstallStatePayload)
	).toString('base64url');
	return `${encodedPayload}.${sign(encodedPayload)}`;
}

export function verifyGithubInstallState(state: string): { organizationId: string } | null {
	const [encodedPayload, signature] = state.split('.');
	if (!encodedPayload || !signature) return null;

	const expected = Buffer.from(sign(encodedPayload));
	const actual = Buffer.from(signature);
	if (expected.length !== actual.length || !timingSafeEqual(expected, actual)) return null;

	let payload: InstallStatePayload;
	try {
		payload = JSON.parse(Buffer.from(encodedPayload, 'base64url').toString('utf-8'));
	} catch {
		return null;
	}
	if (typeof payload.organizationId !== 'string' || typeof payload.issuedAt !== 'number') return null;
	if (Date.now() - payload.issuedAt > MAX_STATE_AGE_MS) return null;

	return { organizationId: payload.organizationId };
}
