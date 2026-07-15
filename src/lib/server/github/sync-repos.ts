import { eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { repo } from '$lib/server/db/schema';
import { listInstallationRepositories } from './app-auth';

/**
 * Upserts (by `fullName`, scoped to this `sourceId`) every repo the GitHub
 * App installation can currently see, so `app` creation can reference a
 * stable local `repo` row instead of re-querying GitHub every time.
 */
export async function syncReposForSource(
	sourceId: string,
	installationId: string
): Promise<number> {
	const remoteRepos = await listInstallationRepositories(installationId);

	const existingRows = await db.select().from(repo).where(eq(repo.sourceId, sourceId));
	const existingByFullName = new Map(existingRows.map((r) => [r.fullName, r]));

	for (const remote of remoteRepos) {
		const existing = existingByFullName.get(remote.full_name);
		if (!existing) {
			await db.insert(repo).values({
				sourceId,
				fullName: remote.full_name,
				defaultBranch: remote.default_branch
			});
		} else if (existing.defaultBranch !== remote.default_branch) {
			await db
				.update(repo)
				.set({ defaultBranch: remote.default_branch })
				.where(eq(repo.id, existing.id));
		}
	}

	return remoteRepos.length;
}
