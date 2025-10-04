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
  <path d="M2 10v4"/>
  <path d="M6 6v12"/>
  <path d="M10 3v18"/>
  <path d="M14 8v8"/>
  <path d="M18 5v14"/>
  <path d="M22 10v4"/>
</svg>