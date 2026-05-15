import { playlistsCache } from '$lib/stores/pageData.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ depends }) => {
    depends('app:playlists');
    const playlists = await playlistsCache.load();
    return { playlists };
};
