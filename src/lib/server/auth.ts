import { betterAuth } from 'better-auth/minimal';
import { drizzleAdapter } from 'better-auth/adapters/drizzle';
import { sveltekitCookies } from 'better-auth/svelte-kit';
import { organization } from 'better-auth/plugins/organization';
import { getRequestEvent } from '$app/server';
import { db } from '$lib/server/db';
import { ac, roles } from '$lib/auth/permissions';

export const auth = betterAuth({
	baseURL: process.env.ORIGIN,
	secret: process.env.BETTER_AUTH_SECRET,
	database: drizzleAdapter(db, { provider: 'pg' }),
	emailAndPassword: { enabled: true },
	socialProviders: {
		github: {
			clientId: process.env.GITHUB_CLIENT_ID ?? '',
			clientSecret: process.env.GITHUB_CLIENT_SECRET ?? ''
		}
	},
	plugins: [
		organization({
			ac,
			roles,
			// The mock roles use "guild-master" where Better Auth defaults to "owner".
			creatorRole: 'guild-master',
			// Guild Master/Head Baker/Baker/Apprentice are the 4 static roles above —
			// no organizationRole DB rows needed for them. dynamicAccessControl only
			// persists *custom* roles a guild creates beyond those 4 (Phase 01 task 14).
			dynamicAccessControl: { enabled: true },
			schema: {
				organizationRole: {
					additionalFields: {
						color: { type: 'string' },
						note: { type: 'string' }
					}
				}
			}
		}),
		sveltekitCookies(getRequestEvent) // make sure this is the last plugin in the array
	]
});
