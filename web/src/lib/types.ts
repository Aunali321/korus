export interface Artist {
    id: number;
    name: string;
    bio?: string;
    image_path?: string;
    mbid?: string;
    created_at?: string;
}

export interface Album {
    id: number;
    title: string;
    artist_id: number;
    artist?: Artist;
    cover_path?: string;
    year?: number;
    mbid?: string;
    created_at?: string;
}

export interface Song {
    id: number;
    title: string;
    album_id: number;
    album?: Album;
    artist?: Artist;
    duration: number;
    track_number?: number;
    file_path?: string;
    lyrics?: string;
    lyrics_synced?: string;
    mbid?: string;
}

export interface Playlist {
    id: number;
    name: string;
    description?: string;
    public: boolean;
    user_id: number;
    songs?: Song[];
    song_count?: number;
    first_song_id?: number;
    created_at?: string;
    updated_at?: string;
}

export interface User {
    id: number;
    username: string;
    email: string;
    role: 'user' | 'admin';
    created_at?: string;
}

export interface PlayHistory {
    id: number;
    song: Song;
    played_at: string;
    duration_listened: number;
    completion_rate: number;
    source?: string;
}

export interface SearchResults {
    songs: Song[];
    albums: Album[];
    artists: Artist[];
    playlists: Playlist[];
}

export interface LibraryStats {
    total_songs: number;
    total_albums: number;
    total_artists: number;
    total_playlists: number;
}

export interface Stats {
    total_plays: number;
    total_duration: number;
    unique_songs: number;
    unique_artists: number;
    top_songs: Array<{ song: Song; play_count: number }>;
    top_artists: Array<{ artist: Artist; play_count: number }>;
    top_albums: Array<{ album: Album; play_count: number }>;
    top_genres: Array<{ genre: string; play_count: number }>;
    listening_by_hour?: Record<string, number>;
    listening_by_day?: Record<string, number>;
}

export interface WrappedData {
    period: string;
    top_song?: Song;
    top_artist?: Artist;
    top_album?: Album;
    total_minutes: number;
    unique_songs: number;
    unique_artists: number;
    personality?: string;
    milestones: string[];
}

export interface Insights {
    current_streak: number;
    longest_streak: number;
    trends: Array<{ label: string; value: number; change: number }>;
    predictions: string[];
    fun_facts: string[];
}

export interface SocialStats {
    leaderboard: Array<{ user: User; play_count: number; rank: number }>;
    taste_matches: Array<{ user: User; similarity: number }>;
    your_rank: number;
}

export interface ScanJob {
    id: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    progress: number;
    total: number;
    current_file?: string;
    started_at?: string;
    finished_at?: string;
    error?: string;
}

export interface SystemInfo {
    version: string;
    uptime: number;
    database_size: number;
    total_songs: number;
    total_albums: number;
    total_artists: number;
}

export type RepeatMode = 'off' | 'one' | 'all';
