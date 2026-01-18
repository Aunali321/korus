<script lang="ts">
	import { page } from "$app/stores";
	import {
		Home,
		Search,
		Library,
		ListMusic,
		Disc3,
		Mic2,
		Heart,
		Settings,
		BarChart3,
		Sparkles,
		Shield,
		X,
	} from "lucide-svelte";
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
		class="fixed inset-0 bg-black/50 z-40 md:hidden"
		onclick={onClose}
		aria-label="Close menu"
	></button>
{/if}

<!-- Sidebar -->
<div class="
	fixed inset-y-0 left-0 z-50 w-64 bg-zinc-950 border-r border-zinc-800 flex flex-col h-full
	transform transition-transform duration-300 ease-in-out
	md:relative md:translate-x-0
	{isOpen ? 'translate-x-0' : '-translate-x-full'}
">
	<div class="p-6 border-b border-zinc-800 flex items-center justify-between">
		<div>
			<h1
				class="text-2xl font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent"
			>
				Korus
			</h1>
			<p class="text-xs text-zinc-500 mt-1">Self-hosted Music</p>
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
				<a
					href={item.href}
					onclick={handleNavClick}
					class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(
						item.href,
					)
						? 'bg-zinc-800 text-emerald-400'
						: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
				>
					<item.icon size={20} />
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
					<a
						href={item.href}
						onclick={handleNavClick}
						class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(
							item.href,
						)
							? 'bg-zinc-800 text-emerald-400'
							: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
					>
						<item.icon size={18} />
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
					class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(
						'/stats',
					)
						? 'bg-zinc-800 text-emerald-400'
						: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
				>
					<BarChart3 size={18} />
					<span class="text-sm">Stats</span>
				</a>
				{#if isWrappedSeason()}
					<a
						href="/wrapped"
						onclick={handleNavClick}
						class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(
							'/wrapped',
						)
							? 'bg-zinc-800 text-emerald-400'
							: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
					>
						<Sparkles size={18} />
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
				class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(
					'/admin',
				)
					? 'bg-zinc-800 text-emerald-400'
					: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
			>
				<Shield size={18} />
				<span class="text-sm">Admin</span>
			</a>
		{/if}
		<a
			href="/settings"
			onclick={handleNavClick}
			class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(
				'/settings',
			)
				? 'bg-zinc-800 text-emerald-400'
				: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
		>
			<Settings size={18} />
			<span class="text-sm">Settings</span>
		</a>
	</div>
</div>
