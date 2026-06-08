// Holds the Ask chat transcript at module scope so it survives client-side
// navigation (the /ask page component unmounts on route change).

export type AskPart =
	| { kind: 'text'; text: string }
	| { kind: 'ui'; spec: any }
	| { kind: 'action'; label: string }
	| { kind: 'tool'; label: string };

export type AskMsg = { role: 'user' | 'assistant'; parts: AskPart[] };

function createAskStore() {
	let messages = $state<AskMsg[]>([]);
	let conversationId = $state(0);
	let busy = $state(false);

	return {
		get messages() {
			return messages;
		},
		get conversationId() {
			return conversationId;
		},
		set conversationId(v: number) {
			conversationId = v;
		},
		get busy() {
			return busy;
		},
		set busy(v: boolean) {
			busy = v;
		},
		reset() {
			messages = [];
			conversationId = 0;
			busy = false;
		},
	};
}

export const ask = createAskStore();
