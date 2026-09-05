import type { PageServerLoad } from './$types';
import type { UpdateSnapshot } from '$lib/system-update';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		const response = await fetch('/internal/system-update', {
			headers: { Accept: 'application/json' }
		});
		if (!response.ok) return { snapshot: null };
		return { snapshot: await response.json() as UpdateSnapshot };
	} catch {
		return { snapshot: null };
	}
};
