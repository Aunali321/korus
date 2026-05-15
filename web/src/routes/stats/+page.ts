import { statsCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url }) => {
    const period = url.searchParams.get('period') || 'all_time';
    const bundle = await statsCache.load(period);
    return { period, bundle };
};
