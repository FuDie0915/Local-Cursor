<script setup>
const props = defineProps({
  enabled: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
  compact: { type: Boolean, default: false },
  label: { type: String, default: "" },
  description: { type: String, default: "" },
  enabledText: { type: String, default: "已开启" },
  disabledText: { type: String, default: "已关闭" },
  busyText: { type: String, default: "切换中..." },
});

const emit = defineEmits(["change"]);

function handleToggle() {
  if (props.disabled || props.busy) {
    return;
  }
  emit("change", !props.enabled);
}
</script>

<template>
  <div
    class="flex items-center justify-between gap-4"
    :class="compact ? 'py-0' : 'py-1'"
  >
    <div class="flex min-w-0 flex-col" :class="compact ? 'gap-[2px]' : 'gap-1'">
      <div :class="compact ? 'text-[12px]' : 'text-sm'" class="font-medium text-white">
        {{ label }}
      </div>
      <div
        v-if="description"
        :class="compact ? 'text-[11px] leading-[16px]' : 'text-xs'"
        class="text-[#a8a8a8]"
      >
        {{ description }}
      </div>
      <div
        :class="[
          compact ? 'text-[11px] leading-[16px]' : 'text-xs',
          enabled ? 'text-white' : 'text-[#666]',
        ]"
      >
        {{ busy ? busyText : enabled ? enabledText : disabledText }}
      </div>
    </div>

    <button
      type="button"
      role="switch"
      :aria-checked="enabled"
      :disabled="disabled || busy"
      class="relative inline-flex h-[18px] w-[34px] shrink-0 cursor-pointer rounded-full outline-none transition-all duration-200 ease-out disabled:cursor-not-allowed disabled:opacity-50 focus-visible:ring-2 focus-visible:ring-white/30"
      :class="enabled ? 'bg-white' : 'bg-[#1c1c1c] border border-[#383838]'"
      @click="handleToggle"
    >
      <span
        class="absolute top-[2px] left-[2px] inline-flex h-[12px] w-[12px] rounded-full transition-all duration-200 ease-out"
        :class="enabled ? 'translate-x-[16px] bg-black' : 'translate-x-0 bg-[#666]'"
      />
    </button>
  </div>
</template>