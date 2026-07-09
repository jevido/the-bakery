import { error, redirect } from '@sveltejs/kit';
import { eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { organization, source } from '$lib/server/db/schema';
import { verifyGithubInstallState } from '$lib/server/github/install-state';
import { syncReposForSource } from '$lib/server/github/sync-repos';

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
	let [sourceRow] = await db
		.select({ id: source.id })
		.from(source)
		.where(eq(source.githubInstallationId, installationId));

	if (!sourceRow) {
		[sourceRow] = await db
			.insert(source)
			.values({
				organizationId: verified.organizationId,
				provider: 'github_app',
				githubInstallationId: installationId
			})
			.returning({ id: source.id });
	}

	// Best-effort: sync now so the sources page has real repos on first
	// visit instead of requiring a manual refresh. A failure here (GitHub
	// API hiccup, etc.) shouldn't block the redirect — the sources page's
	// "Refresh" action (task 05) covers retrying.
	await syncReposForSource(sourceRow.id, installationId).catch(() => null);

	redirect(302, `/${org.slug}/deploy/sources?github_connected=1`);
};
