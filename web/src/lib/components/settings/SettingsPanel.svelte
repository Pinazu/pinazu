<script lang="ts">
	import { onMount } from "svelte";
	import { theme } from "$lib/stores/theme";
	import { goto } from '$app/navigation';
	import XIcon from "$lib/icons/XIcon.svelte";
	import { SETTING_NAV } from "$lib/types/settings";
	import { Toggle } from "flowbite-svelte";
    import ThemeToggle from "./ThemeToggle.svelte";
    import LanguageSelect from "./LanguageSelect.svelte";
    import VoiceLanguageSelect from "./VoiceLanguageSelect.svelte";
    import ResponseNotification from "./ResponseNotification.svelte";
    import MoreIcon from "$lib/icons/MoreIcon.svelte";
	
	interface Props {
		hash: string;
	}

	let { hash }: Props = $props();

	let overlayRef: HTMLDivElement | undefined = $state();

	const isOpen = $derived(hash.startsWith('#settings'));
	const settingPanel = $derived(hash.substring(10) || 'General');

	// Close the panel
	const closePanel = () => {
		goto(window.location.pathname + window.location.search, { replaceState: true });
	};

	// Handle click outside to close panel
	const handleOverlayClick = (event: MouseEvent) => {
		if (event.target === overlayRef) {
			closePanel();
		}
	};

	// Handle escape key to close panel
	onMount(() => {
		const handleEscape = (event: KeyboardEvent) => {
			if (event.key === 'Escape' && isOpen) {
				closePanel();
			}
		};

		document.addEventListener('keydown', handleEscape);

		return () => {
			document.removeEventListener('keydown', handleEscape);
		};
	});
</script>

{#if isOpen}
	<div 
		bind:this={overlayRef}
		onclick={handleOverlayClick}
		onkeydown={(e) => e.key === 'Enter' && e.target === overlayRef && closePanel()}
		role="dialog"
		aria-modal="true"
		aria-labelledby="settings-title"
		tabindex="-1"
		class="fixed inset-0 z-[100] flex items-center justify-center bg-black/20 backdrop-blur-[1px] transition-all duration-500"
	>
		<div class="bg-[var(--bg-secondary)] flex rounded-2xl shadow-xs border border-[var(--border-color)] max-w-2xl w-full h-full max-h-[70vh] overflow-hidden mx-6">
			{@render leftNavBar()}
			<div class="flex-1 p-3 mx-1.5">
				{#if settingPanel === 'General'}
					{@render generalSettings()}
				{:else if settingPanel === 'Notifications'}
					{@render notificationsSettings()}
				{:else if settingPanel === 'Security'}
					{@render securitySettings()}
				{:else if settingPanel === 'Account'}
					{@render accountSettings()}
				{:else if settingPanel === 'About'}
					{@render aboutSettings()}
				{/if}
			</div>
		</div>
	</div>
{/if}

{#snippet leftNavBar()}
	<div class="flex flex-col bg-[var(--bg-primary)]/50 p-2 rounded-l-2xl">
		<button 
			class="text-[var(--text-primary)] hover:bg-[var(--bg-third)]/50 rounded-full mb-3 p-2 cursor-pointer select-none" 
			style="width: 2rem;"
			onclick={closePanel}
			aria-label="Close settings"
		>
			<XIcon />
		</button>
		{#each Object.entries(SETTING_NAV) as [key, item] (key)}
			{@const IconComponent = item.icon}
			<button
				class="flex text-[var(--text-primary)] hover:bg-[var(--bg-third)]/70 rounded-lg px-3 py-2 {settingPanel === key ? 'bg-[var(--bg-third)]/70' : ''} cursor-pointer gap-2"
				onclick={() => window.location.hash = `#settings/${key}`}
				aria-pressed={settingPanel === key}
			>
				<div class="flex justify-center items-center">
					<IconComponent />
				</div>
				<div class="text-[0.9rem] text-center justify-center">
					{item.label}
				</div>
			</button>
		{/each}
	</div>
{/snippet}

{#snippet generalSettings()}
	<div class="flex flex-col">
		<p class="text-xl">General</p>
		<div class="mt-3 border-1 border-b border-[var(--border-color)]"></div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between items-center text-sm">
			<p>Theme</p>
			<ThemeToggle/>
		</div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between items-center text-sm">
			<p>Language</p>
			<LanguageSelect/>
		</div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between pb-2 text-sm items-center">
			<p>Spoken language</p>
			<VoiceLanguageSelect/>
		</div>
		<div class="flex w-4/5">
			<p class="text-xs text-[var(--text-secondary)]/60">
				For best results, select the language you mainly speak. If it's not listed, it may still be supported via auto-detection.
			</p>
		</div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between text-sm">
			<p>Show follow up suggestions in chats</p>
			<Toggle checked={true}></Toggle>
		</div>
	</div>
{/snippet}

{#snippet notificationsSettings()}
	<div class="flex flex-col">
		<p class="text-xl">Notifications</p>
		<div class="mt-3 border-1 border-b border-[var(--border-color)]"></div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between items-center text-sm">
			<p>Responses</p>
			<ResponseNotification/>
		</div>
		<div class="flex w-4/5">
			<div class="text-xs text-[var(--text-secondary)]/60">
				Get notified when the AI responds to requests that take time, like research or image generation.
			</div>
		</div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between pb-2 text-sm">
			<p>Tasks</p>
			<ResponseNotification/>
		</div>
		<div class="flex w-4/5">
			<div class="text-xs text-[var(--text-secondary)]/60">
				Get notified when tasks you’ve created have updates.
				<a href="/settings/tasks" class="text-underline">Manage tasks</a>
			</div>
		</div>
	</div>
{/snippet}

{#snippet securitySettings()}
	<div class="flex flex-col">
		<p class="text-xl">Security</p>
		<div class="mt-3 border-1 border-b border-[var(--border-color)]"></div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between text-sm pb-2">
			<p>Multi-factor authentication</p>
			<Toggle checked={false}></Toggle>
		</div>
		<div class="text-xs text-[var(--text-secondary)]/60">
			Require an extra security challenge when logging in. If you are unable to pass this challenge, you will have the option to recover your account via email.
		</div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between text-sm items-center">
			<p>Log out of this device</p>
			{@render logoutButton()}
		</div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between text-sm pb-2 items-center">
			<p>Log out of all devices</p>
			{@render logoutAllButton()}
		</div>
		<div class="flex w-4/5">
			<div class="text-xs text-[var(--text-secondary)]/60">
				Log out of all active sessions across all devices, including your current session. It may take up to 30 minutes for other devices to be logged out.
			</div>
		</div>
	</div>
{/snippet}

{#snippet accountSettings()}
	<div class="flex flex-col">
		<p class="text-xl">Account</p>
		<div class="mt-3 border-1 border-b border-[var(--border-color)]"></div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<div class="flex justify-between pb-2 text-sm items-center">
			<p>Delete account</p>
			{@render deleteAccountButton()}
		</div>
		<div class="text-xs text-[var(--text-secondary)]/60">
			Delete all data associated with your account. This action is irreversible.
		</div>
	</div>
{/snippet}

{#snippet aboutSettings()}
	<div class="flex flex-col">
		<p class="text-xl">About</p>
		<div class="mt-3 border-1 border-b border-[var(--border-color)]"></div>
	</div>
	<div class="flex flex-col border-b border-[var(--border-color)]/80 py-4">
		<a 
			href="https://youtu.be/QDia3e12czc"
			target="_blank"
			class="flex justify-between pb-2 text-sm items-center"
			aria-label="Watch the demo video"
		>
			<div class="flex flex-col gap-2">
				<p>Demo video</p>
				<div class="text-xs text-[var(--text-secondary)]/60">
					What the introduction video about our team and the product.
				</div>
			</div>
			<div class="flex items-center mr-10">
				<MoreIcon/>
			</div>
		</a>	
	</div>
{/snippet}

{#snippet logoutButton()}
	<button
		type="button"
		onclick={() => {}}
		class="
			bg-transparent hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			px-3 py-1.5
			rounded-2xl
			cursor-pointer
			text-[0.85rem]
			border border-[var(--border-color)]/50
			hover:border-[var(--border-color)]
			transition-all duration-200
			hover:scale-105
		"
	>
		Log out
	</button>
{/snippet}

{#snippet logoutAllButton()}
	<button
		type="button"
		onclick={() => {}}
		class="
			bg-transparent
			{$theme === 'dark' ? 'hover:bg-red-500/20 text-red-500 hover:text-red-600' : 'hover:bg-red-500/10 text-red-500 hover:text-red-600'}
			px-3 py-1.5
			rounded-2xl
			cursor-pointer
			text-[0.85rem]
			border border-red-500/30
			hover:border-red-500/50
			transition-all duration-200
			hover:scale-105
		"
	>
		Log out all
	</button>
{/snippet}

{#snippet deleteAccountButton()}
	<button
		type="button"
		onclick={() => {}}
		class="
			bg-transparent 
			{$theme === 'dark' ? 'hover:bg-red-500/20 text-red-500 hover:text-red-600' : 'hover:bg-red-500/10 text-red-500 hover:text-red-600'}
			px-3 py-1.5
			rounded-2xl
			cursor-pointer
			text-[0.85rem]
			border border-red-500/30
			hover:border-red-500/50
			transition-all duration-200
			hover:scale-105
		"
	>
		Delete
	</button>
{/snippet}
