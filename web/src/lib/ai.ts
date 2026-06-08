import { getAccessToken, getApiUrl } from './api';
import { api } from './api';
import { player } from './stores/player.svelte';
import type { Song } from './types';

export interface UINode {
	type?: string;
	props?: Record<string, any>;
	children?: UINode[];
}

export type ChatAction = {
	type: 'action';
	action: string;
	songs?: Song[];
	position?: 'next' | 'end';
	playlist_id?: number;
	indices?: number[];
	order?: number[];
	control?: string;
};

export type ChatEvent =
	| { type: 'text'; delta: string }
	| { type: 'tool'; name: string; phase: 'start' | 'end' }
	| { type: 'ui'; spec: any }
	| ChatAction
	| { type: 'done'; conversation_id: number }
	| { type: 'error'; message: string };

function authHeaders(): Record<string, string> {
	const token = getAccessToken();
	return token ? { Authorization: `Bearer ${token}` } : {};
}

// streamChat POSTs a message and parses the SSE response, invoking onEvent for
// each event as it arrives.
export interface PlayerContext {
	now_playing_id?: number;
	queue_ids?: number[];
}

export async function streamChat(
	message: string,
	conversationId: number,
	onEvent: (e: ChatEvent) => void,
	player?: PlayerContext,
	signal?: AbortSignal,
): Promise<void> {
	const res = await fetch(`${getApiUrl()}/ai/chat`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json', ...authHeaders() },
		body: JSON.stringify({
			message,
			conversation_id: conversationId || 0,
			now_playing_id: player?.now_playing_id ?? 0,
			queue_ids: player?.queue_ids ?? [],
		}),
		signal,
	});
	if (!res.ok || !res.body) throw new Error(`chat failed (${res.status})`);

	const reader = res.body.getReader();
	const decoder = new TextDecoder();
	let buf = '';
	for (;;) {
		const { done, value } = await reader.read();
		if (done) break;
		buf += decoder.decode(value, { stream: true });
		let idx: number;
		while ((idx = buf.indexOf('\n\n')) >= 0) {
			const block = buf.slice(0, idx);
			buf = buf.slice(idx + 2);
			const line = block.split('\n').find((l) => l.startsWith('data:'));
			if (!line) continue;
			try {
				onEvent(JSON.parse(line.slice(5).trim()));
			} catch {
				/* ignore malformed event */
			}
		}
	}
}

export interface Conversation {
	id: number;
	title: string;
	created_at: string;
	updated_at: string;
}

export interface AiMessage {
	role: string;
	content: string;
}

async function get<T>(path: string): Promise<T> {
	const res = await fetch(`${getApiUrl()}${path}`, { headers: authHeaders() });
	if (!res.ok) throw new Error(String(res.status));
	return res.json();
}

export const listConversations = () => get<{ conversations: Conversation[] }>('/ai/conversations');
export const getConversation = (id: number) => get<{ messages: AiMessage[] }>(`/ai/conversations/${id}`);
export const getWrapped = (period: 'month' | 'year' = 'month', refresh = false) =>
	get<{ html: string; period_type: string; period_key: string; cached: boolean }>(
		`/ai/wrapped?period=${period}${refresh ? '&refresh=1' : ''}`,
	);

export async function deleteConversation(id: number): Promise<void> {
	await fetch(`${getApiUrl()}/ai/conversations/${id}`, { method: 'DELETE', headers: authHeaders() });
}

// resolveSongs fetches full Song objects for the given IDs (UI specs only carry
// brief song data; playback needs the full object).
async function resolveSongs(ids: number[]): Promise<Song[]> {
	const out: Song[] = [];
	for (const id of ids) {
		try {
			out.push(await api.getSong(id));
		} catch {
			/* skip missing */
		}
	}
	return out;
}

export async function playSongIds(ids: number[]): Promise<void> {
	const songs = await resolveSongs(ids);
	if (songs.length) player.playQueue(songs);
}

export async function enqueueSongIds(ids: number[], position: 'next' | 'end' = 'end'): Promise<void> {
	const songs = await resolveSongs(ids);
	if (songs.length) player.enqueue(songs, position);
}

// runAction applies a chat action event (whose song payloads are already full
// Song objects, resolved server-side) to the player.
export function runAction(ev: ChatAction): void {
	switch (ev.action) {
		case 'play_now':
			if (ev.songs?.length) player.playQueue(ev.songs);
			break;
		case 'queue':
			if (ev.songs?.length) player.enqueue(ev.songs, ev.position === 'next' ? 'next' : 'end');
			break;
		case 'clear_queue':
			player.clearQueue();
			break;
		case 'remove_from_queue':
			[...(ev.indices ?? [])].sort((a, b) => b - a).forEach((i) => player.removeFromQueue(i));
			break;
		case 'reorder_queue':
			if (ev.order) player.reorderQueue(ev.order);
			break;
		case 'playback_control':
			if (ev.control === 'pause') player.pause();
			else if (ev.control === 'resume') player.play();
			else if (ev.control === 'next') player.next();
			else if (ev.control === 'previous') player.prev();
			break;
	}
}
