<script lang="ts">
	import { page } from "$app/stores";
	import { fade } from "svelte/transition";
	import Home from "@lucide/svelte/icons/home";
	import Search from "@lucide/svelte/icons/search";
	import Library from "@lucide/svelte/icons/library";
	import ListMusic from "@lucide/svelte/icons/list-music";
	import Disc3 from "@lucide/svelte/icons/disc-3";
	import Mic2 from "@lucide/svelte/icons/mic-2";
	import Heart from "@lucide/svelte/icons/heart";
	import Settings from "@lucide/svelte/icons/settings";
	import BarChart3 from "@lucide/svelte/icons/bar-chart-3";
	import Sparkles from "@lucide/svelte/icons/sparkles";
	import Shield from "@lucide/svelte/icons/shield";
	import X from "@lucide/svelte/icons/x";
	import { auth } from "$lib/stores/auth.svelte";

	let { isOpen = false, onClose = () => {} }: { isOpen?: boolean; onClose?: () => void } = $props();

	const navItems = [
		{ icon: Home, label: "Home", href: "/" },
		{ icon: Search, label: "Search", href: "/search" },
		{ icon: Library, label: "Library", href: "/library" },
	];

	const libraryItems = [
		{ icon: ListMusic, label: "Playlists", href: "/playlists" },
		{ icon: Disc3, label: "Albums", href: "/albums" },
		{ icon: Mic2, label: "Artists", href: "/artists" },
		{ icon: Heart, label: "Favorites", href: "/favorites" },
	];

	// Wrapped is shown only during last week of month or in December
	function isWrappedSeason(): boolean {
		const now = new Date();
		const month = now.getMonth(); // 0-11
		const date = now.getDate();
		const lastDay = new Date(now.getFullYear(), month + 1, 0).getDate();

		// Show in December (month 11) or last 7 days of any month
		return month === 11 || lastDay - date < 7;
	}

	function isActive(href: string): boolean {
		if (href === "/") return $page.url.pathname === "/";
		return $page.url.pathname.startsWith(href);
	}

	function handleNavClick() {
		onClose();
	}
</script>

<!-- Mobile overlay -->
{#if isOpen}
	<button
		transition:fade={{ duration: 200 }}
		class="fixed inset-0 bg-black/50 z-40 md:hidden"
		onclick={onClose}
		aria-label="Close menu"
	></button>
{/if}

<!-- Sidebar -->
<div class="
	fixed inset-y-0 left-0 z-50 w-64 bg-zinc-950 border-r border-zinc-800 flex flex-col h-full
	transform transition-transform duration-[420ms] ease-[cubic-bezier(0.32,0.72,0,1)] will-change-transform
	md:relative md:translate-x-0
	{isOpen ? 'translate-x-0' : '-translate-x-full'}
">
	<div class="p-6 border-b border-zinc-800 flex items-center justify-between">
		<div class="flex items-center gap-2">
			<img src="/logo.svg" alt="Korus" class="w-7 h-7" />
			<h1
				class="text-2xl font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent"
			>
				Korus
			</h1>
		</div>
		<button
			onclick={onClose}
			class="p-2 hover:bg-zinc-800 rounded-lg transition-colors md:hidden"
			aria-label="Close menu"
		>
			<X size={20} />
		</button>
	</div>

	<nav class="flex-1 overflow-y-auto scrollbar-thin">
		<div class="p-3 space-y-1">
			{#each navItems as item}
				{@const active = isActive(item.href)}
				<a
					href={item.href}
					onclick={handleNavClick}
					class="nav-link {active ? 'is-active' : ''}"
				>
					<span class="nav-indicator" aria-hidden="true"></span>
					<item.icon size={20} class="nav-icon" />
					<span class="font-medium">{item.label}</span>
				</a>
			{/each}
		</div>

		<div class="mt-6 px-3">
			<h2
				class="text-xs font-semibold text-zinc-500 uppercase tracking-wider px-3 py-2"
			>
				Library
			</h2>
			<div class="space-y-1 mt-1">
				{#each libraryItems as item}
					{@const active = isActive(item.href)}
					<a
						href={item.href}
						onclick={handleNavClick}
						class="nav-link nav-link--sm {active ? 'is-active' : ''}"
					>
						<span class="nav-indicator" aria-hidden="true"></span>
						<item.icon size={18} class="nav-icon" />
						<span class="text-sm">{item.label}</span>
					</a>
				{/each}
			</div>
		</div>

		<div class="mt-6 px-3">
			<h2
				class="text-xs font-semibold text-zinc-500 uppercase tracking-wider px-3 py-2"
			>
				Stats
			</h2>
			<div class="space-y-1 mt-1">
				<a
					href="/stats"
					onclick={handleNavClick}
					class="nav-link nav-link--sm {isActive('/stats') ? 'is-active' : ''}"
				>
					<span class="nav-indicator" aria-hidden="true"></span>
					<BarChart3 size={18} class="nav-icon" />
					<span class="text-sm">Stats</span>
				</a>
				{#if isWrappedSeason()}
					<a
						href="/wrapped"
						onclick={handleNavClick}
						class="nav-link nav-link--sm {isActive('/wrapped') ? 'is-active' : ''}"
					>
						<span class="nav-indicator" aria-hidden="true"></span>
						<Sparkles size={18} class="nav-icon" />
						<span class="text-sm">Wrapped</span>
					</a>
				{/if}
			</div>
		</div>
	</nav>

	<div class="p-3 border-t border-zinc-800 space-y-1">
		{#if auth.isAdmin}
			<a
				href="/admin"
				onclick={handleNavClick}
				class="nav-link nav-link--sm {isActive('/admin') ? 'is-active' : ''}"
			>
				<span class="nav-indicator" aria-hidden="true"></span>
				<Shield size={18} class="nav-icon" />
				<span class="text-sm">Admin</span>
			</a>
		{/if}
		<a
			href="/settings"
			onclick={handleNavClick}
			class="nav-link nav-link--sm {isActive('/settings') ? 'is-active' : ''}"
		>
			<span class="nav-indicator" aria-hidden="true"></span>
			<Settings size={18} class="nav-icon" />
			<span class="text-sm">Settings</span>
		</a>
	</div>
</div>

<style>
	.nav-link {
		position: relative;
		display: flex;
		align-items: center;
		gap: 0.75rem;
		width: 100%;
		padding: 0.625rem 0.75rem;
		border-radius: 0.5rem;
		color: rgb(161 161 170);
		transition:
			color var(--dur-base) var(--ease-soft),
			background-color var(--dur-base) var(--ease-soft);
	}
	.nav-link:hover {
		color: rgb(244 244 245);
		background-color: rgb(24 24 27);
	}
	.nav-link.is-active {
		color: rgb(52 211 153);
		background-color: rgb(39 39 42);
	}

	.nav-indicator {
		position: absolute;
		left: 0;
		top: 50%;
		width: 3px;
		height: 60%;
		background: #10b981;
		border-radius: 0 3px 3px 0;
		transform: translateY(-50%) scaleY(0);
		transform-origin: center;
		opacity: 0;
		transition:
			transform var(--dur-base) var(--ease-out-expo),
			opacity var(--dur-fast) var(--ease-soft);
	}
	.nav-link.is-active .nav-indicator {
		transform: translateY(-50%) scaleY(1);
		opacity: 1;
	}
</style>
