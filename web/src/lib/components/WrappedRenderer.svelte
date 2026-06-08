<script lang="ts">
	import { getApiUrl } from '$lib/api';

	// The LLM authors a complete, self-contained HTML diary page (a <style> block
	// plus markup, no scripts). We render it in a fully sandboxed iframe — no
	// scripts, no same-origin, no forms — so untrusted model HTML can't touch the
	// app, cookies, or run JS. Album covers load from the public /api/artwork/:id.
	let { html }: { html: string } = $props();

	// Wrap the body in a minimal document so it scales on mobile and sits on a dark
	// base, and point artwork at the configured API origin (the model emits the
	// relative "/api/artwork/<id>", which only resolves same-origin on its own).
	const srcdoc = $derived.by(() => {
		if (!html) return '';
		const base = getApiUrl(); // e.g. "/api" (same-origin) or "https://host/api"
		const body = base === '/api' ? html : html.replaceAll('/api/artwork/', `${base}/artwork/`);
		// NB: keep no literal <style> tag in this source — the Svelte preprocessor
		// would try to parse it as a component stylesheet. Base styles go inline.
		return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head><body style="margin:0;background:#060608;color:#fff">${body}</body></html>`;
	});
</script>

<iframe {srcdoc} title="Your Wrapped" sandbox="" class="block h-screen w-full border-0 bg-[#060608]"></iframe>
