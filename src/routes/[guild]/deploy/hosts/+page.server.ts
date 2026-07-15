import { fail } from '@sveltejs/kit';
import { superValidate } from 'sveltekit-superforms';
import { zod } from '$lib/forms/zod4-adapter';
import { z } from 'zod';
import { and, desc, eq, inArray } from 'drizzle-orm';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host, hostMetricSample } from '$lib/server/db/schema';
import { issueHostToken } from '$lib/server/hosts/tokens';
import { computeHostStatus } from '$lib/server/hosts/status';

const addHostSchema = z.object({
	name: z.string().trim().min(1, 'Host name is required').max(100),
	location: z.string().trim().max(200).optional(),
	spec: z.string().trim().max(200).optional()
});

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	// Revoked hosts stay visible (with their historical metric samples
	// retained — deletion is Phase 06's retention concern) so revocation
	// reads as a status, not a disappearance.
	//
	// Explicit column list, not `select()` — `tokenHash` must never reach the
	// client (Phase 07 task 05's encryption-at-rest review): this page's data
	// is serialized straight to the browser, so an unqualified `select()`
	// would ship it in every load even though nothing renders it.
	const hostRows = await db
		.select({
			id: host.id,
			organizationId: host.organizationId,
			name: host.name,
			location: host.location,
			spec: host.spec,
			tokenLastFour: host.tokenLastFour,
			status: host.status,
			agentVersion: host.agentVersion,
			createdAt: host.createdAt,
			lastSeenAt: host.lastSeenAt,
			revokedAt: host.revokedAt
		})
		.from(host)
		.where(eq(host.organizationId, organization.id));

	const hostIds = hostRows.map((h) => h.id);
	const samples = hostIds.length
		? await db
				.select()
				.from(hostMetricSample)
				.where(inArray(hostMetricSample.hostId, hostIds))
				.orderBy(desc(hostMetricSample.ts))
		: [];

	// First row per hostId wins — samples are already ordered newest-first.
	const latestSampleByHost = new Map<string, (typeof samples)[number]>();
	for (const sample of samples) {
		if (!latestSampleByHost.has(sample.hostId)) latestSampleByHost.set(sample.hostId, sample);
	}

	const hosts = hostRows.map((h) => {
		const computedStatus = computeHostStatus(h);
		return {
			...h,
			computedStatus,
			online: computedStatus === 'online',
			latestSample: latestSampleByHost.get(h.id) ?? null
		};
	});

	const form = await superValidate(zod(addHostSchema));

	return { hosts, form };
};

export const actions: Actions = {
	addHost: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_hosts' });
		const form = await superValidate(event, zod(addHostSchema));
		if (!form.valid) return fail(400, { form });

		const issued = issueHostToken();

		// Explicit column list, not `.returning()` — `created` below is sent
		// straight back to the client as the action response, and `tokenHash`
		// must never reach it (Phase 07 task 05's encryption-at-rest review).
		// The plaintext token is the deliberate one-time exception, carried
		// separately in `installCommand`, never stored anywhere itself.
		const [created] = await db
			.insert(host)
			.values({
				organizationId: organization.id,
				name: form.data.name,
				location: form.data.location || null,
				spec: form.data.spec || null,
				tokenHash: issued.hash,
				tokenLastFour: issued.lastFour,
				status: 'pending'
			})
			.returning({
				id: host.id,
				organizationId: host.organizationId,
				name: host.name,
				location: host.location,
				spec: host.spec,
				tokenLastFour: host.tokenLastFour,
				status: host.status,
				agentVersion: host.agentVersion,
				createdAt: host.createdAt,
				lastSeenAt: host.lastSeenAt,
				revokedAt: host.revokedAt
			});

		// bakery-domain: prefer the configured public origin, fall back to the
		// request's own origin so this works before ORIGIN is set in dev.
		const bakeryDomain = process.env.ORIGIN ?? event.url.origin;
		const installCommand = `curl -fsSL ${bakeryDomain}/install.sh | sh -s -- --token=${issued.plaintext} --url=${bakeryDomain}`;

		return { form, host: created, installCommand };
	},

	revokeHost: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_hosts' });

		const formData = await event.request.formData();
		const hostId = formData.get('hostId');
		if (typeof hostId !== 'string' || !hostId) {
			return fail(400, { message: 'Missing host id' });
		}

		const [existing] = await db
			.select({ id: host.id, revokedAt: host.revokedAt })
			.from(host)
			.where(and(eq(host.id, hostId), eq(host.organizationId, organization.id)));
		if (!existing) {
			return fail(404, { message: 'Host not found' });
		}
		if (existing.revokedAt) {
			return { revoked: true };
		}

		// Clear tokenHash too (belt-and-suspenders alongside the revokedAt
		// check the check-in endpoint already enforces): an empty string can
		// never equal a real token's SHA-256 hex digest, so even if that
		// check were ever removed, a revoked host's old token can't match.
		await db.update(host).set({ revokedAt: new Date(), tokenHash: '' }).where(eq(host.id, hostId));

		return { revoked: true };
	}
};
