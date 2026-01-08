<script lang="ts">
	import CommandPalette, { defineActions } from 'svelte-command-palette';
	import { goto } from '$app/navigation';
	import { player } from '$lib/stores/player.svelte';
	import { settings } from '$lib/stores/settings.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { favorites } from '$lib/stores/favorites.svelte';
	import { onMount } from 'svelte';

	interface Props {
		onToggleQueue: () => void;
		onToggleLyrics: () => void;
	}

	let { onToggleQueue, onToggleLyrics }: Props = $props();

	function toggleCurrentFavorite() {
		if (player.currentSong) {
			favorites.toggle(player.currentSong.id);
		}
	}

	function isEditableElement(el: EventTarget | null): boolean {
		if (!el || !(el instanceof HTMLElement)) return false;
		const tag = el.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
		if (el.isContentEditable) return true;
		return false;
	}

	onMount(() => {
		function handleKeydown(e: KeyboardEvent) {
			if (isEditableElement(e.target)) return;

			if (e.code === 'Space') {
				e.preventDefault();
				player.toggle();
			} else if (e.code === 'ArrowLeft' && !e.shiftKey) {
				e.preventDefault();
				player.seek(Math.max(0, player.progress - 10));
			} else if (e.code === 'ArrowRight' && !e.shiftKey) {
				e.preventDefault();
				player.seek(player.progress + 10);
			}
		}
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	function volumeUp() {
		player.setVolume(Math.min(1, player.volume + 0.1));
	}

	function volumeDown() {
		player.setVolume(Math.max(0, player.volume - 0.1));
	}

	const actions = defineActions([
		// Navigation
		{
			title: 'Go to Home',
			subTitle: 'View your home page',
			onRun: () => goto('/'),
			shortcut: '$mod+0',
			icon: '~',
			group: 'Navigation',
			keywords: ['home', 'main', 'dashboard'],
		},
		{
			title: 'Go to Search',
			subTitle: 'Search your music library',
			onRun: () => goto('/search'),
			shortcut: '$mod+1',
			icon: '/',
			group: 'Navigation',
			keywords: ['search', 'find', 'query'],
		},
		{
			title: 'Go to Library',
			subTitle: 'Browse all your music',
			onRun: () => goto('/library'),
			shortcut: '$mod+2',
			icon: '=',
			group: 'Navigation',
			keywords: ['library', 'songs', 'all'],
		},
		{
			title: 'Go to Playlists',
			subTitle: 'View your playlists',
			onRun: () => goto('/playlists'),
			shortcut: '$mod+3',
			icon: '#',
			group: 'Navigation',
			keywords: ['playlists', 'lists'],
		},
		{
			title: 'Go to Albums',
			subTitle: 'Browse albums',
			onRun: () => goto('/albums'),
			shortcut: '$mod+4',
			icon: '@',
			group: 'Navigation',
			keywords: ['albums', 'records'],
		},
		{
			title: 'Go to Artists',
			subTitle: 'Browse artists',
			onRun: () => goto('/artists'),
			shortcut: '$mod+5',
			icon: '&',
			group: 'Navigation',
			keywords: ['artists', 'musicians'],
		},
		{
			title: 'Go to Favorites',
			subTitle: 'View your liked songs',
			onRun: () => goto('/favorites'),
			shortcut: '$mod+6',
			icon: '*',
			group: 'Navigation',
			keywords: ['favorites', 'liked', 'hearts'],
		},
		{
			title: 'Go to Stats',
			subTitle: 'View listening statistics',
			onRun: () => goto('/stats'),
			shortcut: '$mod+7',
			icon: '%',
			group: 'Navigation',
			keywords: ['stats', 'statistics', 'charts'],
		},
		{
			title: 'Go to Settings',
			subTitle: 'Configure your preferences',
			onRun: () => goto('/settings'),
			shortcut: '$mod+8',
			icon: ':',
			group: 'Navigation',
			keywords: ['settings', 'preferences', 'config'],
		},
		{
			title: 'Go to Admin',
			subTitle: 'Admin panel',
			onRun: () => goto('/admin'),
			shortcut: '$mod+9',
			icon: '!',
			group: 'Navigation',
			keywords: ['admin', 'manage'],
			canActionRun: () => auth.isAdmin,
		},
		// Playback
		{
			title: 'Play / Pause',
			subTitle: 'Toggle playback (Space)',
			onRun: () => player.toggle(),
			icon: '>',
			group: 'Playback',
			keywords: ['play', 'pause', 'toggle'],
		},
		{
			title: 'Next Track',
			subTitle: 'Skip to next song',
			onRun: () => player.next(),
			shortcut: 'Shift+ArrowRight',
			icon: '>|',
			group: 'Playback',
			keywords: ['next', 'skip', 'forward'],
		},
		{
			title: 'Previous Track',
			subTitle: 'Go to previous song',
			onRun: () => player.prev(),
			shortcut: 'Shift+ArrowLeft',
			icon: '|<',
			group: 'Playback',
			keywords: ['previous', 'back'],
		},
		{
			title: 'Seek Forward',
			subTitle: 'Skip forward 10 seconds (Right)',
			onRun: () => player.seek(player.progress + 10),
			icon: '>>',
			group: 'Playback',
			keywords: ['seek', 'forward', 'skip'],
		},
		{
			title: 'Seek Backward',
			subTitle: 'Skip backward 10 seconds (Left)',
			onRun: () => player.seek(Math.max(0, player.progress - 10)),
			icon: '<<',
			group: 'Playback',
			keywords: ['seek', 'backward', 'rewind'],
		},
		{
			title: 'Toggle Shuffle',
			subTitle: 'Turn shuffle on/off',
			onRun: () => player.toggleShuffle(),
			shortcut: 'Shift+S',
			icon: 'X',
			group: 'Playback',
			keywords: ['shuffle', 'random'],
		},
		{
			title: 'Toggle Repeat',
			subTitle: 'Cycle repeat modes',
			onRun: () => player.toggleRepeat(),
			shortcut: 'Shift+R',
			icon: 'O',
			group: 'Playback',
			keywords: ['repeat', 'loop'],
		},
		{
			title: 'Volume Up',
			subTitle: 'Increase volume by 10%',
			onRun: () => volumeUp(),
			shortcut: 'Shift+Equal',
			icon: '+',
			group: 'Playback',
			keywords: ['volume', 'up', 'louder'],
		},
		{
			title: 'Volume Down',
			subTitle: 'Decrease volume by 10%',
			onRun: () => volumeDown(),
			shortcut: 'Shift+Minus',
			icon: '-',
			group: 'Playback',
			keywords: ['volume', 'down', 'quieter'],
		},
		// View
		{
			title: 'Toggle Queue',
			subTitle: 'Show/hide the play queue',
			onRun: () => onToggleQueue(),
			shortcut: 'Shift+Q',
			icon: '#',
			group: 'View',
			keywords: ['queue', 'upcoming'],
		},
		{
			title: 'Toggle Lyrics',
			subTitle: 'Show/hide lyrics panel',
			onRun: () => onToggleLyrics(),
			shortcut: 'Shift+L',
			icon: '"',
			group: 'View',
			keywords: ['lyrics', 'words'],
		},
		{
			title: 'Favorite Current Song',
			subTitle: 'Add or remove current song from favorites',
			onRun: () => toggleCurrentFavorite(),
			shortcut: 'Shift+F',
			icon: '*',
			group: 'Playback',
			keywords: ['favorite', 'like', 'heart', 'love'],
			canActionRun: () => !!player.currentSong,
		},
		// Account
		{
			title: 'Logout',
			subTitle: 'Sign out of your account',
			onRun: () => auth.logout(),
			icon: '<-',
			group: 'Account',
			keywords: ['logout', 'signout', 'exit'],
		},
	]);
</script>

<CommandPalette
	commands={actions}
	placeholder="Search commands..."
	shortcut="$mod+k"
	unstyled={false}
	overlayClass="!bg-black/70 !backdrop-blur-sm"
	paletteWrapperInnerClass="!bg-zinc-900 !border !border-zinc-700 !rounded-xl !shadow-2xl !max-w-lg"
	inputClass="!bg-transparent !text-zinc-100 !placeholder-zinc-500 !border-b !border-zinc-700 !rounded-none"
	resultsContainerClass="!bg-transparent !max-h-80"
	resultContainerClass="!text-zinc-300 hover:!bg-zinc-800 !rounded-lg !mx-2 !my-0.5"
	optionSelectedClass="!bg-zinc-800 !text-emerald-400"
	titleClass="!text-zinc-100 !font-medium"
	subtitleClass="!text-zinc-500 !text-sm"
	keyboardButtonClass="!bg-zinc-800 !text-zinc-400 !border-zinc-600 !rounded !text-xs !px-1.5 !py-0.5"
/>
