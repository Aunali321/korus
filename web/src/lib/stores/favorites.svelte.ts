import { api } from '$lib/api';

function createFavoritesStore() {
    let songIds = $state<Set<number>>(new Set());
    let loaded = $state(false);

    async function load() {
        if (loaded) return;
        try {
            const data = await api.getFavorites();
            songIds = new Set((data.songs || []).map(s => s.id));
            loaded = true;
        } catch (err) {
            console.error('Failed to load favorites:', err);
        }
    }

    function isFavorite(songId: number): boolean {
        return songIds.has(songId);
    }

    async function toggle(songId: number): Promise<boolean> {
        const wasFavorite = songIds.has(songId);
        try {
            if (wasFavorite) {
                await api.unfavoriteSong(songId);
                songIds.delete(songId);
                songIds = new Set(songIds);
            } else {
                await api.favoriteSong(songId);
                songIds.add(songId);
                songIds = new Set(songIds);
            }
            return !wasFavorite;
        } catch (err) {
            console.error('Failed to toggle favorite:', err);
            return wasFavorite;
        }
    }

    function reset() {
        songIds = new Set();
        loaded = false;
    }

    return {
        get loaded() { return loaded; },
        load,
        isFavorite,
        toggle,
        reset
    };
}

export const favorites = createFavoritesStore();
