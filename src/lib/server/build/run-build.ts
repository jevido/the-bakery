import { spawn } from 'node:child_process';
import { createInterface } from 'node:readline';
import { mkdtemp, rm, access } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { build, repo, app, source } from '$lib/server/db/schema';
import { getInstallationAccessToken } from '$lib/server/github/app-auth';
import { appendBuildLog } from './log';
import { detectBuildFile, candidateBuildFilePaths } from './detect-build-file';
import type { Build } from './claim';

class BuildFailure extends Error {}

/** Runs `command` to completion, streaming each stdout/stderr line into `buildLogLine`. */
function runStreamed(
	buildId: string,
	command: string,
	args: string[],
	cwd?: string
): Promise<void> {
	return new Promise((resolve, reject) => {
		const child = spawn(command, args, { cwd });
		let settled = false;

		const pipe = (stream: NodeJS.ReadableStream) => {
			createInterface({ input: stream }).on('line', (line) => {
				void appendBuildLog(buildId, line);
			});
		};
		pipe(child.stdout);
		pipe(child.stderr);

		child.on('error', (err) => {
			if (settled) return;
			settled = true;
			reject(err);
		});
		child.on('close', (code) => {
			if (settled) return;
			settled = true;
			if (code === 0) resolve();
			else reject(new BuildFailure(`${command} exited with code ${code}`));
		});
	});
}

/**
 * Clones the repo, locates a Dockerfile/Containerfile, runs `podman build`,
 * and updates the `build` row's status as it goes. Never throws — a failure
 * anywhere in the pipeline is captured as a log line and a `failed` status
 * so the worker loop can move on to the next build.
 */
export async function processBuild(buildRow: Build): Promise<void> {
	let workDir: string | null = null;
	try {
		const [repoRow] = await db.select().from(repo).where(eq(repo.id, buildRow.repoId));
		if (!repoRow) throw new BuildFailure(`repo ${buildRow.repoId} not found`);

		const [appRow] = await db.select().from(app).where(eq(app.id, buildRow.appId));
		if (!appRow) throw new BuildFailure(`app ${buildRow.appId} not found`);

		const [sourceRow] = await db.select().from(source).where(eq(source.id, repoRow.sourceId));
		if (!sourceRow?.githubInstallationId) {
			throw new BuildFailure(`source for repo ${repoRow.fullName} has no GitHub installation`);
		}

		workDir = await mkdtemp(join(tmpdir(), 'bakery-build-'));
		const cloneDir = join(workDir, 'repo');

		await appendBuildLog(buildRow.id, `Cloning ${repoRow.fullName}@${buildRow.commitSha}...`);
		const token = await getInstallationAccessToken(sourceRow.githubInstallationId);
		const authedUrl = `https://x-access-token:${token}@github.com/${repoRow.fullName}.git`;
		await runStreamed(buildRow.id, 'git', [
			'clone',
			'--branch',
			buildRow.branch,
			authedUrl,
			cloneDir
		]);
		await runStreamed(buildRow.id, 'git', ['checkout', buildRow.commitSha], cloneDir);

		// `buildContextDir`, not `cloneDir`, becomes podman's build context
		// below, so relative COPY/ADD paths in the Dockerfile resolve against
		// the subdirectory rather than the repo root.
		const buildContextDir = join(cloneDir, appRow.buildContext);
		let dockerfile: string;
		if (appRow.dockerfilePath) {
			dockerfile = join(buildContextDir, appRow.dockerfilePath);
			await access(dockerfile).catch(() => {
				throw new BuildFailure(
					`Configured dockerfilePath "${appRow.dockerfilePath}" not found under ${appRow.buildContext}`
				);
			});
		} else {
			const detected = await detectBuildFile(buildContextDir);
			if (!detected) {
				const checked = candidateBuildFilePaths(buildContextDir).join(', ');
				throw new BuildFailure(
					`No Dockerfile or Containerfile found in ${appRow.buildContext} (checked ${checked})`
				);
			}
			dockerfile = detected.path;
			await appendBuildLog(buildRow.id, `Detected ${detected.filename} at ${detected.path}`);
		}

		// Tagged locally for now — task 08 (registry) re-tags and pushes this
		// to Bakery's registry, then overwrites `imageRef` with the pushed ref.
		const imageRef = `localhost/bakery/${appRow.id}:${buildRow.commitSha.slice(0, 12)}`;
		await appendBuildLog(buildRow.id, `Building ${dockerfile} -> ${imageRef}...`);
		await runStreamed(buildRow.id, 'podman', [
			'build',
			'-f',
			dockerfile,
			'-t',
			imageRef,
			buildContextDir
		]);

		await db
			.update(build)
			.set({ status: 'succeeded', imageRef, finishedAt: new Date() })
			.where(eq(build.id, buildRow.id));
		await appendBuildLog(buildRow.id, 'Build succeeded.');
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		await appendBuildLog(buildRow.id, `Build failed: ${message}`);
		await db
			.update(build)
			.set({ status: 'failed', finishedAt: new Date() })
			.where(eq(build.id, buildRow.id));
	} finally {
		if (workDir) await rm(workDir, { recursive: true, force: true });
	}
}
