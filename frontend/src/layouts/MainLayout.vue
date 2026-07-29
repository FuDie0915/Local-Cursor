<script setup>
import { Window } from "@wailsio/runtime";
import LocaleSelect from "@/components/LocaleSelect.vue";
import { appState, syncServiceState } from "@/state/appState";
import { isWindows } from "@/utils/isWindows";
import { computed, onMounted, onUnmounted } from "vue";
import { useRoute } from "vue-router";
import Logo from "@/assets/logo.png";

const route = useRoute();
const showIcon = computed(() => route.meta.showIcon !== false);
const title = computed(() => route.meta.title ?? "Local-Curosr");
const directlyClose = computed(() => route.meta.directlyClose === true);
const showFooter = computed(() => route.path === "/");
let proxyStateTimer = null;
const proxyStatePollIntervalMs = 10000;
const netProxyEndpoint = computed(
  () => appState.netProxyHttps || appState.netProxyHttp || "",
);
const proxyBadgeText = computed(() => {
  if (appState.netProxyUsingSystem) {
    return "已识别系统代理";
  }
  return "";
});
const proxyBadgeTitle = computed(() => {
  if (appState.netProxyUsingSystem) {
    return netProxyEndpoint.value
      ? `当前出站请求使用系统代理：${netProxyEndpoint.value}`
      : "当前出站请求使用系统代理";
  }
  if (appState.netProxyUsingEnv) {
    return netProxyEndpoint.value
      ? `当前出站请求使用环境变量代理：${netProxyEndpoint.value}`
      : "当前出站请求使用环境变量代理";
  }
  if (appState.netProxyPacIgnored) {
    return "检测到系统 PAC/自动代理，当前版本按直连处理";
  }
  return "当前出站请求未使用系统代理";
});

async function minimizeWindow() {
  await Window.Minimise();
}

async function closeWindow() {
  if (directlyClose.value) {
    await Window.Close();
    return;
  }
  await new Promise((resolve) => setTimeout(resolve, 200));
  await Window.Hide();
}

onMounted(() => {
  proxyStateTimer = window.setInterval(() => {
    if (showFooter.value) {
      void syncServiceState().catch(() => {});
    }
  }, proxyStatePollIntervalMs);
});

onUnmounted(() => {
  if (proxyStateTimer) {
    window.clearInterval(proxyStateTimer);
    proxyStateTimer = null;
  }
});
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden flex-col">
    <div
      class="fixed top-0 w-screen h-[40px] z-9999 w-full"
      style="--wails-draggable: drag"
    ></div>

    <header
      class="flex h-[40px] center-row px-[20px] w-full min-h-0 shrink-0 justify-between relative"
      style="--wails-draggable: drag"
      :class="{ '!justify-center': !isWindows }"
    >
      <div class="center-row gap-2" style="font-family: var(--font-num);">
        <img v-if="showIcon" :src="Logo" class="w-[18px] h-[18px]" />
        <div>{{ title }}</div>
      </div>
      <div
        v-if="isWindows"
        class="absolute right-[10px] top-[8px] z-99999 center-row gap-[1px]"
      >
        <button
          class="text-[20px] center-row justify-center w-[30px] h-[23px] rounded-[4px] text-[#777] hover:bg-[#333] hover:text-[#ddd] cursor-pointer"
          @click="minimizeWindow"
        >
          <span class="icon-[ic--round-minus]"></span>
        </button>
        <button
          class="text-[20px] center-row justify-center w-[30px] h-[23px] rounded-[4px] text-[#777] hover:bg-[#333] hover:text-[#ddd] cursor-pointer"
          @click="closeWindow"
        >
          <span class="icon-[ic--round-close]"></span>
        </button>
      </div>
    </header>

    <main class="flex-1 min-h-0 overflow-hidden flex flex-col w-full">
      <router-view />
    </main>

    <footer
      v-if="showFooter"
      class="flex !pr-1 h-[30px] shrink-0 items-center gap-[8px] border-t border-[#242424] px-[14px] text-[12px] text-[#8f8f8f]"
    >
      <div
        v-if="proxyBadgeText"
        class="center-row  border-none gap-[2px]  border-none  px-[0px] py-[3px] leading-none "
        aria-live="polite"
      >
        <span class="icon-[mdi--wifi] text-[15px]"></span>
        <span class="truncate">{{ proxyBadgeText }}</span>
      </div>
      <div class="ml-auto flex shrink-0 items-center gap-[8px]">
        <span v-if="appState.appVersion" class="text-[12px] text-[#666] tabular-nums">v{{ appState.appVersion }}</span>
        <LocaleSelect
          :border="false"
          aria-label="界面语言"
          wrapper-class="w-auto"
          button-class="h-[24px] bg-transparent px-1.5 text-[12px] !text-[#8f8f8f] !hover:text-[#e5e5e5]"
          menu-class="text-[12px]"
        />
      </div>
    </footer>
  </div>
</template>
