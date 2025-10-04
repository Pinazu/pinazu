<script lang="ts">
  export let size: string = "18";
  export let className: string = "";
</script>

<svg 
  xmlns="http://www.w3.org/2000/svg" 
  width={size} height={size} 
  viewBox="0 0 24 24" 
  class={className}
  fill="none" 
  stroke="currentColor" 
  stroke-width="2" 
  stroke-linecap="round" 
  stroke-linejoin="round"
>
  <rect x="2" y="4" width="20" height="16" rx="2"/>
  <path d="M6 8h.01"/>
  <path d="M10 8h.01"/>
  <path d="M14 8h.01"/>
  <path d="M18 8h.01"/>
  <path d="M8 12h.01"/>
  <path d="M12 12h.01"/>
  <path d="M16 12h.01"/>
  <path d="M7 16h10"/>
</svg>