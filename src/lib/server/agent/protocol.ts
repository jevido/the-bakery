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

/**
 * `payload` shape is by convention, not validated here — task 05's agent
 * executor is the only consumer and interprets it per `type`:
 *   deploy:  { unitName, unitContent, envFileContent, healthCheckPort, networkName }
 *   stop/restart: { unitName }
 *   configureProxy: { caddyfileContent } (Phase 05 task 03)
 */
export interface PendingCommand {
	id: string;
	type: 'deploy' | 'stop' | 'restart' | 'configureProxy';
	payload: unknown;
}

export interface CheckinResponse {
	pendingCommands: PendingCommand[];
}

/** Body for `POST /api/v1/agent/commands/[id]/complete` (task 04). */
export const commandCompletionSchema = z.object({
	status: z.enum(['succeeded', 'failed']),
	errorMessage: z.string().trim().max(2000).optional()
});

export type CommandCompletionPayload = z.infer<typeof commandCompletionSchema>;
