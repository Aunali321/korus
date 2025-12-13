<script lang="ts">
	import "../app.css";
	import { onMount } from "svelte";
	import { goto } from "$app/navigation";
	import { page } from "$app/stores";
	import { auth } from "$lib/stores/auth.svelte";
	import Sidebar from "$lib/components/Sidebar.svelte";
	import Player from "$lib/components/Player.svelte";
	import Queue from "$lib/components/Queue.svelte";
	import Toast from "$lib/components/Toast.svelte";

	let { children } = $props();
	let showQueue = $state(false);

	const publicRoutes = ["/login", "/register", "/setup"];

	onMount(async () => {
		await auth.init();
	});

	$effect(() => {
		if (!auth.isLoading && !auth.isAuthenticated) {
			const isPublic = publicRoutes.some((r) =>
				$page.url.pathname.startsWith(r),
			);
			if (!isPublic) {
				goto("/login");
			}
		}
	});
</script>

<svelte:head>
	<title>Korus</title>
	<meta name="description" content="Self-hosted music streaming" />
</svelte:head>

{#if auth.isLoading}
	<div class="h-screen bg-black flex items-center justify-center">
		<div class="text-zinc-400">Loading...</div>
	</div>
{:else if auth.isAuthenticated}
	<div
		class="h-screen bg-black text-zinc-100 flex flex-col overflow-hidden font-sans"
	>
		<div class="flex flex-1 overflow-hidden">
			<Sidebar />
			<main
				class="flex-1 overflow-y-auto scrollbar-thin bg-gradient-to-b from-zinc-900 to-black"
			>
				{@render children()}
			</main>
			<Queue isOpen={showQueue} onClose={() => (showQueue = false)} />
		</div>
		<Player onToggleQueue={() => (showQueue = !showQueue)} />
		<Toast />
	</div>
{:else}
	<div class="min-h-screen bg-black text-zinc-100 font-sans">
		{@render children()}
		<Toast />
	</div>
{/if}
