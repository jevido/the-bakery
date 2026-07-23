import { relations } from 'drizzle-orm';
import {
	pgTable,
	text,
	timestamp,
	uuid,
	real,
	integer,
	bigint,
	boolean,
	jsonb,
	index,
	unique,
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
	deployments: many(deployment),
	commands: many(hostCommand),
	volumes: many(volume)
}));

export const hostMetricSampleRelations = relations(hostMetricSample, ({ one }) => ({
	host: one(host, {
		fields: [hostMetricSample.hostId],
		references: [host.id]
	})
}));

/**
 * Per-container CPU/memory sample, extending Phase 02's host-level
 * `hostMetricSample` with per-app data (task 01). One row per reported
 * container per check-in — matched back to an `app` via its currently
 * `running` deployment's unit name (`runningUnitName` in proxy.ts), not
 * stored directly, same "recompute, don't store" convention as
 * `podmanVolumeName`.
 */
export const appMetricSample = pgTable(
	'app_metric_sample',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		appId: uuid('app_id')
			.notNull()
			.references(() => app.id, { onDelete: 'cascade' }),
		hostId: uuid('host_id')
			.notNull()
			.references(() => host.id, { onDelete: 'cascade' }),
		ts: timestamp('ts').defaultNow().notNull(),
		cpuPct: real('cpu_pct'),
		memBytes: bigint('mem_bytes', { mode: 'number' })
	},
	(table) => [index('appMetricSample_appId_ts_idx').on(table.appId, table.ts)]
);

export const appMetricSampleRelations = relations(appMetricSample, ({ one }) => ({
	app: one(app, {
		fields: [appMetricSample.appId],
		references: [app.id]
	}),
	host: one(host, {
		fields: [appMetricSample.hostId],
		references: [host.id]
	})
}));

/**
 * Real app runtime log lines (task 05), replacing the app detail page's
 * static `LOG_LINES` mock — populated by the agent tailing each managed
 * container's `podman logs` output and reported alongside metrics on
 * check-in. `level` is left null in v1 (no log-level parsing) — the raw
 * line is what's shown.
 */
export const appLogLine = pgTable(
	'app_log_line',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		appId: uuid('app_id')
			.notNull()
			.references(() => app.id, { onDelete: 'cascade' }),
		hostId: uuid('host_id')
			.notNull()
			.references(() => host.id, { onDelete: 'cascade' }),
		ts: timestamp('ts').defaultNow().notNull(),
		level: text('level'),
		message: text('message').notNull()
	},
	(table) => [index('appLogLine_appId_ts_idx').on(table.appId, table.ts)]
);

export const appLogLineRelations = relations(appLogLine, ({ one }) => ({
	app: one(app, {
		fields: [appLogLine.appId],
		references: [app.id]
	}),
	host: one(host, {
		fields: [appLogLine.hostId],
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
		// Gates whether a webhook-triggered build (task 07) auto-deploys on
		// success. Manual "Build now" builds never auto-deploy regardless of
		// this flag — only `build.triggeredBy === 'webhook'` does.
		autoDeployEnabled: boolean('auto_deploy_enabled').default(true).notNull(),
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
		// Nullable — a build normally always has a repo (webhook/manual "Build
		// now", both always set this), except the one exception created
		// directly by the bootstrap endpoint (Phase 08 task 06) for the
		// control plane's own self-deploy, which points `imageRef` straight at
		// an already-published image with no source repo involved at all.
		repoId: uuid('repo_id').references(() => repo.id, { onDelete: 'cascade' }),
		// Doubles as the deploy pipeline's version identifier (orchestrator.ts's
		// `versionedUnitName`), not just a git commit display value, so it
		// stays required even for a repo-less build — the bootstrap endpoint
		// sets it to a generated identifier instead of a real commit sha.
		commitSha: text('commit_sha').notNull(),
		// Unlike commitSha, branch is pure display/clone metadata — never read
		// by the deploy pipeline itself — so it's the one that's genuinely
		// meaningless and left null for a repo-less build.
		branch: text('branch'),
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
	envVars: many(envVar),
	domains: many(domain),
	volumes: many(volume),
	metricSamples: many(appMetricSample),
	logLines: many(appLogLine)
}));

export const domain = pgTable(
	'domain',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		appId: uuid('app_id')
			.notNull()
			.references(() => app.id, { onDelete: 'cascade' }),
		hostname: text('hostname').notNull().unique(),
		isDefaultSubdomain: boolean('is_default_subdomain').default(false).notNull(),
		// Null for default subdomains (task 02), which don't need ownership
		// verification — populated once a custom domain (task 05) proves DNS
		// control.
		verifiedAt: timestamp('verified_at'),
		createdAt: timestamp('created_at').defaultNow().notNull()
	},
	(table) => [index('domain_appId_idx').on(table.appId)]
);

export const domainRelations = relations(domain, ({ one }) => ({
	app: one(app, {
		fields: [domain.appId],
		references: [app.id]
	})
}));

/**
 * Driver simplified to Podman-managed `local` only — no `driver` column, per
 * the "hide behind defaults" pattern task 06 established for networks.
 * `name` is the user-declared logical name (e.g. "data"); the real on-host
 * Podman volume name is always derived from it via `podmanVolumeName()`
 * (quadlet.ts) rather than stored, so two apps can reuse the same logical
 * name without colliding on disk. `sizeBytes`/`lastReportedAt` start empty
 * and are filled in by the agent's check-in report (task 07) once the
 * volume actually exists on the host.
 */
export const volume = pgTable(
	'volume',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		appId: uuid('app_id')
			.notNull()
			.references(() => app.id, { onDelete: 'cascade' }),
		hostId: uuid('host_id')
			.notNull()
			.references(() => host.id, { onDelete: 'cascade' }),
		name: text('name').notNull(),
		mountPath: text('mount_path').notNull(),
		sizeBytes: bigint('size_bytes', { mode: 'number' }).default(0).notNull(),
		createdAt: timestamp('created_at').defaultNow().notNull(),
		lastReportedAt: timestamp('last_reported_at')
	},
	(table) => [
		index('volume_appId_idx').on(table.appId),
		index('volume_hostId_idx').on(table.hostId),
		unique('volume_appId_name_unique').on(table.appId, table.name)
	]
);

export const volumeRelations = relations(volume, ({ one }) => ({
	app: one(app, {
		fields: [volume.appId],
		references: [app.id]
	}),
	host: one(host, {
		fields: [volume.hostId],
		references: [host.id]
	})
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

export const deploymentRelations = relations(deployment, ({ one, many }) => ({
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
	}),
	commands: many(hostCommand)
}));

export const envVarRelations = relations(envVar, ({ one }) => ({
	app: one(app, {
		fields: [envVar.appId],
		references: [app.id]
	})
}));

export const hostCommandType = pgEnum('host_command_type', [
	'deploy',
	'stop',
	'restart',
	'configureProxy'
]);

export const hostCommandStatus = pgEnum('host_command_status', [
	'pending',
	'delivered',
	'succeeded',
	'failed'
]);

export const hostCommand = pgTable(
	'host_command',
	{
		id: uuid('id').primaryKey().defaultRandom(),
		hostId: uuid('host_id')
			.notNull()
			.references(() => host.id, { onDelete: 'cascade' }),
		deploymentId: uuid('deployment_id')
			.notNull()
			.references(() => deployment.id, { onDelete: 'cascade' }),
		type: hostCommandType('type').notNull(),
		// The Quadlet unit + env file content (task 03) for `deploy`, just the
		// unit name for `stop`/`restart`, or the full desired Caddyfile content
		// for `configureProxy` (Phase 05 task 03) — shape isn't enforced at the
		// column level since the agent (task 05) is the one interpreting it.
		payload: jsonb('payload').notNull(),
		status: hostCommandStatus('status').default('pending').notNull(),
		createdAt: timestamp('created_at').defaultNow().notNull(),
		deliveredAt: timestamp('delivered_at'),
		completedAt: timestamp('completed_at'),
		errorMessage: text('error_message')
	},
	(table) => [
		index('hostCommand_hostId_status_idx').on(table.hostId, table.status),
		index('hostCommand_deploymentId_idx').on(table.deploymentId)
	]
);

export const hostCommandRelations = relations(hostCommand, ({ one }) => ({
	host: one(host, {
		fields: [hostCommand.hostId],
		references: [host.id]
	}),
	deployment: one(deployment, {
		fields: [hostCommand.deploymentId],
		references: [deployment.id]
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
