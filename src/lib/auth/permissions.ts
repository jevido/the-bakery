import { createAccessControl } from 'better-auth/plugins/access';
import { defaultStatements, ownerAc } from 'better-auth/plugins/organization/access';

/**
 * Resource/action shape merged with Better Auth's own organization resources
 * (organization/member/invitation/team/ac) so built-in operations (invite,
 * role CRUD via dynamicAccessControl, etc.) stay authorized too.
 */
const statement = {
	...defaultStatements,
	guild: ['view_guild', 'manage_guild', 'manage_roles', 'manage_members', 'audit_log'],
	apps: ['view_apps', 'create_apps', 'deploy_apps', 'manage_env', 'view_secrets', 'delete_apps'],
	hosts: ['view_hosts', 'manage_hosts', 'manage_domains']
} as const;

export const ac = createAccessControl(statement);

// guild-master: "Sits above every position" — full permissions on everything.
export const guildMaster = ac.newRole({
	...ownerAc.statements,
	guild: ['view_guild', 'manage_guild', 'manage_roles', 'manage_members', 'audit_log'],
	apps: ['view_apps', 'create_apps', 'deploy_apps', 'manage_env', 'view_secrets', 'delete_apps'],
	hosts: ['view_hosts', 'manage_hosts', 'manage_domains']
});

// head-baker: "Manage apps, hosts, and all positions beneath them" — everything except guild settings themselves.
export const headBaker = ac.newRole({
	organization: [],
	member: ['create', 'update', 'delete'],
	invitation: ['create', 'cancel'],
	team: [],
	ac: ['create', 'read', 'update', 'delete'],
	guild: ['view_guild', 'manage_roles', 'manage_members', 'audit_log'],
	apps: ['view_apps', 'create_apps', 'deploy_apps', 'manage_env', 'view_secrets', 'delete_apps'],
	hosts: ['view_hosts', 'manage_hosts', 'manage_domains']
});

// baker: "Deploy apps and manage environments, but cannot touch roles or hosts."
export const baker = ac.newRole({
	organization: [],
	member: [],
	invitation: [],
	team: [],
	ac: ['read'],
	guild: ['view_guild'],
	apps: ['view_apps', 'create_apps', 'deploy_apps', 'manage_env'],
	hosts: ['view_hosts', 'manage_domains']
});

// apprentice: "Read-only access."
export const apprentice = ac.newRole({
	organization: [],
	member: [],
	invitation: [],
	team: [],
	ac: ['read'],
	guild: ['view_guild'],
	apps: ['view_apps'],
	hosts: ['view_hosts']
});

/** Keyed to match the ROLES ids below and the `member.role` values Better Auth stores. */
export const roles = {
	'guild-master': guildMaster,
	'head-baker': headBaker,
	baker,
	apprentice
};

/** The custom (non-Better-Auth-builtin) resources — what dynamic/custom roles are scoped to. */
export const CUSTOM_ROLE_RESOURCES = ['guild', 'apps', 'hosts'] as const;
export type CustomRoleResource = (typeof CUSTOM_ROLE_RESOURCES)[number];

/** Maps each permission id in PERM_GROUPS below to the access-control
 * resource it lives under in this statement. Shared by guild-context.ts's
 * requireGuild() and any UI rendering the permission catalogue (e.g. the
 * roles page). */
export const PERMISSION_RESOURCE: Record<string, CustomRoleResource> = {
	view_guild: 'guild',
	manage_guild: 'guild',
	manage_roles: 'guild',
	manage_members: 'guild',
	audit_log: 'guild',
	view_apps: 'apps',
	create_apps: 'apps',
	deploy_apps: 'apps',
	manage_env: 'apps',
	view_secrets: 'apps',
	delete_apps: 'apps',
	view_hosts: 'hosts',
	manage_hosts: 'hosts',
	manage_domains: 'hosts'
};

/** Flattens a static role's guild/apps/hosts grants into one set of permission ids. */
export function staticRolePermissionIds(roleKey: keyof typeof roles): Set<string> {
	const statements = roles[roleKey].statements as Record<string, readonly string[] | undefined>;
	const ids = CUSTOM_ROLE_RESOURCES.flatMap((resource) => statements[resource] ?? []);
	return new Set(ids);
}

// ---------------------------------------------------------------------------
// Static role/permission display metadata. These are NOT mock organizational
// data (unlike bakery.ts's removed Guild/Member/GUILDS/MEMBERS) — they're
// permanent, code-defined app configuration: the 4 default roles' display
// color/description, and the human-readable grouping/labels/descriptions for
// the permission catalogue whose real grants live in `roles` above. There's
// no database row these could be sourced from instead (see Phase 01 task 14).
// ---------------------------------------------------------------------------

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

export const ROLES: Role[] = [
	{
		id: 'guild-master',
		name: 'Guild Master',
		color: '#e0a83e',
		count: 1,
		master: true,
		note: 'The founder of the guild. Sits above every position and cannot be edited or deleted.'
	},
	{
		id: 'head-baker',
		name: 'Head Baker',
		color: '#a978e6',
		count: 2,
		note: 'Trusted maintainers. Manage apps, hosts, and all positions beneath them.'
	},
	{
		id: 'baker',
		name: 'Baker',
		color: '#3fb984',
		count: 3,
		note: 'Developers. Deploy apps and manage environments, but cannot touch roles or hosts.'
	},
	{
		id: 'apprentice',
		name: 'Apprentice',
		color: '#5b8def',
		count: 4,
		note: 'Read-only access. Perfect for stakeholders watching the ovens.'
	}
];

export const PERM_GROUPS: PermGroup[] = [
	{
		name: 'GENERAL',
		perms: [
			{ id: 'view_guild', label: 'View Guild', desc: 'See the guild and its modules' },
			{ id: 'manage_guild', label: 'Manage Guild', desc: 'Edit name, invites, and settings' },
			{
				id: 'manage_roles',
				label: 'Manage Roles',
				desc: 'Create and edit positions below your own'
			},
			{ id: 'manage_members', label: 'Manage Members', desc: 'Assign roles, kick, and ban' },
			{ id: 'audit_log', label: 'View Audit Log', desc: 'Review every action taken in the guild' }
		]
	},
	{
		name: 'APPS',
		perms: [
			{ id: 'view_apps', label: 'View Apps', desc: 'See apps and their status' },
			{ id: 'create_apps', label: 'Create Apps', desc: 'Deploy new apps from a repo' },
			{ id: 'deploy_apps', label: 'Deploy & Redeploy', desc: 'Trigger deployments and rollbacks' },
			{ id: 'manage_env', label: 'Manage Environment', desc: 'Edit variables and secrets' },
			{ id: 'view_secrets', label: 'View Secrets', desc: 'Reveal masked secret values' },
			{ id: 'delete_apps', label: 'Delete Apps', desc: 'Permanently remove an app' }
		]
	},
	{
		name: 'INFRASTRUCTURE',
		perms: [
			{ id: 'view_hosts', label: 'View Hosts', desc: 'See connected machines' },
			{ id: 'manage_hosts', label: 'Manage Hosts', desc: 'Add, remove, and configure hosts' },
			{
				id: 'manage_domains',
				label: 'Manage Domains & Proxy',
				desc: 'Edit domains and Caddy routes'
			}
		]
	},
	{
		name: 'DANGER',
		perms: [
			{
				id: 'administrator',
				label: 'Administrator',
				desc: 'All permissions. Grant with caution.',
				danger: true
			}
		]
	}
];
