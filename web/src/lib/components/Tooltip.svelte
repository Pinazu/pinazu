<script lang="ts">
	import { onMount } from "svelte";
	import { gsap } from "gsap";

	interface TooltipProps {
		text: string;
		position?: 'top' | 'bottom' | 'left' | 'right';
		delay?: number;
		maxWidth?: string;
		disabled?: boolean;
		block?: boolean;
		children: any;
	}

	let {
		text,
		position = 'top',
		delay = 300,
		maxWidth = '12rem',
		disabled = false,
		block = false,
		children
	}: TooltipProps = $props();

	let showTooltip = $state(false);
	let tooltipElement = $state<HTMLElement>();
	let timeoutId: number | undefined;

	const positionClasses = {
		top: 'bottom-full left-1/2 transform -translate-x-1/2 mb-2',
		bottom: 'top-full left-1/2 transform -translate-x-1/2 mt-2',
		left: 'right-full top-1/2 transform -translate-y-1/2 mr-2',
		right: 'left-full top-1/2 transform -translate-y-1/2 ml-2'
	};

	function animateIn() {
		if (tooltipElement) {
			gsap.fromTo(tooltipElement, 
				{ opacity: 0, scale: 0.95 },
				{ opacity: 1, scale: 1, duration: 0.15, ease: "power2.out" }
			);
		}
	}

	function animateOut() {
		if (tooltipElement) {
			gsap.to(tooltipElement, {
				opacity: 0,
				scale: 0.95,
				duration: 0.1,
				ease: "power2.in",
				onComplete: () => {
					showTooltip = false;
				}
			});
		}
	}

	function handleMouseEnter() {
		if (disabled) return;
		clearTimeout(timeoutId);
		timeoutId = setTimeout(() => {
			showTooltip = true;
			requestAnimationFrame(animateIn);
		}, delay) as unknown as number;
	}

	function handleMouseLeave() {
		if (disabled) return;
		clearTimeout(timeoutId);
		if (showTooltip) {
			animateOut();
		}
	}

	function handleFocusIn() {
		if (disabled) return;
		showTooltip = true;
		requestAnimationFrame(animateIn);
	}

	function handleFocusOut() {
		if (disabled) return;
		if (showTooltip) {
			animateOut();
		}
	}

	onMount(() => {
		return () => {
			clearTimeout(timeoutId);
		};
	});
</script>

<div 
	class="relative {block ? 'block' : 'inline-block'}"
	onmouseenter={handleMouseEnter}
	onmouseleave={handleMouseLeave}
	onfocusin={handleFocusIn}
	onfocusout={handleFocusOut}
	role="presentation"
>
	{@render children()}
	
	{#if showTooltip && !disabled}
		<div
			bind:this={tooltipElement}
			class="
				absolute z-[9999] px-2 py-1 text-xs 
				bg-[var(--tooltip-bg)] text-[var(--tooltip-text)]
				rounded-lg shadow-lg border border-[var(--tooltip-border)]
				whitespace-nowrap pointer-events-none
				{positionClasses[position]}
			"
			style="max-width: {maxWidth}; opacity: 0;"
			role="tooltip"
			aria-hidden="false"
		>
			{text}
		</div>
	{/if}
</div>

<style>
	:global(:root) {
		--tooltip-bg: oklch(20% 0.005 285.823);
		--tooltip-text: oklch(96% 0.001 286.375);
		--tooltip-border: oklch(30% 0.006 286.033);
	}

	:global(.dark) {
		--tooltip-bg: oklch(85% 0.004 286.32);
		--tooltip-text: oklch(20% 0.005 285.823);
		--tooltip-border: oklch(75% 0.015 286.067);
	}
</style>