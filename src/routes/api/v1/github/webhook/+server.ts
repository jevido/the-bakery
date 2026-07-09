import { error, json } from '@sveltejs/kit';
import { createHmac, timingSafeEqual } from 'node:crypto';
import { and, eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { source, repo, app, build } from '$lib/server/db/schema';
import {
	pushEventSchema,
	installationEventSchema,
	installationRepositoriesEventSchema
} from '$lib/server/github/webhook-payloads';

function verifySignature(rawBody: string, signatureHeader: string | null): boolean {
	if (!signatureHeader?.startsWith('sha256=')) return false;
	const secret = process.env.GITHUB_WEBHOOK_SECRET;
	if (!secret) return false;

	const expected = `sha256=${createHmac('sha256', secret).update(rawBody).digest('hex')}`;
	const expectedBuf = Buffer.from(expected);
	const actualBuf = Buffer.from(signatureHeader);
	return expectedBuf.length === actualBuf.length && timingSafeEqual(expectedBuf, actualBuf);
}

async function handlePush(rawPayload: unknown) {
	const parsed = pushEventSchema.safeParse(rawPayload);
	if (!parsed.success) return;
	const { ref, after: commitSha, repository } = parsed.data;

	const branch = ref.replace(/^refs\/heads\//, '');

	const [repoRow] = await db.select().from(repo).where(eq(repo.fullName, repository.full_name));
	// No app has a branch of its own to configure (task 02's schema scopes
	// that to the repo) — a push only triggers a build when it lands on the
	// repo's configured default branch.
	if (!repoRow || repoRow.defaultBranch !== branch) return;

	const matchingApps = await db.select().from(app).where(eq(app.repoId, repoRow.id));
	if (matchingApps.length === 0) return;

	await db.insert(build).values(
		matchingApps.map((a) => ({
			appId: a.id,
			repoId: repoRow.id,
			commitSha,
			branch,
			triggeredBy: 'webhook',
			status: 'queued' as const
		}))
	);
}

async function handleInstallation(rawPayload: unknown) {
	const parsed = installationEventSchema.safeParse(rawPayload);
	if (!parsed.success) return;
	const installationId = String(parsed.data.installation.id);

	if (parsed.data.action === 'deleted') {
		await db.delete(source).where(eq(source.githubInstallationId, installationId));
	}
}

async function handleInstallationRepositories(rawPayload: unknown) {
	const parsed = installationRepositoriesEventSchema.safeParse(rawPayload);
	if (!parsed.success) return;
	const { action, installation, repositories_added, repositories_removed } = parsed.data;

	const [sourceRow] = await db
		.select()
		.from(source)
		.where(eq(source.githubInstallationId, String(installation.id)));
	if (!sourceRow) return;

	if (action === 'added') {
		for (const r of repositories_added ?? []) {
			const [existing] = await db
				.select({ id: repo.id })
				.from(repo)
				.where(and(eq(repo.sourceId, sourceRow.id), eq(repo.fullName, r.full_name)));
			if (!existing) {
				// installation_repositories payloads don't carry default_branch —
				// task 05's real repo listing (via the GitHub API) corrects this.
				await db.insert(repo).values({ sourceId: sourceRow.id, fullName: r.full_name, defaultBranch: 'main' });
			}
		}
	}

	if (action === 'removed') {
		for (const r of repositories_removed ?? []) {
			await db.delete(repo).where(and(eq(repo.sourceId, sourceRow.id), eq(repo.fullName, r.full_name)));
		}
	}
}

export const POST: RequestHandler = async (event) => {
	const rawBody = await event.request.text();
	const signature = event.request.headers.get('x-hub-signature-256');

	if (!verifySignature(rawBody, signature)) {
		error(401, 'Invalid signature');
	}

	const githubEvent = event.request.headers.get('x-github-event');
	const payload: unknown = JSON.parse(rawBody);

	if (githubEvent === 'push') {
		await handlePush(payload);
	} else if (githubEvent === 'installation') {
		await handleInstallation(payload);
	} else if (githubEvent === 'installation_repositories') {
		await handleInstallationRepositories(payload);
	}

	return json({ ok: true });
};
