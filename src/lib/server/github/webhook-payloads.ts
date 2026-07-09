import { z } from 'zod';

/** Only the fields the webhook receiver actually reads — GitHub's real payloads carry much more. */
export const pushEventSchema = z.object({
	ref: z.string(),
	after: z.string(),
	repository: z.object({
		full_name: z.string()
	})
});

export const installationEventSchema = z.object({
	action: z.string(),
	installation: z.object({ id: z.number() })
});

export const installationRepositoriesEventSchema = z.object({
	action: z.string(),
	installation: z.object({ id: z.number() }),
	repositories_added: z.array(z.object({ full_name: z.string() })).optional(),
	repositories_removed: z.array(z.object({ full_name: z.string() })).optional()
});
