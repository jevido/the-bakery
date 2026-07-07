import { createAccessControl } from 'better-auth/plugins/access';
import { defaultStatements, ownerAc } from 'better-auth/plugins/organization/access';

/**
 * Resource/action shape ported from the mock PERM_GROUPS/ROLE_PERMS catalogue
 * in src/lib/data/bakery.ts, merged with Better Auth's own organization
 * resources (organization/member/invitation/team/ac) so built-in operations
 * (invite, role CRUD via dynamicAccessControl, etc.) stay authorized too.
 */
const statement = {
	...defaultStatements,
	guild: ['view_guild', 'manage_guild', 'manage_roles', 'manage_members', 'audit_log'],
	apps: ['view_apps', 'create_apps', 'deploy_apps', 'manage_env', 'view_secrets', 'delete_apps'],
	hosts: ['view_hosts', 'manage_hosts', 'manage_domains']
} as const;

export const ac = createAccessControl(statement);

// guild-master: "Sits above every position" — full permissions on everything (ROLE_PERMS: ['administrator']).
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

/** Keyed to match the mock ROLES/ROLE_PERMS ids and the `member.role` values Better Auth stores. */
export const roles = {
	'guild-master': guildMaster,
	'head-baker': headBaker,
	baker,
	apprentice
};
