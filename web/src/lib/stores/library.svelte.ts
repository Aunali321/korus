import { api } from '$lib/api';
import type { Song, Album, Artist } from '$lib/types';

interface LibraryData {
    songs: Song[];
    albums: Album[];
    artists: Artist[];
}

function createLibraryStore() {
    let data = $state<LibraryData | null>(null);
    let loading = $state(false);
    let promise = $state<Promise<LibraryData> | null>(null);

    async function load(force = false): Promise<LibraryData> {
        if (data && !force) return data;
        if (promise && !force) return promise;

        loading = true;
        promise = api.getLibrary().then((result) => {
            data = result;
            loading = false;
            promise = null;
            return result;
        }).catch((err) => {
            loading = false;
            promise = null;
            throw err;
        });

        return promise;
    }

    function invalidate() {
        data = null;
        promise = null;
    }

    function reset() {
        data = null;
        promise = null;
        loading = false;
    }

    return {
        get data() { return data; },
        get loading() { return loading; },
        get songs() { return data?.songs ?? []; },
        get albums() { return data?.albums ?? []; },
        get artists() { return data?.artists ?? []; },
        load,
        invalidate,
        reset
    };
}

export const library = createLibraryStore();
