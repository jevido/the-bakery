import { relations } from 'drizzle-orm';
import { pgTable, text, timestamp, uuid, real, integer, index, pgEnum } from 'drizzle-orm/pg-core';
import { organization } from './auth.schema';

export const hostStatus = pgEnum('host_status', ['pending', 'online', 'offline']);

export const host = pgTable(
	'host',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		organizationId: text('organization_id')
			.notNull()
			.references(() => organization.id, { onDelete: 'cascade' }),
		name: text('name').notNull(),
		location: text('location'),
		spec: text('spec'),
		tokenHash: text('token_hash').notNull(),
		tokenLastFour: text('token_last_four'),
		status: hostStatus('status').default('pending').notNull(),
		agentVersion: text('agent_version'),
		createdAt: timestamp('created_at').defaultNow().notNull(),
		lastSeenAt: timestamp('last_seen_at'),
		revokedAt: timestamp('revoked_at')
	},
	(table) => [index('host_organizationId_idx').on(table.organizationId)]
);

export const hostMetricSample = pgTable(
	'host_metric_sample',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		hostId: uuid('host_id')
			.notNull()
			.references(() => host.id, { onDelete: 'cascade' }),
		ts: timestamp('ts').defaultNow().notNull(),
		cpuPct: real('cpu_pct'),
		memPct: real('mem_pct'),
		diskPct: real('disk_pct'),
		podmanVersion: text('podman_version'),
		containerCount: integer('container_count')
	},
	(table) => [index('hostMetricSample_hostId_ts_idx').on(table.hostId, table.ts)]
);

export const hostRelations = relations(host, ({ one, many }) => ({
	organization: one(organization, {
		fields: [host.organizationId],
		references: [organization.id]
	}),
	metricSamples: many(hostMetricSample)
}));

export const hostMetricSampleRelations = relations(hostMetricSample, ({ one }) => ({
	host: one(host, {
		fields: [hostMetricSample.hostId],
		references: [host.id]
	})
}));
