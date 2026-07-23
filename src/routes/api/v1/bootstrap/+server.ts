import { json, error } from '@sveltejs/kit';
import { randomBytes, timingSafeEqual } from 'node:crypto';
import { z } from 'zod';
import type { RequestHandler } from './$types';
import { auth } from '$lib/server/auth';
import { APIError } from 'better-auth/api';
import { db } from '$lib/server/db';
import { user, organization, host, app, build, domain } from '$lib/server/db/schema';
import { issueHostToken } from '$lib/server/hosts/tokens';

const bootstrapSchema = z.object({
	email: z.string().trim().email(),
	password: z.string().min(8),
	name: z.string().trim().min(1).max(100).default('Admin'),
	guildName: z.string().trim().min(1).max(100).default('Bakery'),
	// The control plane's own public hostname (e.g. bakery.example.com) —
	// trusted and marked verified immediately: unlike a normal guild's
	// custom-domain flow (CNAME-verified against another tenant's app on a
	// shared instance), there's no other tenant to protect against here —
	// this domain resolving to this exact box is the precondition for
	// `bakery bootstrap` to have been run against it at all.
	domain: z.string().trim().min(1).max(253)
});

function slugify(name: string): string {
	return (
		name
			.toLowerCase()
			.replace(/\s+/g, '-')
			.replace(/[^a-z0-9-]/g, '') || 'guild'
	);
}

/**
 * One-time, unauthenticated bootstrap: drives the exact same signup/guild/
 * host/app business logic the UI does, so a brand-new self-hosted instance
 * can create its first admin, guild, host token, and self-deploying app
 * record without a browser (`bakery bootstrap`, Phase 08 task 07). Reuses
 * Better Auth's server API and the existing token-issuance util directly —
 * never reimplements them — per the confirmed "no duplicated business
 * logic" direction for this phase.
 *
 * Self-disabling: refuses to run at all once any user or guild exists, so
 * this endpoint has no effect on an instance that's already live, no matter
 * how long it stays reachable after bootstrap. `bakery bootstrap` also only
 * ever calls this against a loopback-only control plane, so the real-world
 * exposure window is already narrow — BAKERY_BOOTSTRAP_SECRET below is
 * defense-in-depth on top of that, not the only thing standing between this
 * endpoint and the internet.
 */
export const POST: RequestHandler = async (event) => {
	// Optional second gate, per this task's own Notes: if `bakery bootstrap`
	// (task 07) generated a one-time secret and passed it to the container
	// as BAKERY_BOOTSTRAP_SECRET, require it here too — protects against a
	// slow/partial install leaving this reachable longer than the "loopback
	// only" window is supposed to last. Absent entirely if unset, same as
	// every other optional instance-level env var in this codebase.
	const bootstrapSecret = process.env.BAKERY_BOOTSTRAP_SECRET;
	if (bootstrapSecret) {
		const provided = event.request.headers.get('x-bootstrap-secret') ?? '';
		const expected = Buffer.from(bootstrapSecret);
		const actual = Buffer.from(provided);
		if (expected.length !== actual.length || !timingSafeEqual(expected, actual)) {
			error(403, 'Invalid or missing bootstrap secret');
		}
	}

	const [existingUser] = await db.select({ id: user.id }).from(user).limit(1);
	const [existingOrg] = await db.select({ id: organization.id }).from(organization).limit(1);
	if (existingUser || existingOrg) {
		error(403, 'Bootstrap already completed — this instance already has a user or guild');
	}

	const selfImage = process.env.BAKERY_SELF_IMAGE;
	if (!selfImage) {
		error(
			500,
			'BAKERY_SELF_IMAGE is not set — this image was not published by CI, so bootstrap has no image to deploy'
		);
	}

	const body = await event.request.json().catch(() => null);
	const parsed = bootstrapSchema.safeParse(body);
	if (!parsed.success) {
		error(400, parsed.error.issues[0]?.message ?? 'Invalid bootstrap payload');
	}
	const { email, password, name, guildName, domain: controlPlaneDomain } = parsed.data;

	let signUpResult;
	try {
		signUpResult = await auth.api.signUpEmail({
			headers: event.request.headers,
			body: { email, password, name }
		});
	} catch (e) {
		if (e instanceof APIError) error(400, e.message || 'Signup failed');
		throw e;
	}
	const userId = signUpResult.user.id;

	// createOrganization must be called *without* session headers to create
	// the guild on behalf of a specific `userId` — passing headers here
	// would make Better Auth silently ignore `userId` and look for a session
	// user that doesn't exist yet (see Better Auth org plugin docs, "For
	// Admins").
	const slug = slugify(guildName);
	let org;
	try {
		org = await auth.api.createOrganization({
			body: {
				name: guildName,
				slug,
				color: '#22c55e',
				inviteCode: `${slug}-${Math.floor(Math.random() * 90 + 10)}`,
				userId
			}
		});
	} catch (e) {
		if (e instanceof APIError) error(400, e.message || 'Guild creation failed');
		throw e;
	}
	if (!org) error(500, 'Guild creation returned no organization');

	const issued = issueHostToken();
	const [hostRow] = await db
		.insert(host)
		.values({
			organizationId: org.id,
			name: 'control-plane',
			tokenHash: issued.hash,
			tokenLastFour: issued.lastFour,
			status: 'pending'
		})
		.returning({ id: host.id });

	const [appRow] = await db
		.insert(app)
		.values({
			organizationId: org.id,
			name: 'bakery',
			hostId: hostRow.id
		})
		.returning({ id: app.id });

	// No repo/source — this is the one image-pointing build the schema
	// supports directly (bakery.schema.ts's build.repoId, task 06). Already
	// `succeeded` with `imageRef` set: nothing needs to actually build this,
	// it's already sitting in the registry this same instance was published
	// from. commitSha still gets a real value (used by orchestrator.ts's
	// versionedUnitName for the deploy unit name), branch stays null (no
	// repo, nothing to display).
	await db.insert(build).values({
		appId: appRow.id,
		repoId: null,
		commitSha: randomBytes(4).toString('hex'),
		branch: null,
		triggeredBy: 'bootstrap',
		status: 'succeeded',
		imageRef: selfImage,
		startedAt: new Date(),
		finishedAt: new Date()
	});

	await db.insert(domain).values({
		appId: appRow.id,
		hostname: controlPlaneDomain,
		isDefaultSubdomain: false,
		verifiedAt: new Date()
	});

	return json({
		hostToken: issued.plaintext,
		organizationSlug: org.slug,
		appId: appRow.id,
		hostId: hostRow.id
	});
};
