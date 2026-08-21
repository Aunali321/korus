import { invalidate } from '$app/navigation';
import { api } from '$lib/api';
import { favoritesPageCache } from './pageData.svelte';

export type FavoriteKind = 'song' | 'album' | 'artist';

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

    function idsFor(kind: FavoriteKind): Set<number> {
        if (kind === 'song') return songIds;
        if (kind === 'album') return albumIds;
        return artistIds;
    }

    // applyLocal reflects a change that already happened on the server.
    function applyLocal(kind: FavoriteKind, id: number, on: boolean) {
        const next = new Set(idsFor(kind));
        if (on) next.add(id);
        else next.delete(id);

        if (kind === 'song') songIds = next;
        else if (kind === 'album') albumIds = next;
        else artistIds = next;

        favoritesPageCache.invalidate();
        invalidate('app:favorites');
    }

    function callApi(kind: FavoriteKind, id: number, on: boolean): Promise<unknown> {
        if (kind === 'song') return on ? api.favoriteSong(id) : api.unfavoriteSong(id);
        if (kind === 'album') return on ? api.favoriteAlbum(id) : api.unfavoriteAlbum(id);
        return on ? api.followArtist(id) : api.unfollowArtist(id);
    }

    async function set(kind: FavoriteKind, id: number, on: boolean): Promise<boolean> {
        try {
            await callApi(kind, id, on);
            applyLocal(kind, id, on);
            return on;
        } catch (err) {
            console.error(`Failed to update ${kind} favorite:`, err);
            return idsFor(kind).has(id);
        }
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
        toggle: (songId: number) => set('song', songId, !songIds.has(songId)),
        toggleAlbum: (albumId: number) => set('album', albumId, !albumIds.has(albumId)),
        toggleArtist: (artistId: number) => set('artist', artistId, !artistIds.has(artistId)),
        set,
        // applyRemote records a change made elsewhere, such as by the assistant.
        applyRemote: applyLocal,
        reset
    };
}

export const favorites = createFavoritesStore();
