import { api } from '$lib/api';

function createFavoritesStore() {
    let songIds = $state<Set<number>>(new Set());
    let albumIds = $state<Set<number>>(new Set());
    let artistIds = $state<Set<number>>(new Set());
    let loaded = $state(false);
    let promise: Promise<void> | null = null;

    async function load() {
        if (loaded) return;
        if (promise) return promise;

        promise = api.getFavorites().then((data) => {
            songIds = new Set((data.songs || []).map(s => s.id));
            albumIds = new Set((data.albums || []).map(a => a.id));
            artistIds = new Set((data.artists || []).map(a => a.id));
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

    function isAlbumFavorite(albumId: number): boolean {
        return albumIds.has(albumId);
    }

    function isArtistFollowed(artistId: number): boolean {
        return artistIds.has(artistId);
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

    async function toggleAlbum(albumId: number): Promise<boolean> {
        const wasFavorite = albumIds.has(albumId);
        try {
            if (wasFavorite) {
                await api.unfavoriteAlbum(albumId);
                albumIds.delete(albumId);
                albumIds = new Set(albumIds);
            } else {
                await api.favoriteAlbum(albumId);
                albumIds.add(albumId);
                albumIds = new Set(albumIds);
            }
            return !wasFavorite;
        } catch (err) {
            console.error('Failed to toggle album favorite:', err);
            return wasFavorite;
        }
    }

    async function toggleArtist(artistId: number): Promise<boolean> {
        const wasFollowed = artistIds.has(artistId);
        try {
            if (wasFollowed) {
                await api.unfollowArtist(artistId);
                artistIds.delete(artistId);
                artistIds = new Set(artistIds);
            } else {
                await api.followArtist(artistId);
                artistIds.add(artistId);
                artistIds = new Set(artistIds);
            }
            return !wasFollowed;
        } catch (err) {
            console.error('Failed to toggle artist follow:', err);
            return wasFollowed;
        }
    }

    function reset() {
        songIds = new Set();
        albumIds = new Set();
        artistIds = new Set();
        loaded = false;
        promise = null;
    }

    return {
        get loaded() { return loaded; },
        load,
        isFavorite,
        isAlbumFavorite,
        isArtistFollowed,
        toggle,
        toggleAlbum,
        toggleArtist,
        reset
    };
}

export const favorites = createFavoritesStore();
