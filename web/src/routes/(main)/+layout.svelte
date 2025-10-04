<script lang="ts">
	import '../../app.css';
	import { page } from '$app/state';

    import SettingsPanel from '$lib/components/settings/SettingsPanel.svelte';
    import ThreadsSideBar from '$lib/components/sidebar/ThreadsSideBar.svelte';
	import { gsap } from 'gsap';
	
	// Import WebSocket store to initialize auto-connection
	import { websocketManager } from '$lib/stores/websocket';

	let hash = $derived(page.url.hash);
	let mainContentRef: HTMLDivElement;

	let { children } = $props();

	// Adjust main content margin when sidebar is toggled
	function handleSidebarToggle(isOpen: boolean, width?: number) {
		if (isOpen && width) {
			gsap.to(mainContentRef, {
				duration: 0.0,
				marginLeft: `${width}px`,
				ease: 'power2.out'
			});
		} else {
			gsap.to(mainContentRef, {
				duration: 0.0,
				marginLeft: '0px',
				ease: 'power2.in'
			});
		}
	}
</script>

<div class="relative">
	<ThreadsSideBar onToggle={handleSidebarToggle} />
	<div bind:this={mainContentRef} class="min-h-screen transition-all duration-300 relative">
		{@render children()}
	</div>
	<SettingsPanel {hash}/>
</div>
