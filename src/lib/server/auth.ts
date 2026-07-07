import {
	ORIGIN,
	BETTER_AUTH_SECRET,
	GITHUB_CLIENT_ID,
	GITHUB_CLIENT_SECRET
} from '$app/env/private';

import { betterAuth } from 'better-auth/minimal';
import { drizzleAdapter } from 'better-auth/adapters/drizzle';
import { sveltekitCookies } from 'better-auth/svelte-kit';
import { organization } from 'better-auth/plugins/organization';
import { getRequestEvent } from '$app/server';
import { db } from '$lib/server/db';
import { ac, roles } from '$lib/server/auth/permissions';
import { ROLES } from '$lib/data/bakery';

export const auth = betterAuth({
	baseURL: ORIGIN,
	secret: BETTER_AUTH_SECRET,
	database: drizzleAdapter(db, { provider: 'pg' }),
	emailAndPassword: { enabled: true },
	socialProviders: {
		github: {
			clientId: GITHUB_CLIENT_ID,
			clientSecret: GITHUB_CLIENT_SECRET
		}
	},
	plugins: [
		organization({
			ac,
			roles,
			// The mock roles use "guild-master" where Better Auth defaults to "owner".
			creatorRole: 'guild-master',
			dynamicAccessControl: { enabled: true },
			schema: {
				organizationRole: {
					additionalFields: {
						color: { type: 'string' },
						note: { type: 'string' }
					}
				}
			},
			organizationHooks: {
				// Seed the 4 default positions (Guild Master/Head Baker/Baker/Apprentice)
				// on every new guild, matching bakery.ts's ROLES/ROLE_PERMS.
				afterCreateOrganization: async ({ organization: org }) => {
					const event = getRequestEvent();
					for (const role of ROLES) {
						await auth.api.createOrgRole({
							headers: event.request.headers,
							body: {
								organizationId: org.id,
								role: role.id,
								permission: roles[role.id as keyof typeof roles].statements as Record<
									string,
									string[]
								>,
								additionalFields: { color: role.color, note: role.note }
							}
						});
					}
				}
			}
		}),
		sveltekitCookies(getRequestEvent) // make sure this is the last plugin in the array
	]
});
