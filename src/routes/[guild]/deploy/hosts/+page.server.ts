import { fail } from '@sveltejs/kit';
import { superValidate } from 'sveltekit-superforms';
import { zod } from '$lib/forms/zod4-adapter';
import { z } from 'zod';
import { and, desc, eq, inArray, isNull } from 'drizzle-orm';
import type { Actions, PageServerLoad } from './$types';
import { requireGuild } from '$lib/server/guild-context';
import { db } from '$lib/server/db';
import { host, hostMetricSample } from '$lib/server/db/schema';
import { issueHostToken } from '$lib/server/hosts/tokens';

const addHostSchema = z.object({
	name: z.string().trim().min(1, 'Host name is required').max(100),
	location: z.string().trim().max(200).optional(),
	spec: z.string().trim().max(200).optional()
});

// No background sweeper exists yet, so "offline" is computed at read time
// from lastSeenAt rather than trusting the stored `status` column, which is
// only ever updated by the check-in endpoint (task 06) and would otherwise
// go stale the moment an agent stops checking in. Formalized into a shared
// helper in task 07 — kept local to this load for now.
const CHECKIN_INTERVAL_MS = 60_000;
const OFFLINE_THRESHOLD_MS = CHECKIN_INTERVAL_MS * 3;

function isEffectivelyOnline(h: { status: string; lastSeenAt: Date | null }): boolean {
	if (h.status !== 'online' || !h.lastSeenAt) return false;
	return Date.now() - h.lastSeenAt.getTime() < OFFLINE_THRESHOLD_MS;
}

export const load: PageServerLoad = async (event) => {
	const { organization } = await requireGuild(event, { permission: 'view_hosts' });

	const hostRows = await db
		.select()
		.from(host)
		.where(and(eq(host.organizationId, organization.id), isNull(host.revokedAt)));

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

	const hosts = hostRows.map((h) => ({
		...h,
		online: isEffectivelyOnline(h),
		latestSample: latestSampleByHost.get(h.id) ?? null
	}));

	const form = await superValidate(zod(addHostSchema));

	return { hosts, form };
};

export const actions: Actions = {
	addHost: async (event) => {
		const { organization } = await requireGuild(event, { permission: 'manage_hosts' });
		const form = await superValidate(event, zod(addHostSchema));
		if (!form.valid) return fail(400, { form });

		const issued = issueHostToken();

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
			.returning();

		// bakery-domain: prefer the configured public origin, fall back to the
		// request's own origin so this works before ORIGIN is set in dev.
		const bakeryDomain = process.env.ORIGIN ?? event.url.origin;
		const installCommand = `curl -fsSL ${bakeryDomain}/install.sh | sh -s -- --token=${issued.plaintext} --url=${bakeryDomain}`;

		return { form, host: created, installCommand };
	}
};
