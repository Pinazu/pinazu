<script lang="ts">
    import ArrowDownIcon from "$lib/icons/ArrowDownIcon.svelte";
    import TickIcon from "$lib/icons/TickIcon.svelte";
    import { onMount } from "svelte";

    let selectedLanguage = $state('auto-detect');
    let showLanguageDropdown = $state(false);
    let languageContainer: HTMLDivElement | undefined = $state();

    const languageOptions = [
        { id: 'auto-detect', label: 'Auto-Detect', country: '' },
        { id: 'en-us', label: 'English', country: 'US' },
        { id: 'separator', label: '', country: '' },
        { id: 'af-za', label: 'Afrikaans', country: 'ZA' },
        { id: 'am-et', label: 'Amharic', country: 'ET' },
        { id: 'ar-sa', label: 'Arabic', country: 'SA' },
        { id: 'hy-am', label: 'Armenian', country: 'AM' },
        { id: 'az-az', label: 'Azerbaijani', country: 'AZ' },
        { id: 'eu-es', label: 'Basque', country: 'ES' },
        { id: 'bn-bd', label: 'Bengali', country: 'BD' },
        { id: 'bg-bg', label: 'Bulgarian', country: 'BG' },
        { id: 'ca-es', label: 'Catalan', country: 'ES' },
        { id: 'zh-cn', label: 'Chinese', country: 'CN' },
        { id: 'zh-tw', label: 'Chinese', country: 'TW' },
        { id: 'hr-hr', label: 'Croatian', country: 'HR' },
        { id: 'cs-cz', label: 'Czech', country: 'CZ' },
        { id: 'da-dk', label: 'Danish', country: 'DK' },
        { id: 'nl-nl', label: 'Dutch', country: 'NL' },
        { id: 'et-ee', label: 'Estonian', country: 'EE' },
        { id: 'tl-ph', label: 'Filipino', country: 'PH' },
        { id: 'fi-fi', label: 'Finnish', country: 'FI' },
        { id: 'fr-fr', label: 'French', country: 'FR' },
        { id: 'gl-es', label: 'Galician', country: 'ES' },
        { id: 'ka-ge', label: 'Georgian', country: 'GE' },
        { id: 'de-de', label: 'German', country: 'DE' },
        { id: 'gu-in', label: 'Gujarati', country: 'IN' },
        { id: 'he-il', label: 'Hebrew', country: 'IL' },
        { id: 'hi-in', label: 'Hindi', country: 'IN' },
        { id: 'hu-hu', label: 'Hungarian', country: 'HU' },
        { id: 'is-is', label: 'Icelandic', country: 'IS' },
        { id: 'id-id', label: 'Indonesian', country: 'ID' },
        { id: 'ga-ie', label: 'Irish', country: 'IE' },
        { id: 'it-it', label: 'Italian', country: 'IT' },
        { id: 'ja-jp', label: 'Japanese', country: 'JP' },
        { id: 'kn-in', label: 'Kannada', country: 'IN' },
        { id: 'kk-kz', label: 'Kazakh', country: 'KZ' },
        { id: 'km-kh', label: 'Khmer', country: 'KH' },
        { id: 'ko-kr', label: 'Korean', country: 'KR' },
        { id: 'ky-kg', label: 'Kyrgyz', country: 'KG' },
        { id: 'lo-la', label: 'Lao', country: 'LA' },
        { id: 'lv-lv', label: 'Latvian', country: 'LV' },
        { id: 'lt-lt', label: 'Lithuanian', country: 'LT' },
        { id: 'ms-my', label: 'Malay', country: 'MY' },
        { id: 'ml-in', label: 'Malayalam', country: 'IN' },
        { id: 'mt-mt', label: 'Maltese', country: 'MT' },
        { id: 'mr-in', label: 'Marathi', country: 'IN' },
        { id: 'mn-mn', label: 'Mongolian', country: 'MN' },
        { id: 'my-mm', label: 'Myanmar', country: 'MM' },
        { id: 'ne-np', label: 'Nepali', country: 'NP' },
        { id: 'no-no', label: 'Norwegian', country: 'NO' },
        { id: 'fa-ir', label: 'Persian', country: 'IR' },
        { id: 'pl-pl', label: 'Polish', country: 'PL' },
        { id: 'pt-br', label: 'Portuguese', country: 'BR' },
        { id: 'pt-pt', label: 'Portuguese', country: 'PT' },
        { id: 'pa-in', label: 'Punjabi', country: 'IN' },
        { id: 'ro-ro', label: 'Romanian', country: 'RO' },
        { id: 'ru-ru', label: 'Russian', country: 'RU' },
        { id: 'sr-rs', label: 'Serbian', country: 'RS' },
        { id: 'si-lk', label: 'Sinhala', country: 'LK' },
        { id: 'sk-sk', label: 'Slovak', country: 'SK' },
        { id: 'sl-si', label: 'Slovenian', country: 'SI' },
        { id: 'es-es', label: 'Spanish', country: 'ES' },
        { id: 'es-mx', label: 'Spanish', country: 'MX' },
        { id: 'sw-ke', label: 'Swahili', country: 'KE' },
        { id: 'sv-se', label: 'Swedish', country: 'SE' },
        { id: 'tg-tj', label: 'Tajik', country: 'TJ' },
        { id: 'ta-in', label: 'Tamil', country: 'IN' },
        { id: 'te-in', label: 'Telugu', country: 'IN' },
        { id: 'th-th', label: 'Thai', country: 'TH' },
        { id: 'tr-tr', label: 'Turkish', country: 'TR' },
        { id: 'uk-ua', label: 'Ukrainian', country: 'UA' },
        { id: 'ur-pk', label: 'Urdu', country: 'PK' },
        { id: 'uz-uz', label: 'Uzbek', country: 'UZ' },
        { id: 'vi-vn', label: 'Vietnamese', country: 'VN' },
        { id: 'cy-gb', label: 'Welsh', country: 'GB' },
        { id: 'zu-za', label: 'Zulu', country: 'ZA' }
    ] as const;

    onMount(() => {
        const handleClickOutside = (event: Event) => {
            const target = event.target as Node;
            const isOutside = !languageContainer?.contains(target);
            
            if (isOutside) showLanguageDropdown = false;
        };
        
        document.addEventListener('click', handleClickOutside);
        
        return () => {
            document.removeEventListener('click', handleClickOutside);
        };
    });

    function selectLanguage(languageId: string) {
        if (languageId === 'separator') return;
        selectedLanguage = languageId;
        showLanguageDropdown = false;
    }

    function getCurrentLanguageLabel() {
        const currentLanguage = languageOptions.find(option => option.id === selectedLanguage);
        if (!currentLanguage) return 'Auto-Detect';
        
        if (currentLanguage.country) {
            return `${currentLanguage.label} (${currentLanguage.country})`;
        }
        return currentLanguage.label;
    }
</script>

<div class="relative" bind:this={languageContainer}>
    <button 
        type="button"
        aria-label="Select Language"
        onclick={() => showLanguageDropdown = !showLanguageDropdown}
        class="
            flex items-center gap-1.5
            bg-transparent hover:bg-[var(--bg-third)]/50
            text-[0.9rem] text-[var(--text-secondary)]
            px-2.5 py-1.5 rounded-2xl
            hover:scale-105
            cursor-pointer border-none
            transition-all duration-200
        "
    >
        <span class="font-medium text-[0.8rem]">{getCurrentLanguageLabel()}</span>
        <div class="flex items-center justify-center -mt-0.5">
            <ArrowDownIcon size=14/>
        </div>
    </button>

    {#if showLanguageDropdown}
        <div 
            class="
                fixed
                mt-1 bg-[var(--bg-primary)]
                border border-[var(--border-color)]
                rounded-2xl shadow-lg z-[150] min-w-48 max-w-64 p-2
                max-h-120 overflow-y-auto
            "
            style="
                top: {languageContainer?.getBoundingClientRect().bottom + 4}px;
                left: {languageContainer?.getBoundingClientRect().left}px;
            "
        >
            {#each languageOptions as option}
                {#if option.id === 'separator'}
                    <div class="border-t border-[var(--border-color)]/50 my-1"></div>
                {:else}
                    <button
                        type="button"
                        onclick={() => selectLanguage(option.id)}
                        class="
                            flex justify-between items-center
                            w-full text-left px-3 py-2 
                            hover:bg-[var(--bg-third)]/50
                            text-[var(--text-primary)]
                            transition-colors duration-150
                            rounded-xl
                            cursor-pointer
                        "
                    >
                        <div class="flex flex-col">
                            <div class="text-[0.85rem] font-medium">
                                {option.label}
                                {#if option.country}
                                    <span class="text-[var(--text-secondary)] ml-1">({option.country})</span>
                                {/if}
                            </div>
                        </div>
                        {#if selectedLanguage === option.id}
                            <div class="flex justify-center items-center">
                                <TickIcon size=16/>
                            </div>
                        {/if}
                    </button>
                {/if}
            {/each}
        </div>
    {/if}
</div>