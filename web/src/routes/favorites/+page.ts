import { favoritesPageCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ depends }) => {
    depends('app:favorites');
    const favorites = await favoritesPageCache.load();
    return { favorites };
};
