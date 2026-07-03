export type AppStatus = 'running' | 'building' | 'failed' | 'stopped';

export interface App {
	id: string;
	name: string;
	type: string;
	status: AppStatus;
	host: string;
	domain: string;
	cpu: number;
	mem: string;
	deployed: string;
	port: string;
	initial: string;
	unit: string;
	quadletPath: string;
}

export interface Host {
	name: string;
	location: string;
	spec: string;
	apps: number;
	podman: string;
	cpu: number;
	mem: number;
	disk: number;
	online: boolean;
}

export interface Guild {
	id: string;
	name: string;
	letter: string;
	color: string;
	invite: string;
	apps: App[];
	hosts: Host[];
}

export interface Member {
	name: string;
	handle: string;
	role: string;
	initials: string;
	avBg: string;
	avColor: string;
	master?: boolean;
	joinedAt?: string;
}

export interface Role {
	id: string;
	name: string;
	color: string;
	count: number;
	master?: boolean;
	note: string;
}

export interface Perm {
	id: string;
	label: string;
	desc: string;
	danger?: boolean;
}

export interface PermGroup {
	name: string;
	perms: Perm[];
}

export interface Template {
	name: string;
	tag: string;
	initial: string;
	bg: string;
	color: string;
	desc: string;
}

export const GUILDS: Record<string, Guild> = {
	sourdough: {
		id: 'sourdough',
		name: 'Sourdough Labs',
		letter: 'S',
		color: '#3fb984',
		invite: 'warm-rye-42',
		apps: [
			{ id: 'crumb-api', name: 'crumb-api', type: 'Node · API', status: 'running', host: 'oven-01', domain: 'api.sourdough.dev', cpu: 12, mem: '240 MB', deployed: '4m ago', port: '3000', initial: 'C', unit: 'crumb-api', quadletPath: 'deploy/crumb-api.container' },
			{ id: 'loaf-web', name: 'loaf-web', type: 'SvelteKit', status: 'running', host: 'oven-01', domain: 'loaf.sourdough.dev', cpu: 6, mem: '180 MB', deployed: '2h ago', port: '3000', initial: 'L', unit: 'loaf-web', quadletPath: 'deploy/loaf-web.container' },
			{ id: 'yeast-worker', name: 'yeast-worker', type: 'Rust · Worker', status: 'building', host: 'oven-02', domain: '— internal —', cpu: 0, mem: '—', deployed: 'now', port: '—', initial: 'Y', unit: 'yeast-worker', quadletPath: 'deploy/yeast-worker.container' },
			{ id: 'proofing-db', name: 'proofing-db', type: 'PostgreSQL 16', status: 'running', host: 'oven-02', domain: '— internal —', cpu: 22, mem: '1.1 GB', deployed: '3d ago', port: '5432', initial: 'P', unit: 'proofing-db', quadletPath: 'deploy/proofing-db.container' },
			{ id: 'starter-bot', name: 'starter-bot', type: 'Python · Bot', status: 'failed', host: 'oven-01', domain: '— internal —', cpu: 0, mem: '—', deployed: '18m ago', port: '—', initial: 'S', unit: 'starter-bot', quadletPath: 'deploy/starter-bot.container' },
			{ id: 'crust-cache', name: 'crust-cache', type: 'Redis 7', status: 'running', host: 'oven-02', domain: '— internal —', cpu: 3, mem: '64 MB', deployed: '1w ago', port: '6379', initial: 'R', unit: 'crust-cache', quadletPath: 'deploy/crust-cache.container' },
		],
		hosts: [
			{ name: 'oven-01', location: 'Hetzner CPX21 · Falkenstein', spec: 'AMD EPYC · 3 vCPU · 4 GB', apps: 3, podman: '5.1', cpu: 41, mem: 62, disk: 55, online: true },
			{ name: 'oven-02', location: 'Homelab · Intel NUC i7', spec: 'Intel · 8 vCPU · 32 GB', apps: 3, podman: '5.1', cpu: 28, mem: 34, disk: 40, online: true },
		],
	},
	northwind: {
		id: 'northwind',
		name: 'Northwind Co',
		letter: 'N',
		color: '#5b8def',
		invite: 'nw-prod-7',
		apps: [
			{ id: 'nw-gateway', name: 'nw-gateway', type: 'Go · API', status: 'running', host: 'prod-eu-1', domain: 'api.northwind.co', cpu: 18, mem: '320 MB', deployed: '1h ago', port: '8080', initial: 'G', unit: 'nw-gateway', quadletPath: 'infra/gateway.container' },
			{ id: 'nw-dashboard', name: 'nw-dashboard', type: 'Next.js', status: 'running', host: 'prod-eu-1', domain: 'app.northwind.co', cpu: 9, mem: '280 MB', deployed: '5h ago', port: '3000', initial: 'D', unit: 'nw-dashboard', quadletPath: 'infra/dashboard.container' },
			{ id: 'nw-billing', name: 'nw-billing', type: 'Node · Worker', status: 'running', host: 'prod-eu-2', domain: '— internal —', cpu: 5, mem: '150 MB', deployed: '2d ago', port: '—', initial: 'B', unit: 'nw-billing', quadletPath: 'infra/billing.container' },
			{ id: 'nw-staging', name: 'nw-staging', type: 'SvelteKit', status: 'stopped', host: 'staging-1', domain: 'staging.northwind.co', cpu: 0, mem: '—', deployed: '6d ago', port: '3000', initial: 'S', unit: 'nw-staging', quadletPath: 'infra/staging.container' },
		],
		hosts: [
			{ name: 'prod-eu-1', location: 'AWS · eu-central-1', spec: 'Graviton · 8 vCPU · 16 GB', apps: 2, podman: '5.1', cpu: 52, mem: 61, disk: 44, online: true },
			{ name: 'prod-eu-2', location: 'AWS · eu-central-1', spec: 'Graviton · 4 vCPU · 8 GB', apps: 1, podman: '5.1', cpu: 22, mem: 30, disk: 28, online: true },
			{ name: 'staging-1', location: 'Hetzner · Nuremberg', spec: 'AMD · 2 vCPU · 4 GB', apps: 1, podman: '5.0', cpu: 4, mem: 12, disk: 33, online: true },
		],
	},
};

export const GUILD_LIST = Object.values(GUILDS);

export const MEMBERS: Record<string, Member[]> = {
	sourdough: [
		{ name: 'Rye', handle: '@rye', role: 'Guild Master', initials: 'RY', avBg: 'linear-gradient(140deg,#d98a4a,#b5632c)', avColor: '#1a0f07', master: true, joinedAt: 'Jan 2024' },
		{ name: 'Poppy', handle: '@poppy', role: 'Head Baker', initials: 'PO', avBg: 'linear-gradient(140deg,#b98ce6,#8b5fd0)', avColor: '#150a24', joinedAt: 'Feb 2024' },
		{ name: 'Sesame', handle: '@sesame', role: 'Head Baker', initials: 'SE', avBg: 'linear-gradient(140deg,#7aa6f5,#4f79d8)', avColor: '#0a1226', joinedAt: 'Feb 2024' },
		{ name: 'Basil', handle: '@basil', role: 'Baker', initials: 'BA', avBg: 'linear-gradient(140deg,#52cc96,#2f9e6c)', avColor: '#07130c', joinedAt: 'Mar 2024' },
		{ name: 'Olive', handle: '@olive', role: 'Baker', initials: 'OL', avBg: 'linear-gradient(140deg,#c7c34a,#9a962f)', avColor: '#161503', joinedAt: 'Apr 2024' },
		{ name: 'Wheat', handle: '@wheat', role: 'Apprentice', initials: 'WH', avBg: 'linear-gradient(140deg,#8a9aa5,#5f6f7a)', avColor: '#0a1013', joinedAt: 'May 2024' },
	],
	northwind: [
		{ name: 'Rye', handle: '@rye', role: 'Baker', initials: 'RY', avBg: 'linear-gradient(140deg,#d98a4a,#b5632c)', avColor: '#1a0f07', joinedAt: 'Mar 2024' },
		{ name: 'Marta', handle: '@marta', role: 'Guild Master', initials: 'MA', avBg: 'linear-gradient(140deg,#7aa6f5,#4f79d8)', avColor: '#0a1226', master: true, joinedAt: 'Jan 2024' },
		{ name: 'Devs', handle: '@devteam', role: 'Head Baker', initials: 'DV', avBg: 'linear-gradient(140deg,#b98ce6,#8b5fd0)', avColor: '#150a24', joinedAt: 'Jan 2024' },
	],
};

export const ROLES: Role[] = [
	{ id: 'guild-master', name: 'Guild Master', color: '#e0a83e', count: 1, master: true, note: 'The founder of the guild. Sits above every position and cannot be edited or deleted.' },
	{ id: 'head-baker', name: 'Head Baker', color: '#a978e6', count: 2, note: 'Trusted maintainers. Manage apps, hosts, and all positions beneath them.' },
	{ id: 'baker', name: 'Baker', color: '#3fb984', count: 3, note: 'Developers. Deploy apps and manage environments, but cannot touch roles or hosts.' },
	{ id: 'apprentice', name: 'Apprentice', color: '#5b8def', count: 4, note: 'Read-only access. Perfect for stakeholders watching the ovens.' },
];

export const PERM_GROUPS: PermGroup[] = [
	{ name: 'GENERAL', perms: [
		{ id: 'view_guild', label: 'View Guild', desc: 'See the guild and its modules' },
		{ id: 'manage_guild', label: 'Manage Guild', desc: 'Edit name, invites, and settings' },
		{ id: 'manage_roles', label: 'Manage Roles', desc: 'Create and edit positions below your own' },
		{ id: 'manage_members', label: 'Manage Members', desc: 'Assign roles, kick, and ban' },
		{ id: 'audit_log', label: 'View Audit Log', desc: 'Review every action taken in the guild' },
	]},
	{ name: 'APPS', perms: [
		{ id: 'view_apps', label: 'View Apps', desc: 'See apps and their status' },
		{ id: 'create_apps', label: 'Create Apps', desc: 'Deploy new apps from a repo' },
		{ id: 'deploy_apps', label: 'Deploy & Redeploy', desc: 'Trigger deployments and rollbacks' },
		{ id: 'manage_env', label: 'Manage Environment', desc: 'Edit variables and secrets' },
		{ id: 'view_secrets', label: 'View Secrets', desc: 'Reveal masked secret values' },
		{ id: 'delete_apps', label: 'Delete Apps', desc: 'Permanently remove an app' },
	]},
	{ name: 'INFRASTRUCTURE', perms: [
		{ id: 'view_hosts', label: 'View Hosts', desc: 'See connected machines' },
		{ id: 'manage_hosts', label: 'Manage Hosts', desc: 'Add, remove, and configure hosts' },
		{ id: 'manage_domains', label: 'Manage Domains & Proxy', desc: 'Edit domains and Caddy routes' },
	]},
	{ name: 'DANGER', perms: [
		{ id: 'administrator', label: 'Administrator', desc: 'All permissions. Grant with caution.', danger: true },
	]},
];

export const ROLE_PERMS: Record<string, string[]> = {
	'guild-master': ['administrator'],
	'head-baker': ['view_guild', 'manage_roles', 'manage_members', 'audit_log', 'view_apps', 'create_apps', 'deploy_apps', 'manage_env', 'view_secrets', 'delete_apps', 'view_hosts', 'manage_hosts', 'manage_domains'],
	'baker': ['view_guild', 'view_apps', 'create_apps', 'deploy_apps', 'manage_env', 'view_hosts', 'manage_domains'],
	'apprentice': ['view_guild', 'view_apps', 'view_hosts'],
};

export const TEMPLATES: Template[] = [
	{ name: 'PostgreSQL', tag: 'postgres:16', initial: 'P', bg: 'rgba(91,141,239,.14)', color: '#7aa6f5', desc: 'Persistent Postgres with a named volume and health check.' },
	{ name: 'Redis', tag: 'redis:7', initial: 'R', bg: 'rgba(229,101,75,.14)', color: '#f0836b', desc: 'In-memory cache, internal-only by default.' },
	{ name: 'Caddy', tag: 'caddy:2', initial: 'C', bg: 'rgba(63,185,132,.14)', color: '#52cc96', desc: 'Reverse proxy with automatic HTTPS for a host.' },
	{ name: 'Node service', tag: 'node:22', initial: 'N', bg: 'rgba(224,168,62,.14)', color: '#efc060', desc: 'Generic Node app with EnvironmentFile wiring.' },
	{ name: 'Umami', tag: 'analytics', initial: 'U', bg: 'rgba(169,120,230,.14)', color: '#c09bf0', desc: 'Privacy-friendly web analytics, batteries included.' },
	{ name: 'MinIO', tag: 's3 · storage', initial: 'M', bg: 'rgba(91,141,239,.14)', color: '#7aa6f5', desc: 'S3-compatible object storage with a console.' },
];

export interface StatusMeta {
	label: string;
	flavor: string;
	c: string;
	bg: string;
	dot: string;
	pulse?: boolean;
	spin?: boolean;
}

export function statusMeta(s: AppStatus): StatusMeta {
	const M: Record<AppStatus, StatusMeta> = {
		running:  { label: 'Running',  flavor: 'Fresh',        c: '#52cc96', bg: 'rgba(63,185,132,.14)',  dot: '#3fb984', pulse: true },
		building: { label: 'Building', flavor: 'In the oven',  c: '#efc060', bg: 'rgba(224,168,62,.15)',  dot: '#e0a83e', spin: true },
		failed:   { label: 'Failed',   flavor: 'Burnt',        c: '#f0836b', bg: 'rgba(229,101,75,.15)',  dot: '#e5654b' },
		stopped:  { label: 'Stopped',  flavor: 'Proofing',     c: '#5c6170', bg: 'rgba(255,255,255,.04)', dot: '#5c6170' },
	};
	return M[s] ?? M.stopped;
}

export function quadletContent(app: App): string {
	const port = app.port === '—' ? '3000' : app.port;
	return `[Unit]
Description=${app.name}
After=network-online.target

[Container]
Image=ghcr.io/sourdough-labs/${app.name}:latest
PublishPort=${port}:${port}
EnvironmentFile=/etc/bakery/${app.unit}.env
AutoUpdate=registry

[Service]
Restart=always
TimeoutStartSec=90

[Install]
WantedBy=default.target`;
}

export const APP_CONTAINERS: Record<string, Array<{name: string; image: string; nets: string; cpu: string; mem: string; status: AppStatus}>> = {
	'crumb-api': [
		{ name: 'crumb-api', image: 'ghcr.io/sourdough-labs/crumb-api:latest', nets: 'bakery-net', cpu: '12%', mem: '240 MB', status: 'running' },
		{ name: 'crumb-api-sidecar', image: 'prom/node-exporter:latest', nets: 'bakery-net', cpu: '0%', mem: '18 MB', status: 'running' },
	],
	'loaf-web': [
		{ name: 'loaf-web', image: 'ghcr.io/sourdough-labs/loaf-web:latest', nets: 'bakery-net, web-net', cpu: '6%', mem: '180 MB', status: 'running' },
	],
	'proofing-db': [
		{ name: 'proofing-db', image: 'postgres:16', nets: 'bakery-net', cpu: '22%', mem: '1.1 GB', status: 'running' },
	],
};

export const APP_DEPLOYMENTS: Array<{msg: string; sha: string; branch: string; by: string; status: 'Deployed' | 'Failed'; time: string}> = [
	{ msg: 'fix: retry pool on cold start',        sha: 'a3f9c21', branch: 'main',  by: 'Basil', status: 'Deployed', time: '4m ago' },
	{ msg: 'chore: bump base image to node:22',    sha: '7b1e004', branch: 'main',  by: 'Rye',   status: 'Deployed', time: '2h ago' },
	{ msg: 'feat: add /healthz endpoint',           sha: 'c9d2f88', branch: 'main',  by: 'Poppy', status: 'Deployed', time: 'yesterday' },
	{ msg: 'wip: experimental cache layer',         sha: 'e04a1b3', branch: 'spike', by: 'Basil', status: 'Failed',   time: '2d ago' },
];

export const LOG_LINES: Array<{t: string; lvl: string; color: string; msg: string}> = [
	{ t: '12:04:41', lvl: 'INFO', color: '#52cc96', msg: 'listening on 0.0.0.0:3000' },
	{ t: '12:04:41', lvl: 'INFO', color: '#52cc96', msg: 'connected to proofing-db (pool=10)' },
	{ t: '12:05:02', lvl: 'INFO', color: '#52cc96', msg: 'GET /healthz 200 1ms' },
	{ t: '12:05:18', lvl: 'INFO', color: '#52cc96', msg: 'GET /api/orders 200 24ms' },
	{ t: '12:05:44', lvl: 'WARN', color: '#efc060', msg: 'slow query 812ms — orders.list' },
	{ t: '12:06:01', lvl: 'INFO', color: '#52cc96', msg: 'POST /api/orders 201 38ms' },
	{ t: '12:06:15', lvl: 'INFO', color: '#52cc96', msg: 'GET /api/products 200 9ms' },
	{ t: '12:06:33', lvl: 'ERROR', color: '#f0836b', msg: 'upstream timeout: checkout-service unreachable after 5000ms' },
	{ t: '12:06:34', lvl: 'INFO', color: '#52cc96', msg: 'retrying... attempt 1/3' },
	{ t: '12:06:35', lvl: 'INFO', color: '#52cc96', msg: 'retrying... attempt 2/3' },
	{ t: '12:06:36', lvl: 'INFO', color: '#52cc96', msg: 'recovered — checkout-service responded in 214ms' },
	{ t: '12:06:51', lvl: 'INFO', color: '#52cc96', msg: 'GET /api/orders 200 19ms' },
	{ t: '12:07:03', lvl: 'INFO', color: '#52cc96', msg: 'auto-update: checking ghcr.io/sourdough-labs/crumb-api' },
	{ t: '12:07:04', lvl: 'INFO', color: '#52cc96', msg: 'image up to date — no pull needed' },
];

export const ENV_SETS: Record<string, Array<{key: string; v: string; secret: boolean}>> = {
	production: [
		{ key: 'NODE_ENV',       v: 'production',                                    secret: false },
		{ key: 'PORT',           v: '3000',                                           secret: false },
		{ key: 'DATABASE_URL',   v: 'postgres://crust:s3cr3t@proofing-db:5432/app',  secret: true },
		{ key: 'REDIS_URL',      v: 'redis://crust-cache:6379',                      secret: false },
		{ key: 'SESSION_SECRET', v: 'a9f3e1c8b7d64f20a1e5c9d3b8f70e42',             secret: true },
	],
	staging: [
		{ key: 'NODE_ENV',     v: 'staging',                                               secret: false },
		{ key: 'PORT',         v: '3000',                                                   secret: false },
		{ key: 'DATABASE_URL', v: 'postgres://crust:s3cr3t@proofing-db-staging:5432/app', secret: true },
	],
};

export interface Repo {
	id: string;
	name: string;
	lang: string;
	langColor: string;
	updated: string;
}

export const REPOS: Repo[] = [
	{ id: 'crumb-api',    name: 'sourdough-labs/crumb-api',    lang: 'TypeScript', langColor: '#3178c6', updated: '4 minutes ago' },
	{ id: 'loaf-web',     name: 'sourdough-labs/loaf-web',     lang: 'Svelte',     langColor: '#ff3e00', updated: '2 hours ago' },
	{ id: 'yeast-worker', name: 'sourdough-labs/yeast-worker', lang: 'Rust',       langColor: '#dea584', updated: 'yesterday' },
	{ id: 'pantry-cli',   name: 'sourdough-labs/pantry-cli',   lang: 'Go',         langColor: '#00add8', updated: '3 days ago' },
	{ id: 'oven-metrics', name: 'sourdough-labs/oven-metrics', lang: 'Python',     langColor: '#3572a5', updated: '1 week ago' },
];

export const ACTIVITY = [
	{ dot: '#34d399', who: 'Basil',  text: ' restarted crumb-api',                          time: '2m ago' },
	{ dot: '#3fb984', who: 'system', text: ' auto-pulled ghcr.io/…/crumb-api:latest',       time: '12m ago' },
	{ dot: '#34d399', who: 'Rye',    text: ' deployed stack loaf-web',                      time: '14m ago' },
	{ dot: '#f0836b', who: 'system', text: ' health check failed on starter-bot',           time: '18m ago' },
	{ dot: '#34d399', who: 'Poppy',  text: ' invited Wheat to the guild',                   time: '2h ago' },
	{ dot: '#3fb984', who: 'Rye',    text: ' updated environment for proofing-db',          time: '3h ago' },
];

export type SourceProvider = 'github_app' | 'gitlab' | 'gitea' | 'bitbucket';

export interface Source {
	id: string;
	name: string;
	provider: SourceProvider;
	host?: string;
	repoCount: number;
	connectedAt: string;
}

export interface SourceMeta {
	label: string;
	color: string;
	bg: string;
}

export function sourceMeta(p: SourceProvider): SourceMeta {
	const M: Record<SourceProvider, SourceMeta> = {
		github_app: { label: 'GitHub App',        color: '#e7e9ee', bg: 'rgba(255,255,255,.08)' },
		gitlab:     { label: 'GitLab',            color: '#fc6d26', bg: 'rgba(252,109,38,.14)'  },
		gitea:      { label: 'Gitea',             color: '#52cc96', bg: 'rgba(63,185,132,.14)'  },
		bitbucket:  { label: 'Bitbucket Server',  color: '#5b8def', bg: 'rgba(91,141,239,.14)'  },
	};
	return M[p];
}

export const SOURCES: Record<string, Source[]> = {
	sourdough: [
		{ id: 'gh-sourdough', name: 'sourdough-labs', provider: 'github_app', repoCount: 5, connectedAt: '3 months ago' },
	],
	northwind: [],
};

export type NetworkDriver = 'bridge' | 'macvlan' | 'ipvlan';

export interface Network {
	id: string;
	name: string;
	driver: NetworkDriver;
	subnet: string;
	gateway: string;
	internal: boolean;
	connectedApps: string[];
	host: string;
	created: string;
}

export interface NetworkDriverMeta {
	label: string;
	color: string;
	bg: string;
	desc: string;
}

export function networkDriverMeta(d: NetworkDriver): NetworkDriverMeta {
	const M: Record<NetworkDriver, NetworkDriverMeta> = {
		bridge:  { label: 'Bridge',  color: '#7aa6f5', bg: 'rgba(91,141,239,.14)',  desc: 'Software bridge — containers on the same host communicate directly.' },
		macvlan: { label: 'macvlan', color: '#c09bf0', bg: 'rgba(169,120,230,.14)', desc: 'Containers appear as physical devices on the parent interface.' },
		ipvlan:  { label: 'ipvlan',  color: '#efc060', bg: 'rgba(224,168,62,.14)',  desc: 'Lightweight alternative to macvlan; shares the parent MAC.' },
	};
	return M[d];
}

export const NETWORKS: Record<string, Network[]> = {
	sourdough: [
		{
			id: 'bakery-net', name: 'bakery-net', driver: 'bridge',
			subnet: '10.88.0.0/16', gateway: '10.88.0.1', internal: false,
			connectedApps: ['crumb-api', 'loaf-web', 'proofing-db', 'crust-cache'],
			host: 'oven-01', created: '3 months ago',
		},
		{
			id: 'web-net', name: 'web-net', driver: 'bridge',
			subnet: '10.89.0.0/16', gateway: '10.89.0.1', internal: false,
			connectedApps: ['loaf-web'],
			host: 'oven-01', created: '3 months ago',
		},
	],
	northwind: [],
};

export type VolumeDriver = 'local' | 'nfs' | 'tmpfs';

export interface Volume {
	id: string;
	name: string;
	driver: VolumeDriver;
	mountpoint: string;
	size: string;
	attachedTo: string[];
	host: string;
	created: string;
}

export interface VolumeDriverMeta {
	label: string;
	color: string;
	bg: string;
	desc: string;
}

export function volumeDriverMeta(d: VolumeDriver): VolumeDriverMeta {
	const M: Record<VolumeDriver, VolumeDriverMeta> = {
		local: { label: 'local', color: '#7aa6f5', bg: 'rgba(91,141,239,.14)',  desc: 'Podman-managed volume on the host filesystem.' },
		nfs:   { label: 'NFS',   color: '#c09bf0', bg: 'rgba(169,120,230,.14)', desc: 'Network filesystem — volume lives on a remote NFS server.' },
		tmpfs: { label: 'tmpfs', color: '#efc060', bg: 'rgba(224,168,62,.14)',  desc: 'In-memory — data is lost when the container stops.' },
	};
	return M[d];
}

export const VOLUMES: Record<string, Volume[]> = {
	sourdough: [
		{ id: 'postgres-data', name: 'postgres-data', driver: 'local', mountpoint: '/var/lib/containers/storage/volumes/postgres-data/_data', size: '892 MB', attachedTo: ['proofing-db'],             host: 'oven-02', created: '3 months ago' },
		{ id: 'redis-data',    name: 'redis-data',    driver: 'local', mountpoint: '/var/lib/containers/storage/volumes/redis-data/_data',    size: '18 MB',  attachedTo: ['crust-cache'],            host: 'oven-02', created: '3 months ago' },
		{ id: 'uploads',       name: 'uploads',       driver: 'local', mountpoint: '/var/lib/containers/storage/volumes/uploads/_data',       size: '64 MB',  attachedTo: ['loaf-web', 'crumb-api'], host: 'oven-01', created: '6 weeks ago'  },
	],
	northwind: [],
};
