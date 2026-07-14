import { z } from 'zod';

/**
 * One Bakery-managed Podman volume's on-host size, as reported by the agent
 * each check-in (task 07). `name` is the real Podman volume name
 * (`podmanVolumeName()` in quadlet.ts), not the user-declared logical name —
 * the checkin endpoint matches reports back to `volume` rows by recomputing
 * that name per row rather than trusting the agent to know the logical one.
 */
export const volumeReportSchema = z.object({
	name: z.string().trim().min(1).max(255),
	sizeBytes: z.number().int().min(0)
});

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
	agentVersion: z.string().trim().min(1).max(50),
	volumes: z.array(volumeReportSchema).max(500).default([])
});

export type CheckinPayload = z.infer<typeof checkinPayloadSchema>;

/**
 * `payload` shape is by convention, not validated here — task 05's agent
 * executor is the only consumer and interprets it per `type`:
 *   deploy:  { unitName, unitContent, envFileContent, healthCheckPort, networkName, volumes: { name, mountPath }[] }
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
