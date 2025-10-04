<script lang="ts">
  export let size: string = "18";
  export let className: string = "";
</script>

<svg 
  xmlns="http://www.w3.org/2000/svg" 
  width={size} 
  height={size} 
  viewBox="0 0 24 24" 
  fill="none" 
  stroke="currentColor" 
  stroke-width="1.8" 
  stroke-linecap="round" 
  stroke-linejoin="round"
  class={className}
>
  <!-- Empty document with rounded corners and corner cut out -->
  <path d="M6 2a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6H6z"/>
  <path d="M14 2v6h6"/>
  
  <!-- Pencil in corner cut out -->
  <path d="m16.5 7.5-1.5 1.5"/>
  <path d="M16.5 7.5 18 6"/>
  <circle cx="16.5" cy="7.5" r="0.5" fill="currentColor"/>
</svg>