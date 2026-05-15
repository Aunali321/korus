import { SvelteMap, SvelteSet } from 'svelte/reactivity';
import { api } from '$lib/api';
import type { Song, Album, Artist, Playlist, Stats, PlayHistory, Insights, WrappedData } from '$lib/types';

export interface HomeData {
    recent_plays: Song[];
    new_additions: Album[];
}

export interface FavoritesData {
    songs: Song[];
    albums: Album[];
    artists: Artist[];
}

export interface AlbumDetail {
    id: number;
    title: string;
    year?: number;
    cover_path?: string;
    mbid?: string;
    artist?: Artist;
    songs: Song[];
    created_at?: string;
}

export interface ArtistDetail {
    id: number;
    name: string;
    bio?: string;
    image_path?: string;
    mbid?: string;
    albums: Album[];
    songs: Song[];
}

export interface StatsBundle {
    stats: Stats;
    history: PlayHistory[];
    insights: Insights;
}

/* Singleton cache for routes that fetch a single resource. */
function createCache<T>(fetcher: () => Promise<T>) {
    let data = $state<T | null>(null);
    let promise: Promise<T> | null = null;

    async function load(force = false): Promise<T> {
        if (data && !force) return data;
        if (promise && !force) return promise;

        promise = fetcher()
            .then((result) => {
                data = result;
                promise = null;
                return result;
            })
            .catch((err) => {
                promise = null;
                throw err;
            });
        return promise;
    }

    return {
        get data() { return data; },
        get hasData() { return data !== null; },
        load,
        invalidate() { data = null; promise = null; },
        set(value: T) { data = value; }
    };
}

/* Per-id cache for /:id detail routes. Uses SvelteMap/SvelteSet for
   proper reactivity without manual reassignment. */
function createKeyedCache<T>(fetcher: (id: number) => Promise<T>) {
    const entries = new SvelteMap<number, T>();
    const loadingKeys = new SvelteSet<number>();
    const promises = new Map<number, Promise<T>>();

    async function load(id: number, force = false): Promise<T> {
        const existing = entries.get(id);
        if (existing && !force) return existing;

        const inflight = promises.get(id);
        if (inflight && !force) return inflight;

        loadingKeys.add(id);
        const p = fetcher(id)
            .then((result) => {
                entries.set(id, result);
                loadingKeys.delete(id);
                promises.delete(id);
                return result;
            })
            .catch((err) => {
                loadingKeys.delete(id);
                promises.delete(id);
                throw err;
            });
        promises.set(id, p);
        return p;
    }

    return {
        get(id: number): T | undefined { return entries.get(id); },
        has(id: number): boolean { return entries.has(id); },
        isLoading(id: number): boolean { return loadingKeys.has(id); },
        load,
        set(id: number, value: T) { entries.set(id, value); },
        invalidate(id?: number) {
            if (id === undefined) {
                entries.clear();
                promises.clear();
            } else {
                entries.delete(id);
                promises.delete(id);
            }
        }
    };
}

export const homeCache = createCache<HomeData>(() => api.getHome());
export const favoritesPageCache = createCache<FavoritesData>(() => api.getFavorites());
export const playlistsCache = createCache<Playlist[]>(() => api.getPlaylists());

/* Wrapped is keyed by period ("year" | "month"). */
const wrappedEntries = new SvelteMap<string, WrappedData>();
const wrappedPromises = new Map<string, Promise<WrappedData>>();
export const wrappedCache = {
    get(period: string): WrappedData | undefined { return wrappedEntries.get(period); },
    has(period: string): boolean { return wrappedEntries.has(period); },
    async load(period: string, force = false): Promise<WrappedData> {
        const existing = wrappedEntries.get(period);
        if (existing && !force) return existing;
        const inflight = wrappedPromises.get(period);
        if (inflight && !force) return inflight;
        const p = api.getWrapped(period)
            .then((result) => {
                wrappedEntries.set(period, result);
                wrappedPromises.delete(period);
                return result;
            })
            .catch((err) => {
                wrappedPromises.delete(period);
                throw err;
            });
        wrappedPromises.set(period, p);
        return p;
    },
    invalidate() {
        wrappedEntries.clear();
        wrappedPromises.clear();
    }
};

export const albumDetailCache = createKeyedCache<AlbumDetail>((id) => api.getAlbum(id));
export const artistDetailCache = createKeyedCache<ArtistDetail>((id) => api.getArtist(id));
export const playlistDetailCache = createKeyedCache<Playlist>((id) => api.getPlaylist(id));

/* Stats has a period selector, so it's keyed by period string. */
const statsEntries = new SvelteMap<string, StatsBundle>();
const statsPromises = new Map<string, Promise<StatsBundle>>();
let insightsAndHistory: { history: PlayHistory[]; insights: Insights } | null = null;

export const statsCache = {
    get(period: string): StatsBundle | undefined { return statsEntries.get(period); },
    has(period: string): boolean { return statsEntries.has(period); },
    async load(period: string, force = false): Promise<StatsBundle> {
        const existing = statsEntries.get(period);
        if (existing && !force) return existing;

        const inflight = statsPromises.get(period);
        if (inflight && !force) return inflight;

        const p = (async () => {
            const [stats, ...rest] = await Promise.all([
                api.getStats(period),
                insightsAndHistory ? Promise.resolve(insightsAndHistory.history) : api.getHistory(50, 0),
                insightsAndHistory ? Promise.resolve(insightsAndHistory.insights) : api.getInsights(),
            ]);
            const history = rest[0];
            const insights = rest[1];
            insightsAndHistory = { history, insights };
            const bundle: StatsBundle = { stats, history, insights };
            statsEntries.set(period, bundle);
            statsPromises.delete(period);
            return bundle;
        })().catch((err) => {
            statsPromises.delete(period);
            throw err;
        });
        statsPromises.set(period, p);
        return p;
    },
    invalidate() {
        statsEntries.clear();
        statsPromises.clear();
        insightsAndHistory = null;
    }
};
