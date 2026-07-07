import { createAuthClient } from 'better-auth/svelte';
import { organizationClient, inferOrgAdditionalFields } from 'better-auth/client/plugins';
import type { auth } from '$lib/server/auth';
import { ac, roles } from '$lib/auth/permissions';

export const authClient = createAuthClient({
	plugins: [
		organizationClient({
			ac,
			roles,
			dynamicAccessControl: { enabled: true },
			schema: inferOrgAdditionalFields<typeof auth>()
		})
	]
});
