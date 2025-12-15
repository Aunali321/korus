import { api } from '$lib/api';

function createFavoritesStore() {
    let songIds = $state<Set<number>>(new Set());
    let loaded = $state(false);
    let promise: Promise<void> | null = null;

    async function load() {
        if (loaded) return;
        if (promise) return promise;

        promise = api.getFavorites().then((data) => {
            songIds = new Set((data.songs || []).map(s => s.id));
            loaded = true;
            promise = null;
        }).catch((err) => {
            console.error('Failed to load favorites:', err);
            promise = null;
        });

        return promise;
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
        promise = null;
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
