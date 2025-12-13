<script lang="ts">
	import { page } from '$app/stores';
	import { Home, Search, Library, ListMusic, Disc3, Mic2, Heart, Settings, BarChart3, Clock, Users, Sparkles, Lightbulb, Shield } from 'lucide-svelte';
	import { auth } from '$lib/stores/auth.svelte';

	const navItems = [
		{ icon: Home, label: 'Home', href: '/' },
		{ icon: Search, label: 'Search', href: '/search' },
		{ icon: Library, label: 'Library', href: '/library' }
	];

	const libraryItems = [
		{ icon: ListMusic, label: 'Playlists', href: '/playlists' },
		{ icon: Disc3, label: 'Albums', href: '/albums' },
		{ icon: Mic2, label: 'Artists', href: '/artists' },
		{ icon: Heart, label: 'Favorites', href: '/favorites' }
	];

	const statsItems = [
		{ icon: Clock, label: 'History', href: '/history' },
		{ icon: BarChart3, label: 'Stats', href: '/stats' },
		{ icon: Sparkles, label: 'Wrapped', href: '/wrapped' },
		{ icon: Lightbulb, label: 'Insights', href: '/insights' },
		{ icon: Users, label: 'Social', href: '/social' }
	];

	function isActive(href: string): boolean {
		if (href === '/') return $page.url.pathname === '/';
		return $page.url.pathname.startsWith(href);
	}
</script>

<div class="w-64 bg-zinc-950 border-r border-zinc-800 flex flex-col h-full">
	<div class="p-6 border-b border-zinc-800">
		<h1 class="text-2xl font-bold bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text text-transparent">
			Korus
		</h1>
		<p class="text-xs text-zinc-500 mt-1">Self-hosted Music</p>
	</div>

	<nav class="flex-1 overflow-y-auto scrollbar-thin">
		<div class="p-3 space-y-1">
			{#each navItems as item}
				<a
					href={item.href}
					class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(item.href)
						? 'bg-zinc-800 text-emerald-400'
						: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
				>
					<item.icon size={20} />
					<span class="font-medium">{item.label}</span>
				</a>
			{/each}
		</div>

		<div class="mt-6 px-3">
			<h2 class="text-xs font-semibold text-zinc-500 uppercase tracking-wider px-3 py-2">
				Library
			</h2>
			<div class="space-y-1 mt-1">
				{#each libraryItems as item}
					<a
						href={item.href}
						class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(item.href)
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
			<h2 class="text-xs font-semibold text-zinc-500 uppercase tracking-wider px-3 py-2">
				Stats
			</h2>
			<div class="space-y-1 mt-1">
				{#each statsItems as item}
					<a
						href={item.href}
						class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive(item.href)
							? 'bg-zinc-800 text-emerald-400'
							: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
					>
						<item.icon size={18} />
						<span class="text-sm">{item.label}</span>
					</a>
				{/each}
			</div>
		</div>
	</nav>

	<div class="p-3 border-t border-zinc-800 space-y-1">
		{#if auth.isAdmin}
			<a
				href="/admin"
				class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive('/admin')
					? 'bg-zinc-800 text-emerald-400'
					: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
			>
				<Shield size={18} />
				<span class="text-sm">Admin</span>
			</a>
		{/if}
		<a
			href="/settings"
			class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all {isActive('/settings')
				? 'bg-zinc-800 text-emerald-400'
				: 'text-zinc-400 hover:text-zinc-100 hover:bg-zinc-900'}"
		>
			<Settings size={18} />
			<span class="text-sm">Settings</span>
		</a>
	</div>
</div>
