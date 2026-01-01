import type { Song } from '$lib/types';

interface ContextMenuState {
    isOpen: boolean;
    position: { x: number; y: number };
    song: Song | null;
    playlistId?: number;
}

function createContextMenuStore() {
    let state = $state<ContextMenuState>({
        isOpen: false,
        position: { x: 0, y: 0 },
        song: null,
        playlistId: undefined,
    });

    function open(song: Song, x: number, y: number, playlistId?: number) {
        const menuWidth = 200;
        const menuHeight = 200;

        state = {
            isOpen: true,
            position: {
                x: Math.min(x, window.innerWidth - menuWidth - 8),
                y: Math.min(y, window.innerHeight - menuHeight - 8),
            },
            song,
            playlistId,
        };
    }

    function close() {
        state = {
            ...state,
            isOpen: false,
        };
    }

    return {
        get isOpen() { return state.isOpen; },
        get position() { return state.position; },
        get song() { return state.song; },
        get playlistId() { return state.playlistId; },
        open,
        close,
    };
}

export const contextMenu = createContextMenuStore();
