import { error } from '@sveltejs/kit';
import { artistDetailCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, depends }) => {
    const id = Number(params.id);
    if (!Number.isFinite(id) || id <= 0) {
        error(404, 'Artist not found');
    }
    depends(`app:artist-${id}`);
    const artist = await artistDetailCache.load(id);
    return { artist };
};
