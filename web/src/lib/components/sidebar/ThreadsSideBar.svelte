<script lang="ts">
	import { onMount } from 'svelte';
	import { gsap } from 'gsap';
    import { goto } from '$app/navigation';
    import { theme } from '$lib/stores/theme';
    import { threadStore, threads, threadsLoading, threadsError } from '$lib/stores/threads';
	// import { tasksStore, tasks, tasksLoading, tasksError } from '$lib/stores/tasks';
    import Tooltip from '../Tooltip.svelte';
    import MenuIcon from '$lib/icons/MenuIcon.svelte';
    import ManageToolIcon from '$lib/icons/ManageToolIcon.svelte';
    import LogoutIcon from '$lib/icons/LogoutIcon.svelte';
    import HelpIcon from '$lib/icons/HelpIcon.svelte';
    import RightArrowIcon from '$lib/icons/RightArrowIcon.svelte';
    import MoreIcon from '$lib/icons/MoreIcon.svelte';
    import BookIcon from '$lib/icons/BookIcon.svelte';
    import PhoneIcon from '$lib/icons/PhoneIcon.svelte';
    import KeyboardIcon from '$lib/icons/KeyboardIcon.svelte';
    import SearchIcon from '$lib/icons/SearchIcon.svelte';
    import VoiceIcon from '$lib/icons/VoiceIcon.svelte';
    import DocumentsIcon2 from '$lib/icons/DocumentsIcon2.svelte';
    import ClockIcon from '$lib/icons/ClockIcon.svelte';
    import ArrowDownIcon from '$lib/icons/ArrowDownIcon.svelte';
    import TaskIcon from '$lib/icons/TaskIcon.svelte';
    import AdminIcon from '$lib/icons/AdminIcon.svelte';
    import ChatPencilIcon from '$lib/icons/ChatPencilIcon.svelte';

	let helpPanelTimeout: number | undefined;
	
	let sidebarRef: HTMLDivElement;
	
	let isThreadSideBarOpen = $state(false);
	let isProfileMenuOpen = $state(false);
	let isHelpPanelOpen = $state(false);
	let isHistoryExpanded = $state(false);
	let isTasksExpanded = $state(false);
	
	let profileMenuRef = $state<HTMLDivElement>();
	let helpContainer: HTMLDivElement | undefined;
	let profileMenuContainer: HTMLDivElement | undefined;

	let { onToggle = () => {} }: { onToggle?: (isOpen: boolean, width?: number) => void } = $props();

	onMount(() => {
		// Initial state - sidebar is closed
		gsap.set(sidebarRef, { x: '-100%' });
		
		// Handle click outside behavior
		const handleClickOutside = (event: Event) => {
			const target = event.target as Node;
			
			// Close help panel when clicking outside of it
			const isOutsideHelp = !helpContainer?.contains(target);
			if (isOutsideHelp) {
				isHelpPanelOpen = false;
			}
			
			// Close profile menu when clicking outside of it
			const isOutsideProfileMenu = !profileMenuContainer?.contains(target);
			if (isProfileMenuOpen && isOutsideProfileMenu) {
				isProfileMenuOpen = false;
			}
		};
		
		document.addEventListener('click', handleClickOutside, true); // Use capture phase
		
		return () => {
			document.removeEventListener('click', handleClickOutside, true);
		};
	});

	// Handle toggle Thread Side sidebar
	function toggleSidebar() {
		isThreadSideBarOpen = !isThreadSideBarOpen;
		if (isThreadSideBarOpen) {
			gsap.to(sidebarRef, { 
				duration: 0.3, 
				x: '0%',
				ease: 'power2.out'
			});
		} else {
			gsap.to(sidebarRef, { 
				duration: 0.3, 
				x: '-100%',
				ease: 'power2.in'
			});
		}
		// Get the actual computed width of the sidebar
		const sidebarWidth = sidebarRef.offsetWidth;
		onToggle(isThreadSideBarOpen, sidebarWidth);
	}

	// Handle toggle profile menu
	function toggleProfileMenu() {
		isProfileMenuOpen = !isProfileMenuOpen;
		if (isProfileMenuOpen) {
			closeProfileMenu();
		} else {
			openProfileMenu();
		}
	}

	// Handle open profile menu
	function openProfileMenu() {		
		if (profileMenuRef && !isProfileMenuOpen) {
			gsap.fromTo(profileMenuRef, 
				{ 
					opacity: 0, 
					scale: 0.95,
					y: 10 
				}, 
				{ 
					opacity: 1, 
					scale: 1, 
					y: 0,
					duration: 0.2, 
					ease: 'power2.out' 
				}
			);
		}
	}

	// Handle close profile menu
	function closeProfileMenu() {
		if (profileMenuRef && isProfileMenuOpen) {
			gsap.to(profileMenuRef, {
				opacity: 0,
				scale: 0.95,
				y: 10,
				duration: 0.15,
				ease: 'power2.in',
				onComplete: () => {
					isProfileMenuOpen = false;
				}
			});
		}
	}

	// Handle header click
	function handleHeaderClick() {
		goto('/');
	}

	// Handle admin click
	function handleAdminClick() {
		goto('/admin');
	}

	// Handle setting click
	function handleSettingsClick() {
		window.location.hash = 'settings';
		closeProfileMenu();
	}

	// Handle logout click
	function handleLogoutClick() {
		closeProfileMenu();
	}

	// Handle help panel hover functions
	function handleHelpPanelEnter() {
		if (helpPanelTimeout) {
			clearTimeout(helpPanelTimeout);
			helpPanelTimeout = undefined;
		}
		isHelpPanelOpen = true;
	}

	// Handle help panel hover functions
	function handleHelpPanelLeave() {
		helpPanelTimeout = setTimeout(() => {
			isHelpPanelOpen = false;
		}, 150); // 150ms delay before hiding
	}

	// Handle history expansion toggle
	function toggleHistoryExpansion() {
		isHistoryExpanded = !isHistoryExpanded;
		// Load threads when history is expanded for the first time
		if (isHistoryExpanded && $threads.length === 0) {
			threadStore.loadThreads();
		}
	}

	// Handle thread click
	function handleThreadClick(threadId: string) {
		goto(`/chat/${threadId}`);
	}

	// Handle tasks expansion toggle
	function toggleTasksExpansion() {
		isTasksExpanded = !isTasksExpanded;
	}
</script>

<!-- Toggle Button -->
<button
	onclick={toggleSidebar}
	class="
		fixed top-2 left-2 z-50
		bg-transparent hover:bg-[var(--bg-third)]/50 text-[var(--text-primary)]
		p-1 rounded-xl hover:scale-105
		transition-all duration-200 
		select-none
		cursor-pointer
	"
	style="width: 2rem; height: 2rem;"
>
	<Tooltip text={isThreadSideBarOpen ? 'Close sidebar' : 'Open sidebar'} position="right" delay={1000}>
		{#snippet children()}
			<div class="transition-transform duration-200 {isThreadSideBarOpen ? 'rotate-90' : 'rotate-0'}">
				<MenuIcon/>
			</div>
		{/snippet}
	</Tooltip>
</button>
	

<!-- Sidebar -->
<div
	bind:this={sidebarRef}
	class="
		fixed top-0 left-0 h-full w-64 z-40
		p-1.25
		{$theme === 'dark' ? 'bg-[var(--bg-primary)] border-r border-[var(--border-color)]/50' : 'bg-[var(--bg-secondary)] border-r border-[var(--border-color)]'}"
>
	<div class="flex flex-col h-full justify-between">
		<div>
			<!-- Header -->
			{@render headerButton()}
			{@render createNewChatButton()}
			{@render findChatButton()}
			{@render voiceChatButton()}
			{@render documentProcessingButton()}
			{@render tasksButton()}
			{@render historyButton()}
		</div>
		<div>
			<div class="flex relative" bind:this={profileMenuContainer}>
				{#if isProfileMenuOpen}
					<!-- Profile Menu Dropdown -->
					<div
						bind:this={profileMenuRef}
						class="
							absolute bottom-full mb-2 left-0 right-0
							bg-[var(--bg-primary)]
							border border-[var(--border-color)]
							p-1.5
							gap-1.5
							shadow-sm
							rounded-3xl
						"
					>	
						{@render AdminButton()}				
						{@render SettingButton()}
						<div class="my-1 mx-3 border-t border-[var(--border-color)]"></div>
						{@render HelpButton()}
						{@render LogoutButton()}
					</div>
				{/if}
				{@render profileMenuButton()}
			</div>
		</div>
	</div>
</div>

{#snippet headerButton()}
	<button
		class="
			flex items-center
			text-right px-2.5 ml-9 mb-4
			bg-transparent
			rounded-2xl
			hover:bg-[var(--bg-secondary)]/30
			cursor-pointer
		"
		aria-label="Open sidebar"
		onclick={handleHeaderClick}
	>
		<div class="h-12 w-full"></div>
	</button>
{/snippet}

{#snippet createNewChatButton()}
	<button
		class="
			flex
			w-full text-left p-2
			rounded-xl
			text-sm
			bg-transparent hover:bg-[var(--bg-third)]/50
			cursor-pointer
			select-none
			gap-2
			group
		"
		aria-label="New chat"
		onclick={()=> goto("/")}
	>
		<div class="flex items-center justify-center px-0.5">
			<ChatPencilIcon/>
		</div>
		New chat
	</button>
{/snippet}

{#snippet findChatButton()}
	<button
		class="
			flex
			w-full text-left p-2
			rounded-xl
			text-sm
			bg-transparent hover:bg-[var(--bg-third)]/50
			cursor-pointer
			select-none
			gap-2
			group
		"
		aria-label="Find chat"
		onclick={()=> console.log("Open find chat panel")}
	>
		<div class="flex items-center justify-center px-0.5">
			<SearchIcon/>
		</div>
		Search chats
	</button>
{/snippet}

{#snippet voiceChatButton()}
	<button
		class="
			flex justify-between items-center
			w-full text-left p-2
			rounded-xl
			text-sm
			bg-transparent hover:bg-[var(--bg-third)]/50
			cursor-pointer
			select-none
			gap-2
			group
		"
		aria-label="Open Voice chat"
	>
		<div class="flex gap-2 justify-start items-center">
			<div class="flex items-center justify-center px-0.5">
				<VoiceIcon/>
			</div>
			Voice
		</div>
		<div 
			class="flex justify-center items-center opacity-0 group-hover:opacity-100 transition-opacity duration-100"
		>
			<MoreIcon size="18" className="text-[var(--text-secondary)]/60 mr-2"/>
		</div>
	</button>
{/snippet}

{#snippet documentProcessingButton()}
	<button
		class="
			flex justify-between items-center
			w-full text-left p-2
			rounded-xl
			text-sm
			bg-transparent hover:bg-[var(--bg-third)]/50
			cursor-pointer
			select-none
			gap-2
			group
		"
		aria-label="Open Intelligent Document Processing"
	>
		<div class="flex gap-2 justify-start items-center">
			<div class="flex items-center justify-center px-0.5">
				<DocumentsIcon2/>
			</div>
			Documents
		</div>
		<div 
			class="flex justify-center items-center opacity-0 group-hover:opacity-100 transition-opacity duration-200"
		>
			<MoreIcon size="18" className="text-[var(--text-secondary)]/60 mr-2"/>
		</div>
	</button>
{/snippet}

{#snippet tasksButton()}
	<div>
		<button
			onclick={() => goto(`/tasks`)}
			class="
				flex items-center
				w-full text-left p-2
				rounded-xl
				text-sm
				bg-transparent hover:bg-[var(--bg-third)]/50
				cursor-pointer
				select-none
				gap-2
				group
			"
			aria-label="Open Tasks"
		>
			<div class="flex items-center justify-center transform translate-x-[0.7px] group-hover:opacity-0 px-0.5">
				<TaskIcon/>
			</div>
			<div 
				onclick={(e) => { e.stopPropagation(); toggleTasksExpansion(); }}
				onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); toggleTasksExpansion(); } }}
				role="button"
				tabindex="0"
				aria-label="Toggle tasks dropdown"
				class="flex items-center justify-center transform absolute opacity-0 group-hover:opacity-100 px-0.5 transition-transform duration-200 {isTasksExpanded ? 'rotate-0 -translate-y-[1px]' : '-rotate-90'}"
			>
				<ArrowDownIcon/>
			</div>
			<div class="">
				Tasks
			</div>
		</button>
		
		{#if isTasksExpanded}
			<div class="mt-2 ml-4.5 border-l border-[var(--text-secondary)] pl-1 space-y-1 max-h-64 overflow-y-auto">
				<!-- Future tasks will go here -->
				<div class="text-sm text-[var(--text-secondary)] italic">
					No tasks yet
				</div>
			</div>
		{/if}
	</div>
{/snippet}

{#snippet historyButton()}
	<div>
		<button
			onclick={toggleHistoryExpansion}
			class="
				flex items-center
				w-full text-left p-2
				rounded-xl
				text-sm
				bg-transparent hover:bg-[var(--bg-third)]/50
				cursor-pointer
				select-none
				gap-2
				group
			"
			aria-label="Toggle Chat History"
		>
			<div class="flex items-center justify-center transform translate-x-[0.7px] group-hover:opacity-0 px-0.5">
				<ClockIcon/>
			</div>
			<div class="flex items-center justify-center transform absolute opacity-0 group-hover:opacity-100 px-0.5 transition-transform duration-200 {isHistoryExpanded ? 'rotate-0 -translate-y-[1px]' : '-rotate-90'}">
				<ArrowDownIcon/>
			</div>
			<div class="">
				History
			</div>
		</button>
		
		{#if isHistoryExpanded}
			<div class="mt-2 ml-4.5 border-l border-[var(--text-secondary)] pl-1 space-y-1 max-h-64 overflow-y-auto">
				{#if $threadsLoading}
					<div class="flex items-center gap-2 py-2 px-3 text-sm text-[var(--text-secondary)]">
						<div class="w-3 h-3 border border-[var(--text-secondary)] border-t-transparent rounded-full animate-spin"></div>
						Loading threads...
					</div>
				{:else if $threadsError}
					<div class="text-sm text-red-500 py-2 px-3">
						Error: {$threadsError}
					</div>
				{:else if $threads.length === 0}
					<div class="text-sm text-[var(--text-secondary)] italic py-2 px-3">
						No threads yet
					</div>
				{:else}
					{#each $threads as thread (thread.thread_id)}
						<button
							onclick={() => handleThreadClick(thread.thread_id)}
							class="
								flex items-center
								w-full text-left py-2 px-3
								rounded-lg
								text-sm
								bg-transparent hover:bg-[var(--bg-third)]/30
								cursor-pointer
								select-none
								transition-colors duration-150
								group
							"
							title={thread.thread_title}
						>
							<div class="truncate group-hover:text-[var(--text-primary)] text-[var(--text-secondary)]">
								{thread.thread_title}
							</div>
						</button>
					{/each}
				{/if}
			</div>
		{/if}
	</div>
{/snippet}

{#snippet profileMenuButton()}
	<button
		class="
			flex items-center gap-3
			w-full text-left px-3 py-2
			rounded-xl
			text-sm
			bg-transparent hover:bg-[var(--bg-third)]/50
			cursor-pointer
		"
		onclick={toggleProfileMenu}
	>
		<!-- User Avatar -->
		<div class="
			flex items-center justify-center
			w-8 h-8
			bg-gradient-to-br from-blue-500 to-purple-600
			rounded-full
			text-white text-sm font-semibold
			flex-shrink-0
		">
			JD
		</div>
		<!-- Username -->
		<span class="text-[var(--text-primary)] truncate">
			john_doe
		</span>
	</button>
{/snippet}

{#snippet AdminButton()}
	<button
		onclick={handleAdminClick}
		class="
			flex justify-between items-center
			w-full text-left px-3 py-2
			text-sm text-[var(--text-primary)]
			hover:bg-[var(--bg-third)]/50
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			group
		"
	>
		<div class="flex gap-2 justify-start items-center">
			<div class="flex justify-center items-center">
				<AdminIcon/>
			</div>
			Admin Panel
		</div>
		<div class="flex justify-center items-center opacity-0 group-hover:opacity-100 transition-opacity duration-100">
			<MoreIcon size="18" className="text-[var(--text-secondary)]/60 mb-[0.18rem] mr-2"/>
		</div>
	</button>
{/snippet}

{#snippet SettingButton()}
	<button
		onclick={handleSettingsClick}
		class="
			flex
			w-full text-left px-3 py-2
			text-sm text-[var(--text-primary)]
			hover:bg-[var(--bg-third)]/50
			transition-colors duration-150
			rounded-2xl
			gap-2
			cursor-pointer
			select-none
		"
	>
		<div class="flex justify-center items-center">
			<ManageToolIcon/>
		</div>
		Settings
	</button>
{/snippet}

{#snippet HelpButton()}
	<div class="relative" bind:this={helpContainer}>
		<button
			onmouseenter={handleHelpPanelEnter}
			onmouseleave={handleHelpPanelLeave}
			class="
				flex justify-between
				w-full text-left px-3 py-2
				text-sm text-[var(--text-primary)]
				hover:bg-[var(--bg-third)]/50
				transition-colors duration-150
				rounded-2xl
				cursor-pointer
				select-none
			"
		>
			<div class="flex gap-2">
				<div class="flex justify-center items-center">
					<HelpIcon/>
				</div>
				<span>Help</span>
			</div>
			<div class="flex flex-col justify-center">
				<RightArrowIcon size=18/>
			</div>
		</button>
		
		{#if isHelpPanelOpen}
			<!-- Invisible bridge to connect button to panel -->
			<div 
				role="presentation"
				class="absolute top-0 left-full w-2 h-full z-40"
				onmouseenter={handleHelpPanelEnter}
				onmouseleave={handleHelpPanelLeave}
			></div>
			
			<div 
				role="menu"
				tabindex="-1"
				class="
					absolute top-0 left-full ml-1 -mt-5
					bg-[var(--bg-primary)]
					border border-[var(--border-color)]
					rounded-3xl shadow-sm z-50 min-w-68 p-2
					transition-all duration-200
				"
				onmouseenter={handleHelpPanelEnter}
				onmouseleave={handleHelpPanelLeave}
			>
				{@render helpDocumentationOption()}
				{@render helpSupportOption()}
				{@render helpKeyboardShortcutsOption()}
			</div>
		{/if}
	</div>
{/snippet}

{#snippet LogoutButton()}
	<button
		onclick={handleLogoutClick}
		class="
			flex
			w-full text-left px-3 py-2
			text-sm
			{$theme === 'dark' ? 'text-red-400' : 'text-red-600'}
			{$theme === 'dark' ? 'hover:bg-red-900/20' : 'hover:bg-red-200/50'}
			transition-colors duration-150
			rounded-2xl
			gap-2
			cursor-pointer
		"
	>
		<div class="flex justify-center items-center">
			<LogoutIcon/>
		</div>
		Logout
	</button>
{/snippet}

{#snippet helpDocumentationOption()}
	<button
		type="button"
		role="menuitem"
		class="
			flex justify-between items-center
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-3
			group
		"
	>
		<div class="flex gap-2 justify-start items-center">
			<div class="flex flex-col justify-center">
				<BookIcon/>
			</div>
			<div class="text-[0.9rem]">Documentation</div>
		</div>
		<div class="flex justify-center items-center opacity-0 group-hover:opacity-100 transition-opacity duration-100">
			<MoreIcon size="18" className="text-[var(--text-secondary)]/60 mb-[0.18rem] mr-2"/>
		</div>
	</button>
{/snippet}

{#snippet helpSupportOption()}
	<button
		type="button"
		role="menuitem"
		class="
			flex justify-between items-center
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			gap-3
			group
		"
	>
		<div class="flex gap-2 justify-start items-center">
			<div class="flex flex-col justify-center">
				<PhoneIcon/>
			</div>
			<div class="text-[0.9rem]">Contact Support</div>
		</div>
		<div class="flex justify-center items-center opacity-0 group-hover:opacity-100 transition-opacity duration-100">
			<MoreIcon size="18" className="text-[var(--text-secondary)]/60 mb-[0.18rem] mr-2"/>
		</div>
	</button>
{/snippet}

{#snippet helpKeyboardShortcutsOption()}
	<button
		type="button"
		role="menuitem"
		class="
			flex justify-between items-center
			w-full text-left px-2 py-1.75
			hover:bg-[var(--bg-third)]/50
			text-[var(--text-primary)]
			transition-colors duration-150
			rounded-2xl
			cursor-pointer
			select-none
			group
		"
	>
		<div class="flex gap-2 justify-start items-center">
			<div class="flex flex-col justify-center">
				<KeyboardIcon/>
			</div>
			<div class="text-[0.9rem]">Keyboard Shortcuts</div>
		</div>
		<div class="flex justify-center items-center opacity-0 group-hover:opacity-100 transition-opacity duration-100">
			<MoreIcon size="18" className="text-[var(--text-secondary)]/60 mb-[0.18rem] mr-2"/>
		</div>
	</button>
{/snippet}