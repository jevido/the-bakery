import { relations } from 'drizzle-orm';
import {
	pgTable,
	text,
	timestamp,
	uuid,
	real,
	integer,
	boolean,
	index,
	pgEnum
} from 'drizzle-orm/pg-core';
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
	metricSamples: many(hostMetricSample),
	deployments: many(deployment)
}));

export const hostMetricSampleRelations = relations(hostMetricSample, ({ one }) => ({
	host: one(host, {
		fields: [hostMetricSample.hostId],
		references: [host.id]
	})
}));

export const sourceProvider = pgEnum('source_provider', ['github_app']);

export const source = pgTable(
	'source',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		organizationId: text('organization_id')
			.notNull()
			.references(() => organization.id, { onDelete: 'cascade' }),
		provider: sourceProvider('provider').default('github_app').notNull(),
		githubInstallationId: text('github_installation_id'),
		connectedAt: timestamp('connected_at').defaultNow().notNull()
	},
	(table) => [index('source_organizationId_idx').on(table.organizationId)]
);

export const repo = pgTable(
	'repo',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		sourceId: uuid('source_id')
			.notNull()
			.references(() => source.id, { onDelete: 'cascade' }),
		fullName: text('full_name').notNull(),
		defaultBranch: text('default_branch').notNull()
	},
	(table) => [index('repo_sourceId_idx').on(table.sourceId)]
);

export const app = pgTable(
	'app',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		organizationId: text('organization_id')
			.notNull()
			.references(() => organization.id, { onDelete: 'cascade' }),
		name: text('name').notNull(),
		repoId: uuid('repo_id').references(() => repo.id, { onDelete: 'set null' }),
		buildContext: text('build_context').default('.').notNull(),
		dockerfilePath: text('dockerfile_path'),
		hostId: uuid('host_id').references(() => host.id, { onDelete: 'set null' }),
		createdAt: timestamp('created_at').defaultNow().notNull()
	},
	(table) => [index('app_organizationId_idx').on(table.organizationId)]
);

export const buildStatus = pgEnum('build_status', ['queued', 'building', 'succeeded', 'failed']);

export const build = pgTable(
	'build',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		appId: uuid('app_id')
			.notNull()
			.references(() => app.id, { onDelete: 'cascade' }),
		repoId: uuid('repo_id')
			.notNull()
			.references(() => repo.id, { onDelete: 'cascade' }),
		commitSha: text('commit_sha').notNull(),
		branch: text('branch').notNull(),
		// A user id (see auth.schema `user`), or the literal 'webhook' — not an FK
		// since it's polymorphic between the two.
		triggeredBy: text('triggered_by'),
		status: buildStatus('status').default('queued').notNull(),
		imageRef: text('image_ref'),
		startedAt: timestamp('started_at'),
		finishedAt: timestamp('finished_at')
	},
	(table) => [index('build_appId_idx').on(table.appId), index('build_repoId_idx').on(table.repoId)]
);

export const buildLogLine = pgTable(
	'build_log_line',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		buildId: uuid('build_id')
			.notNull()
			.references(() => build.id, { onDelete: 'cascade' }),
		ts: timestamp('ts').defaultNow().notNull(),
		line: text('line').notNull()
	},
	(table) => [index('buildLogLine_buildId_ts_idx').on(table.buildId, table.ts)]
);

export const sourceRelations = relations(source, ({ one, many }) => ({
	organization: one(organization, {
		fields: [source.organizationId],
		references: [organization.id]
	}),
	repos: many(repo)
}));

export const repoRelations = relations(repo, ({ one, many }) => ({
	source: one(source, {
		fields: [repo.sourceId],
		references: [source.id]
	}),
	apps: many(app),
	builds: many(build)
}));

export const appRelations = relations(app, ({ one, many }) => ({
	organization: one(organization, {
		fields: [app.organizationId],
		references: [organization.id]
	}),
	repo: one(repo, {
		fields: [app.repoId],
		references: [repo.id]
	}),
	host: one(host, {
		fields: [app.hostId],
		references: [host.id]
	}),
	builds: many(build),
	deployments: many(deployment),
	envVars: many(envVar)
}));

export const buildRelations = relations(build, ({ one, many }) => ({
	app: one(app, {
		fields: [build.appId],
		references: [app.id]
	}),
	repo: one(repo, {
		fields: [build.repoId],
		references: [repo.id]
	}),
	logLines: many(buildLogLine),
	deployments: many(deployment)
}));

export const buildLogLineRelations = relations(buildLogLine, ({ one }) => ({
	build: one(build, {
		fields: [buildLogLine.buildId],
		references: [build.id]
	})
}));

export const deploymentStatus = pgEnum('deployment_status', [
	'starting_new',
	'health_checking',
	'flipping_proxy',
	'stopping_old',
	'running',
	'failed',
	'rolled_back'
]);

export const deployment = pgTable(
	'deployment',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		appId: uuid('app_id')
			.notNull()
			.references(() => app.id, { onDelete: 'cascade' }),
		buildId: uuid('build_id')
			.notNull()
			.references(() => build.id, { onDelete: 'cascade' }),
		hostId: uuid('host_id').references(() => host.id, { onDelete: 'set null' }),
		status: deploymentStatus('status').default('starting_new').notNull(),
		startedAt: timestamp('started_at').defaultNow().notNull(),
		finishedAt: timestamp('finished_at'),
		// A user id (see auth.schema `user`), or the literal 'webhook' — not an FK
		// since it's polymorphic between the two.
		triggeredBy: text('triggered_by')
	},
	(table) => [
		index('deployment_appId_idx').on(table.appId),
		index('deployment_buildId_idx').on(table.buildId),
		index('deployment_hostId_idx').on(table.hostId)
	]
);

export const envVar = pgTable(
	'env_var',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		appId: uuid('app_id')
			.notNull()
			.references(() => app.id, { onDelete: 'cascade' }),
		key: text('key').notNull(),
		valueCiphertext: text('value_ciphertext').notNull(),
		isSecret: boolean('is_secret').default(false).notNull(),
		updatedAt: timestamp('updated_at').defaultNow().notNull()
	},
	(table) => [index('envVar_appId_idx').on(table.appId)]
);

export const deploymentRelations = relations(deployment, ({ one }) => ({
	app: one(app, {
		fields: [deployment.appId],
		references: [app.id]
	}),
	build: one(build, {
		fields: [deployment.buildId],
		references: [build.id]
	}),
	host: one(host, {
		fields: [deployment.hostId],
		references: [host.id]
	})
}));

export const envVarRelations = relations(envVar, ({ one }) => ({
	app: one(app, {
		fields: [envVar.appId],
		references: [app.id]
	})
}));

/**
 * Manually seeded for now — no CI/release pipeline yet (Phase 02 task 12).
 * The Go agent polls GET /api/v1/agent/version?platform=... and self-updates
 * when the latest row's version differs from its own compiled-in version.
 */
export const agentRelease = pgTable(
	'agent_release',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		version: text('version').notNull(),
		platform: text('platform').notNull(),
		downloadUrl: text('download_url').notNull(),
		sha256: text('sha256').notNull(),
		releasedAt: timestamp('released_at').defaultNow().notNull()
	},
	(table) => [index('agentRelease_platform_releasedAt_idx').on(table.platform, table.releasedAt)]
);
