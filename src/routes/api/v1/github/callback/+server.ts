import { error, redirect } from '@sveltejs/kit';
import { eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { organization, source } from '$lib/server/db/schema';
import { verifyGithubInstallState } from '$lib/server/github/install-state';

export const GET: RequestHandler = async (event) => {
	const state = event.url.searchParams.get('state');
	const installationId = event.url.searchParams.get('installation_id');

	const verified = state ? verifyGithubInstallState(state) : null;
	if (!verified) {
		// No signed state to trust means we also don't know which guild to
		// send the user back to — surface an error rather than guessing.
		error(400, "Invalid or expired install request — please retry 'Connect GitHub' from your guild's Sources page.");
	}

	const [org] = await db
		.select({ slug: organization.slug })
		.from(organization)
		.where(eq(organization.id, verified.organizationId));
	if (!org) {
		error(404, 'Guild not found for this GitHub installation.');
	}

	if (!installationId) {
		redirect(302, `/${org.slug}/deploy/sources?github_error=missing_installation_id`);
	}

	// Re-installs/updates redeliver this same callback with the same
	// installation_id — keep it idempotent instead of inserting duplicates.
	const [existing] = await db
		.select({ id: source.id })
		.from(source)
		.where(eq(source.githubInstallationId, installationId));

	if (!existing) {
		await db.insert(source).values({
			organizationId: verified.organizationId,
			provider: 'github_app',
			githubInstallationId: installationId
		});
	}

	redirect(302, `/${org.slug}/deploy/sources?github_connected=1`);
};
