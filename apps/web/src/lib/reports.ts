// Reporting a member or a guild to the operators.

import { api } from './api'

export type ReportTarget = 'member' | 'guild'

export const reports = {
  file: (targetKind: ReportTarget, targetId: number, reason: string) =>
    api<{ report: { id: number } }>('POST', '/api/reports', { target_kind: targetKind, target_id: targetId, reason }),
}
