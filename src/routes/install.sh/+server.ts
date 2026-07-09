import type { RequestHandler } from './$types';
import installScript from '$lib/server/agent/install.sh?raw';

export const GET: RequestHandler = () => {
	return new Response(installScript, {
		headers: { 'content-type': 'text/x-shellscript; charset=utf-8' }
	});
};
