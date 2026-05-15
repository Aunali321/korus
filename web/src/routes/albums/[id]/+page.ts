import { error } from '@sveltejs/kit';
import { albumDetailCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, depends }) => {
    const id = Number(params.id);
    if (!Number.isFinite(id) || id <= 0) {
        error(404, 'Album not found');
    }
    depends(`app:album-${id}`);
    const album = await albumDetailCache.load(id);
    return { album };
};
