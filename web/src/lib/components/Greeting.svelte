<script lang="ts">
    import "../../app.css"

    interface GrettingProps {
        input: string;
    }

    let { input }: GrettingProps = $props();
</script>

<div 
    class="
        flex flex-col items-center justify-end text-center
        py-8 px-4 md:py-4 md:px-6 sm:py-6 sm:px-3 mt-64 max-w-full
    "
>
    <div 
        class="
            text-3xl md:text-2xl sm:text-lg
            tracking-normal
            leading-relaxed md:leading-normal sm:leading-6
            select-none
            transition-all duration-200
            text-[var(--text-primary)]"
        style="
            text-rendering: optimizeLegibility;
            -webkit-font-smoothing: antialiased;
        "
    >
        {input}
    </div>
</div>