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
  stroke-width="1.9" 
  stroke-linecap="round" 
  stroke-linejoin="round"
>
  <path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/>
  <path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/>
</svg>