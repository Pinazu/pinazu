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
  stroke-width="2" 
  stroke-linecap="round" 
  stroke-linejoin="round"
  class={className}
>
  <circle cx="6.5" cy="6.5" r="3.5"/>
  <circle cx="17.5" cy="6.5" r="3.5"/>
  <circle cx="6.5" cy="17.5" r="3.5"/>
  <path d="M17.5 14v7"/>
  <path d="M14 17.5h7"/>
</svg>