import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { auth } from '$lib/server/auth';
import { APIError } from 'better-auth/api';

export const load: PageServerLoad = (event) => {
	if (event.locals.user) {
		redirect(302, '/');
	}
	return {};
};

export const actions: Actions = {
	signUp: async (event) => {
		const formData = await event.request.formData();
		const email = formData.get('email')?.toString() ?? '';
		const password = formData.get('password')?.toString() ?? '';
		const name = formData.get('name')?.toString() ?? '';

		try {
			await auth.api.signUpEmail({
				headers: event.request.headers,
				body: { email, password, name }
			});
		} catch (error) {
			if (error instanceof APIError) {
				return fail(400, { message: error.message || 'Registration failed' });
			}
			return fail(500, { message: 'Unexpected error' });
		}

		redirect(302, '/');
	},
	signInSocial: async (event) => {
		const result = await auth.api.signInSocial({
			headers: event.request.headers,
			body: { provider: 'github', callbackURL: '/' }
		});

		if (result.url) {
			redirect(302, result.url);
		}
		return fail(400, { message: 'Social sign-in failed' });
	}
};
