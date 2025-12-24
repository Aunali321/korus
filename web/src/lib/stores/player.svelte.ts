import type { Song } from '$lib/types';
import { api, getAccessToken } from '$lib/api';
import { settings } from './settings.svelte';
import { library } from './library.svelte';

const VOLUME_KEY = 'korus_volume';
const STATE_KEY = 'korus_player_state';
const SAVE_INTERVAL = 30000;

interface PlayerStateData {
    currentSongId: number | null;
    queue: number[];
    queueIndex: number;
    progress: number;
}

function createPlayerStore() {
    let currentSong = $state<Song | null>(null);
    let queue = $state<Song[]>([]);
    let originalQueue: Song[] = [];
    let queueIndex = $state(0);
    let isPlaying = $state(false);
    let volume = $state(0.7);
    let progress = $state(0);
    let duration = $state(0);
    let audio: HTMLAudioElement | null = null;
    let playStartTime = 0;
    let initialized = $state(false);
    
    let saveTimeout: ReturnType<typeof setTimeout> | null = null;
    let lastSaveTime = 0;
    let periodicSaveInterval: ReturnType<typeof setInterval> | null = null;
    let globalListenersAdded = false;

    function loadVolumeLocal() {
        if (typeof localStorage === 'undefined') return;
        try {
            const stored = localStorage.getItem(VOLUME_KEY);
            if (stored) volume = parseFloat(stored);
        } catch {}
    }

    function saveVolumeLocal() {
        if (typeof localStorage === 'undefined') return;
        localStorage.setItem(VOLUME_KEY, String(volume));
    }

    function getStateData(): PlayerStateData {
        return {
            currentSongId: currentSong?.id ?? null,
            queue: queue.map(s => s.id),
            queueIndex,
            progress: audio?.currentTime ?? progress,
        };
    }

    function saveStateLocal() {
        if (typeof localStorage === 'undefined') return;
        localStorage.setItem(STATE_KEY, JSON.stringify(getStateData()));
    }

    function saveStateDebounced() {
        saveStateLocal();
        
        const now = Date.now();
        if (now - lastSaveTime < 5000) {
            if (saveTimeout) clearTimeout(saveTimeout);
            saveTimeout = setTimeout(() => {
                syncStateToServer();
            }, 5000);
            return;
        }
        
        syncStateToServer();
    }

    async function syncStateToServer() {
        lastSaveTime = Date.now();
        const state = getStateData();
        try {
            await api.savePlayerState({
                current_song_id: state.currentSongId,
                queue: state.queue,
                queue_index: state.queueIndex,
                progress: state.progress,
            });
        } catch (err) {
            console.error('Failed to sync player state:', err);
        }
    }

    function saveStateImmediate() {
        saveStateLocal();
        syncStateToServer();
    }

    async function loadState() {
        if (initialized) return;
        loadVolumeLocal();

        let stateData: PlayerStateData | null = null;

        if (typeof localStorage !== 'undefined') {
            try {
                const stored = localStorage.getItem(STATE_KEY);
                if (stored) stateData = JSON.parse(stored);
            } catch {}
        }

        try {
            const remote = await api.getPlayerState();
            if (remote.queue && remote.queue.length > 0) {
                stateData = {
                    currentSongId: remote.current_song_id,
                    queue: remote.queue,
                    queueIndex: remote.queue_index,
                    progress: remote.progress,
                };
            }
        } catch {}

        if (stateData && stateData.queue.length > 0) {
            await restoreState(stateData);
        }

        initialized = true;
    }

    async function restoreState(state: PlayerStateData) {
        // Don't restore if no current song was saved
        if (!state.currentSongId) return;
        
        try {
            await library.load();
        } catch (err) {
            console.error('Failed to load library for state restore:', err);
            return;
        }
        
        const songMap = new Map(library.songs.map(s => [s.id, s]));

        const restoredQueue: Song[] = [];
        for (const id of state.queue) {
            const song = songMap.get(id);
            if (song) restoredQueue.push(song);
        }

        if (restoredQueue.length === 0) return;

        // Verify the current song exists in the queue
        const currentSongInLibrary = songMap.get(state.currentSongId);
        if (!currentSongInLibrary) return;

        queue = restoredQueue;
        originalQueue = [...restoredQueue];

        const songIndex = queue.findIndex(s => s.id === state.currentSongId);
        if (songIndex < 0) return;

        queueIndex = songIndex;
        currentSong = currentSongInLibrary;

        initAudio();
        duration = currentSong.duration || 0;
        progress = state.progress || 0;
        loadSong(currentSong);
        if (audio && state.progress > 0) {
            audio.currentTime = state.progress;
        }
    }

    function recordHistory() {
        if (!currentSong || playStartTime === 0 || !audio) return;

        const listenedSeconds = Math.floor(audio.currentTime);
        const totalDuration = duration || currentSong.duration || 1;
        const completionRate = Math.min(listenedSeconds / totalDuration, 1);

        if (listenedSeconds >= 10) {
            api.recordPlay(currentSong.id, listenedSeconds, completionRate, 'web')
                .catch(err => console.error('Failed to record play:', err));
        }

        playStartTime = 0;
    }

    function initAudio() {
        if (typeof window === 'undefined') return;
        if (audio) return;

        audio = new Audio();
        audio.volume = volume;

        // Only add global listeners once
        if (!globalListenersAdded) {
            globalListenersAdded = true;

            window.addEventListener('beforeunload', () => {
                recordHistory();
                saveStateLocal();
                const state = getStateData();
                const data = JSON.stringify({
                    current_song_id: state.currentSongId,
                    queue: state.queue,
                    queue_index: state.queueIndex,
                    progress: state.progress,
                });
                const token = getAccessToken();
                const url = token ? `/api/player/state?token=${token}` : '/api/player/state';
                navigator.sendBeacon(url, new Blob([data], { type: 'application/json' }));
            });
        }

        if (periodicSaveInterval) clearInterval(periodicSaveInterval);
        periodicSaveInterval = setInterval(() => {
            if (isPlaying) {
                saveStateDebounced();
            }
        }, SAVE_INTERVAL);

        audio.addEventListener('timeupdate', () => {
            progress = audio!.currentTime;
        });

        audio.addEventListener('loadedmetadata', () => {
            if (audio!.duration && isFinite(audio!.duration)) {
                duration = audio!.duration;
            }
        });

        audio.addEventListener('error', () => {
            console.error('Audio error:', audio?.error);
        });

        audio.addEventListener('ended', () => {
            recordHistory();
            if (settings.repeat === 'one') {
                audio!.currentTime = 0;
                playStartTime = Date.now();
                audio!.play();
            } else {
                next();
            }
        });

        audio.addEventListener('play', () => {
            isPlaying = true;
            if (playStartTime === 0) {
                playStartTime = Date.now();
            }
        });

        audio.addEventListener('pause', () => {
            isPlaying = false;
            saveStateDebounced();
        });

        setupMediaSession();
    }

    function setupMediaSession() {
        if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) return;

        navigator.mediaSession.setActionHandler('play', () => {
            audio?.play().catch(console.error);
        });

        navigator.mediaSession.setActionHandler('pause', () => {
            audio?.pause();
        });

        navigator.mediaSession.setActionHandler('previoustrack', () => {
            prev();
        });

        navigator.mediaSession.setActionHandler('nexttrack', () => {
            next();
        });

        navigator.mediaSession.setActionHandler('seekto', (details) => {
            if (audio && details.seekTime !== undefined) {
                audio.currentTime = details.seekTime;
            }
        });

        navigator.mediaSession.setActionHandler('seekbackward', (details) => {
            if (audio) {
                audio.currentTime = Math.max(0, audio.currentTime - (details.seekOffset || 10));
            }
        });

        navigator.mediaSession.setActionHandler('seekforward', (details) => {
            if (audio) {
                audio.currentTime = Math.min(duration, audio.currentTime + (details.seekOffset || 10));
            }
        });
    }

    function updateMediaSessionMetadata(song: Song) {
        if (typeof navigator === 'undefined' || !('mediaSession' in navigator)) return;

        const artwork = api.getArtworkUrl(song.id);
        navigator.mediaSession.metadata = new MediaMetadata({
            title: song.title,
            artist: song.artist?.name || 'Unknown Artist',
            album: song.album?.title || 'Unknown Album',
            artwork: [
                { src: artwork, sizes: '512x512', type: 'image/jpeg' },
            ],
        });
    }

    function loadSong(song: Song) {
        initAudio();
        if (!audio) return;

        duration = song.duration || 0;
        progress = 0;

        const { format, bitrate } = settings.getStreamParams();
        const streamUrl = api.getStreamUrl(song.id, format, bitrate);

        audio.src = streamUrl;
        audio.load();
        updateMediaSessionMetadata(song);
    }

    function shuffleQueue(songs: Song[], currentIndex: number): { shuffled: Song[], newIndex: number } {
        const current = songs[currentIndex];
        const remaining = songs.filter((_, i) => i !== currentIndex);
        for (let i = remaining.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [remaining[i], remaining[j]] = [remaining[j], remaining[i]];
        }
        return { shuffled: [current, ...remaining], newIndex: 0 };
    }

    function play(song?: Song, songs?: Song[], index?: number) {
        initAudio();
        if (!audio) return;

        if (song) {
            recordHistory();

            currentSong = song;
            playStartTime = 0;
            if (songs) {
                originalQueue = [...songs];
                const startIndex = index ?? songs.findIndex((s) => s.id === song.id);
                
                if (settings.shuffle) {
                    const result = shuffleQueue(songs, startIndex);
                    queue = result.shuffled;
                    queueIndex = result.newIndex;
                } else {
                    queue = songs;
                    queueIndex = startIndex;
                }
            }
            loadSong(song);
            saveStateDebounced();
        }

        audio.play().catch(console.error);
    }

    function pause() {
        audio?.pause();
    }

    function toggle() {
        if (isPlaying) pause();
        else play();
    }

    function next() {
        if (queue.length === 0) return;

        recordHistory();

        let nextIndex = queueIndex + 1;
        if (nextIndex >= queue.length) {
            if (settings.repeat === 'all') nextIndex = 0;
            else {
                pause();
                return;
            }
        }

        queueIndex = nextIndex;
        currentSong = queue[nextIndex];
        loadSong(currentSong);
        saveStateDebounced();
        audio?.play().catch(console.error);
    }

    function prev() {
        if (queue.length === 0) return;

        recordHistory();

        if (audio && audio.currentTime > 3) {
            audio.currentTime = 0;
            playStartTime = Date.now();
            return;
        }

        let prevIndex = queueIndex - 1;
        if (prevIndex < 0) {
            if (settings.repeat === 'all') prevIndex = queue.length - 1;
            else prevIndex = 0;
        }

        queueIndex = prevIndex;
        currentSong = queue[prevIndex];
        playStartTime = 0;
        loadSong(currentSong);
        saveStateDebounced();
        audio?.play().catch(console.error);
    }

    function seek(time: number) {
        if (audio) audio.currentTime = time;
    }

    function setVolume(v: number) {
        volume = v;
        if (audio) audio.volume = v;
        saveVolumeLocal();
    }

    async function toggleShuffle() {
        await settings.toggleShuffle();
        if (queue.length === 0 || !currentSong) return;

        if (settings.shuffle) {
            originalQueue = [...queue];
            const result = shuffleQueue(queue, queueIndex);
            queue = result.shuffled;
            queueIndex = result.newIndex;
        } else {
            const current = currentSong;
            queue = [...originalQueue];
            queueIndex = queue.findIndex(s => s.id === current.id);
            if (queueIndex < 0) queueIndex = 0;
        }
        saveStateDebounced();
    }

    async function toggleRepeat() {
        await settings.toggleRepeat();
    }

    function addToQueue(song: Song) {
        queue = [...queue, song];
        saveStateDebounced();
    }

    function clearQueue() {
        queue = [];
        queueIndex = 0;
        saveStateDebounced();
    }

    function playQueue(songs: Song[], startIndex = 0) {
        recordHistory();
        originalQueue = [...songs];
        
        if (settings.shuffle) {
            const result = shuffleQueue(songs, startIndex);
            queue = result.shuffled;
            queueIndex = result.newIndex;
            currentSong = queue[queueIndex];
        } else {
            queue = songs;
            queueIndex = startIndex;
            currentSong = songs[startIndex];
        }
        
        playStartTime = 0;
        loadSong(currentSong);
        saveStateDebounced();
        audio?.play().catch(console.error);
    }

    function playShuffled(songs: Song[]) {
        recordHistory();
        originalQueue = [...songs];
        const startIndex = Math.floor(Math.random() * songs.length);
        const result = shuffleQueue(songs, startIndex);
        queue = result.shuffled;
        queueIndex = result.newIndex;
        currentSong = queue[queueIndex];
        settings.setShuffle(true);
        playStartTime = 0;
        loadSong(currentSong);
        saveStateDebounced();
        audio?.play().catch(console.error);
    }

    function reset() {
        if (audio) {
            audio.pause();
            audio.src = '';
        }
        currentSong = null;
        queue = [];
        originalQueue = [];
        queueIndex = 0;
        isPlaying = false;
        progress = 0;
        duration = 0;
        playStartTime = 0;
        initialized = false;
        if (periodicSaveInterval) {
            clearInterval(periodicSaveInterval);
            periodicSaveInterval = null;
        }
        if (typeof localStorage !== 'undefined') {
            localStorage.removeItem(STATE_KEY);
        }
    }

    loadVolumeLocal();

    return {
        get currentSong() { return currentSong; },
        get queue() { return queue; },
        get queueIndex() { return queueIndex; },
        get isPlaying() { return isPlaying; },
        get volume() { return volume; },
        get progress() { return progress; },
        get duration() { return duration; },
        get shuffle() { return settings.shuffle; },
        get repeat() { return settings.repeat; },
        get initialized() { return initialized; },
        play,
        pause,
        toggle,
        next,
        prev,
        seek,
        setVolume,
        toggleShuffle,
        toggleRepeat,
        addToQueue,
        clearQueue,
        playQueue,
        playShuffled,
        loadState,
        reset,
    };
}

export const player = createPlayerStore();
