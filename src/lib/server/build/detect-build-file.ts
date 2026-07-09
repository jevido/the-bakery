import { access } from 'node:fs/promises';
import { join } from 'node:path';

export interface DetectedBuildFile {
	path: string;
	filename: 'Dockerfile' | 'Containerfile';
}

/**
 * Exported so the remote (pre-clone, GitHub Contents API) detector used by
 * the app-creation UI preview (task 10) checks the same two filenames in
 * the same order as this local (post-clone) build-time check, rather than
 * maintaining a second hardcoded list that could drift.
 */
export const CANDIDATE_FILENAMES = ['Dockerfile', 'Containerfile'] as const;

/**
 * Checks `buildContextDir` for a `Dockerfile`, then a `Containerfile`
 * (podman's native name) — this order and the two candidate filenames are
 * shared between the build worker's pre-build check (this module, called
 * from run-build.ts) and the app-creation UI's detection preview (task 10),
 * so the two never drift out of sync.
 */
export async function detectBuildFile(buildContextDir: string): Promise<DetectedBuildFile | null> {
	for (const filename of CANDIDATE_FILENAMES) {
		const path = join(buildContextDir, filename);
		try {
			await access(path);
			return { path, filename };
		} catch {
			// try the next candidate
		}
	}
	return null;
}

/** The paths `detectBuildFile` checks, for use in "not found" error messages. */
export function candidateBuildFilePaths(buildContextDir: string): string[] {
	return CANDIDATE_FILENAMES.map((filename) => join(buildContextDir, filename));
}
