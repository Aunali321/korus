import { error } from '@sveltejs/kit';
import { playlistDetailCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, depends }) => {
    const id = Number(params.id);
    if (!Number.isFinite(id) || id <= 0) {
        error(404, 'Playlist not found');
    }
    depends(`app:playlist-${id}`);
    const playlist = await playlistDetailCache.load(id);
    return { playlist };
};
