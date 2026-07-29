<script setup>
defineProps({
  variant: {
    type: String,
    default: "default",
    validator: (v) => ["default", "primary", "text"].includes(v),
  },
});
</script>

<template>
  <button
    v-if="variant === 'text'"
    type="button"
    class="!whitespace-nowrap shrink-0 cursor-pointer text-sm text-[#a8a8a8] transition-colors duration-150 hover:text-white active:text-white"
  >
    <slot />
  </button>
  <button
    v-else
    type="button"
    class="!whitespace-nowrap cursor-pointer inline-flex items-center gap-[6px] min-h-[28px] rounded-[4px] px-[10px] py-[5px] text-[12px] font-medium transition-all duration-150 active:scale-[.98] border"
    :class="{
      'bg-[#141414] border-[#383838] text-white hover:bg-[#1c1c1c] hover:border-[#666]': variant === 'default',
      'bg-white border-white text-black hover:bg-[#a8a8a8] hover:border-[#a8a8a8]': variant === 'primary',
    }"
  >
    <slot />
  </button>
</template>