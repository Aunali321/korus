import type {
  User,
  Song,
  Album,
  Artist,
  Playlist,
  PlayHistory,
  SearchResults,
  Stats,
  WrappedData,
  Insights,
  ScanJob,
  SystemInfo,
  StreamingOptions,
} from "./types";

const API_URL_KEY = "korus_api_url";
const ACCESS_TOKEN_KEY = "korus_access_token";
const REFRESH_TOKEN_KEY = "korus_refresh_token";

function getApiUrl(): string {
  if (typeof localStorage === "undefined") return "/api";
  return localStorage.getItem(API_URL_KEY) || "/api";
}

export function getAccessToken(): string | null {
  if (typeof localStorage === "undefined") return null;
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

function getRefreshToken(): string | null {
  if (typeof localStorage === "undefined") return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setApiUrl(url: string): void {
  localStorage.setItem(API_URL_KEY, url);
}

export function setTokens(access: string, refresh: string): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, access);
  localStorage.setItem(REFRESH_TOKEN_KEY, refresh);
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

async function refreshTokens(): Promise<boolean> {
  if (isRefreshing && refreshPromise) {
    return refreshPromise;
  }

  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  isRefreshing = true;
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${getApiUrl()}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (!res.ok) {
        clearTokens();
        return false;
      }

      const data = await res.json();
      setTokens(data.access_token, data.refresh_token);
      return true;
    } catch {
      clearTokens();
      return false;
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  retry = true,
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };

  // Get token fresh each time (not cached in closure)
  const token = getAccessToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${getApiUrl()}${path}`, {
    ...options,
    headers,
  });

  if (res.status === 401 && retry) {
    const refreshed = await refreshTokens();
    if (refreshed) {
      // Retry with fresh token from localStorage
      return request<T>(path, options, false);
    }
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(error.error || error.message || res.statusText);
  }

  if (res.status === 204) return {} as T;
  return res.json();
}

export const api = {
  login: (username: string, password: string) =>
    request<{ access_token: string; refresh_token: string; user: User }>(
      "/auth/login",
      {
        method: "POST",
        body: JSON.stringify({ username, password }),
      },
    ),

  register: (username: string, email: string, password: string) =>
    request<{ access_token: string; refresh_token: string; user: User }>(
      "/auth/register",
      {
        method: "POST",
        body: JSON.stringify({ username, email, password }),
      },
    ),

  logout: () =>
    request<{ success: boolean }>("/auth/logout", { method: "POST" }),

  me: () => request<User>("/auth/me"),

  completeOnboarding: () =>
    request<{ success: boolean }>("/auth/onboarded", { method: "POST" }),

  getLibrary: () =>
    request<{ songs: Song[]; albums: Album[]; artists: Artist[] }>(
      `/library`,
    ),

  getSong: (id: number) => request<Song>(`/songs/${id}`),

  getAlbum: (id: number) =>
    request<{ id: number; title: string; year?: number; cover_path?: string; mbid?: string; artist?: Artist; songs: Song[]; created_at?: string }>(`/albums/${id}`),

  getArtist: (id: number) =>
    request<{ id: number; name: string; bio?: string; image_path?: string; mbid?: string; albums: Album[]; songs: Song[] }>(
      `/artists/${id}`,
    ),

  search: (q: string, limit = 25, offset = 0) =>
    request<SearchResults>(
      `/search?q=${encodeURIComponent(q)}&limit=${limit}&offset=${offset}`,
    ),

  getArtworkUrl: (id: number, type?: 'song' | 'album') => {
    let url = `${getApiUrl()}/artwork/${id}`;
    if (type === 'album') url += '?type=album';
    return url;
  },

  getStreamUrl: (id: number, format?: string, bitrate?: number) => {
    const token = getAccessToken();
    let url = `${getApiUrl()}/stream/${id}`;
    const params: string[] = [];
    if (token) params.push(`token=${token}`);
    if (format) params.push(`format=${format}`);
    if (bitrate) params.push(`bitrate=${bitrate}`);
    if (params.length) url += "?" + params.join("&");
    return url;
  },

  getLyrics: (id: number) =>
    request<{ lyrics?: string; synced?: string }>(`/lyrics/${id}`),

  getStreamingOptions: () => request<StreamingOptions>("/streaming/options"),

  getSettings: () => request<{ shuffle: boolean; repeat: string }>("/settings"),

  updateSettings: (settings: { shuffle: boolean; repeat: string }) =>
    request<{ shuffle: boolean; repeat: string }>("/settings", {
      method: "PUT",
      body: JSON.stringify(settings),
    }),

  getPlayerState: () =>
    request<{ current_song_id: number | null; queue: number[]; queue_index: number; progress: number }>("/player/state"),

  savePlayerState: (state: { current_song_id: number | null; queue: number[]; queue_index: number; progress: number }) =>
    request<{ success: boolean }>("/player/state", {
      method: "PUT",
      body: JSON.stringify(state),
    }),

  getPlaylists: (limit = 50, offset = 0) =>
    request<Playlist[]>(`/playlists?limit=${limit}&offset=${offset}`),

  getPlaylist: (id: number) => request<Playlist>(`/playlists/${id}`),

  createPlaylist: (name: string, description?: string, isPublic = false) =>
    request<Playlist>("/playlists", {
      method: "POST",
      body: JSON.stringify({ name, description, public: isPublic }),
    }),

  updatePlaylist: (
    id: number,
    name: string,
    description?: string,
    isPublic?: boolean,
  ) =>
    request<Playlist>(`/playlists/${id}`, {
      method: "PUT",
      body: JSON.stringify({ name, description, public: isPublic }),
    }),

  uploadPlaylistCover: async (id: number, file: File): Promise<{ cover_path: string }> => {
    const formData = new FormData();
    formData.append("cover", file);
    const token = getAccessToken();
    const headers: Record<string, string> = {};
    if (token) headers["Authorization"] = `Bearer ${token}`;
    const res = await fetch(`${getApiUrl()}/playlists/${id}/cover`, {
      method: "POST",
      headers,
      body: formData,
    });
    if (!res.ok) {
      const error = await res.json().catch(() => ({ error: res.statusText }));
      throw new Error(error.error || error.message || res.statusText);
    }
    return res.json();
  },

  getPlaylistCoverUrl: (id: number) => `${getApiUrl()}/playlists/${id}/cover`,

  deletePlaylist: (id: number) =>
    request<{ success: boolean }>(`/playlists/${id}`, { method: "DELETE" }),

  addSongToPlaylist: (playlistId: number, songId: number) =>
    request<{ success: boolean }>(`/playlists/${playlistId}/songs`, {
      method: "POST",
      body: JSON.stringify({ song_id: songId }),
    }),

  removeSongFromPlaylist: (playlistId: number, songId: number) =>
    request<{ success: boolean }>(`/playlists/${playlistId}/songs/${songId}`, {
      method: "DELETE",
    }),

  reorderPlaylist: (playlistId: number, songIds: number[]) =>
    request<{ success: boolean }>(`/playlists/${playlistId}/reorder`, {
      method: "PUT",
      body: JSON.stringify({ song_ids: songIds }),
    }),

  getFavorites: () =>
    request<{ songs: Song[]; albums: Album[]; artists: Artist[] }>(
      "/favorites",
    ),

  favoriteSong: (id: number) =>
    request<{ success: boolean }>(`/favorites/songs/${id}`, { method: "POST" }),

  unfavoriteSong: (id: number) =>
    request<{ success: boolean }>(`/favorites/songs/${id}`, {
      method: "DELETE",
    }),

  favoriteAlbum: (id: number) =>
    request<{ success: boolean }>(`/favorites/albums/${id}`, {
      method: "POST",
    }),

  unfavoriteAlbum: (id: number) =>
    request<{ success: boolean }>(`/favorites/albums/${id}`, {
      method: "DELETE",
    }),

  followArtist: (id: number) =>
    request<{ success: boolean }>(`/follows/artists/${id}`, { method: "POST" }),

  unfollowArtist: (id: number) =>
    request<{ success: boolean }>(`/follows/artists/${id}`, {
      method: "DELETE",
    }),

  getHistory: (limit = 50, offset = 0) =>
    request<PlayHistory[]>(`/history?limit=${limit}&offset=${offset}`),

  recordPlay: (
    songId: number,
    durationListened: number,
    completionRate: number,
    source = "web",
  ) =>
    request<{ success: boolean }>("/history", {
      method: "POST",
      body: JSON.stringify({
        song_id: songId,
        duration_listened: durationListened,
        completion_rate: completionRate,
        source,
        timestamp: Math.floor(Date.now() / 1000),
      }),
    }),

  getHome: () =>
    request<{
      recent_plays: Song[];
      recommendations: Song[];
      new_additions: Album[];
    }>("/home"),

  getStats: (period = "all_time") => request<Stats>(`/stats?period=${period}`),

  getWrapped: (period = "year") =>
    request<WrappedData>(`/stats/wrapped?period=${period}`),

  getInsights: () => request<Insights>("/stats/insights"),

  startScan: () => request<ScanJob>("/scan", { method: "POST" }),

  getScanStatus: () => request<ScanJob>("/scan/status"),

  getSystemInfo: () => request<SystemInfo>("/admin/system"),

  cleanupSessions: (olderThanDays: number) =>
    request<{ deleted: number }>("/admin/sessions/cleanup", {
      method: "DELETE",
      body: JSON.stringify({ older_than_days: olderThanDays }),
    }),

  health: () => request<{ status: string }>("/health"),

  getRadio: (songId: number, limit = 20, mode: 'curator' | 'mainstream' = 'curator') =>
    request<{ songs: Song[] }>(`/radio/${songId}?limit=${limit}&mode=${mode}`),

  getAppSettings: () =>
    request<{ radio_enabled: boolean }>("/admin/settings"),

  updateAppSettings: (settings: { radio_enabled?: boolean }) =>
    request<{ radio_enabled: boolean }>("/admin/settings", {
      method: "PUT",
      body: JSON.stringify(settings),
    }),
};
