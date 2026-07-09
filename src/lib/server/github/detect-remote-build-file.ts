import { getInstallationAccessToken } from './app-auth';
import { CANDIDATE_FILENAMES, type DetectedBuildFile } from '$lib/server/build/detect-build-file';

/**
 * Same Dockerfile-then-Containerfile check as `detectBuildFile`, but against
 * GitHub's Contents API instead of a local clone — used for the app-creation
 * form's live preview (task 10), which runs before any clone happens.
 */
export async function detectRemoteBuildFile(
	installationId: string,
	fullName: string,
	branch: string,
	buildContext: string
): Promise<DetectedBuildFile | null> {
	const token = await getInstallationAccessToken(installationId);

	for (const filename of CANDIDATE_FILENAMES) {
		const path =
			buildContext === '.' || buildContext === '' ? filename : `${buildContext}/${filename}`;
		const res = await fetch(
			`https://api.github.com/repos/${fullName}/contents/${path}?ref=${encodeURIComponent(branch)}`,
			{
				headers: {
					Authorization: `Bearer ${token}`,
					Accept: 'application/vnd.github+json',
					'X-GitHub-Api-Version': '2022-11-28'
				}
			}
		);
		if (res.ok) return { path, filename };
	}
	return null;
}
