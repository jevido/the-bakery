import { z } from 'zod';

/**
 * Shared by the check-in endpoint and any future agent-facing endpoints
 * (Phase 04's command dispatch). Changing this shape means updating
 * already-deployed agent binaries, not just this codebase.
 */
export const checkinPayloadSchema = z.object({
	cpuPct: z.number().min(0).max(100),
	memPct: z.number().min(0).max(100),
	diskPct: z.number().min(0).max(100),
	podmanVersion: z.string().trim().min(1).max(50),
	containerCount: z.number().int().min(0),
	agentVersion: z.string().trim().min(1).max(50)
});

export type CheckinPayload = z.infer<typeof checkinPayloadSchema>;

export interface CheckinResponse {
	pendingCommands: unknown[];
}
