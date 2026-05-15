import { homeCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ depends }) => {
    depends('app:home');
    const home = await homeCache.load();
    return { home };
};
