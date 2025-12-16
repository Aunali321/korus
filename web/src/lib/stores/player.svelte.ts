import type { Song, RepeatMode } from '$lib/types';
import { api } from '$lib/api';
import { settings } from './settings.svelte';

function createPlayerStore() {
    let currentSong = $state<Song | null>(null);
    let queue = $state<Song[]>([]);
    let originalQueue: Song[] = []; // Store original order for unshuffle
    let queueIndex = $state(0);
    let isPlaying = $state(false);
    let volume = $state(0.7);
    let progress = $state(0);
    let duration = $state(0);
    let shuffle = $state(false);
    let repeat = $state<RepeatMode>('off');
    let audio: HTMLAudioElement | null = null;
    let playStartTime = 0; // Track when playback started

    // Record play history for the current song
    function recordHistory() {
        if (!currentSong || playStartTime === 0 || !audio) return;

        const listenedSeconds = Math.floor(audio.currentTime);
        const totalDuration = duration || currentSong.duration || 1;
        const completionRate = Math.min(listenedSeconds / totalDuration, 1);

        // Only record if listened for at least 10 seconds
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

        // Record history when tab becomes hidden (covers refresh, close, tab switch)
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'hidden') {
                recordHistory();
            }
        });

        audio.addEventListener('timeupdate', () => {
            progress = audio!.currentTime;
        });

        audio.addEventListener('loadedmetadata', () => {
            // Only use audio.duration if valid (not Infinity, which happens with transcoded streams)
            if (audio!.duration && isFinite(audio!.duration)) {
                duration = audio!.duration;
            }
        });

        audio.addEventListener('error', (e) => {
            console.error('Audio error:', audio?.error);
        });

        audio.addEventListener('ended', () => {
            recordHistory(); // Record play before moving to next
            if (repeat === 'one') {
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
        });
    }

    function loadSong(song: Song) {
        initAudio();
        if (!audio) return;

        // Use song duration from API (works for transcoded streams where metadata may not be available)
        duration = song.duration || 0;
        progress = 0;

        const { format, bitrate } = settings.getStreamParams();
        const streamUrl = api.getStreamUrl(song.id, format, bitrate);

        audio.src = streamUrl;
        audio.load();
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
            // Record history for previous song before switching
            recordHistory();

            currentSong = song;
            playStartTime = 0; // Reset for new song
            if (songs) {
                originalQueue = [...songs];
                const startIndex = index ?? songs.findIndex((s) => s.id === song.id);
                
                if (shuffle) {
                    const result = shuffleQueue(songs, startIndex);
                    queue = result.shuffled;
                    queueIndex = result.newIndex;
                } else {
                    queue = songs;
                    queueIndex = startIndex;
                }
            }
            loadSong(song);
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

        // Record history for current song before switching
        recordHistory();

        let nextIndex = queueIndex + 1;
        if (nextIndex >= queue.length) {
            if (repeat === 'all') nextIndex = 0;
            else {
                pause();
                return;
            }
        }

        queueIndex = nextIndex;
        currentSong = queue[nextIndex];
        loadSong(currentSong);
        audio?.play().catch(console.error);
    }

    function prev() {
        if (queue.length === 0) return;

        // Record history for current song before switching or restarting
        recordHistory();

        if (audio && audio.currentTime > 3) {
            audio.currentTime = 0;
            playStartTime = Date.now();
            return;
        }

        let prevIndex = queueIndex - 1;
        if (prevIndex < 0) {
            if (repeat === 'all') prevIndex = queue.length - 1;
            else prevIndex = 0;
        }

        queueIndex = prevIndex;
        currentSong = queue[prevIndex];
        playStartTime = 0; // Reset for new song
        loadSong(currentSong);
        audio?.play().catch(console.error);
    }

    function seek(time: number) {
        if (audio) audio.currentTime = time;
    }

    function setVolume(v: number) {
        volume = v;
        if (audio) audio.volume = v;
    }

    function toggleShuffle() {
        shuffle = !shuffle;
        if (queue.length === 0 || !currentSong) return;

        if (shuffle) {
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
    }

    function toggleRepeat() {
        const modes: RepeatMode[] = ['off', 'all', 'one'];
        const idx = modes.indexOf(repeat);
        repeat = modes[(idx + 1) % modes.length];
    }

    function addToQueue(song: Song) {
        queue = [...queue, song];
    }

    function clearQueue() {
        queue = [];
        queueIndex = 0;
    }

    function playQueue(songs: Song[], startIndex = 0) {
        recordHistory();
        originalQueue = [...songs];
        
        if (shuffle) {
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
        shuffle = true;
        playStartTime = 0;
        loadSong(currentSong);
        audio?.play().catch(console.error);
    }

    return {
        get currentSong() { return currentSong; },
        get queue() { return queue; },
        get queueIndex() { return queueIndex; },
        get isPlaying() { return isPlaying; },
        get volume() { return volume; },
        get progress() { return progress; },
        get duration() { return duration; },
        get shuffle() { return shuffle; },
        get repeat() { return repeat; },
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
        playShuffled
    };
}

export const player = createPlayerStore();
