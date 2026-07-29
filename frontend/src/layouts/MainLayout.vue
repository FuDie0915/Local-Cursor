<script setup>
import { Window } from "@wailsio/runtime";
import LocaleSelect from "@/components/LocaleSelect.vue";
import { appState, syncServiceState } from "@/state/appState";
import { isWindows } from "@/utils/isWindows";
import { computed, onMounted, onUnmounted } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();
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
  <div class="flex h-screen w-screen overflow-hidden">
    <!-- drag region for frameless window -->
    <div
      class="fixed top-0 left-0 h-[40px] z-9999"
      style="--wails-draggable: drag; width: 52px;"
    ></div>

    <!-- ═══ SIDEBAR ═══ -->
    <nav class="flex flex-col items-center w-[52px] shrink-0 bg-[#0a0a0a] border-r border-[#2a2a2a] pt-[12px] gap-[4px] z-50">
      <div class="flex items-center justify-center w-[28px] h-[28px] border-[1.5px] border-white rounded-[6px] font-extrabold text-[13px] tracking-tight mb-[16px]">
        LC
      </div>
      <div class="flex-1"></div>
      <!-- nav buttons handled by router -->
      <!-- bottom: language + version -->
      <div class="flex flex-col items-center gap-[6px] pb-[10px]">
        <LocaleSelect
          :border="false"
          aria-label="界面语言"
          wrapper-class="w-auto"
          button-class="h-[24px] bg-transparent px-1.5 text-[12px] !text-[#666] !hover:text-[#fff]"
          menu-class="text-[12px]"
        />
        <span v-if="appState.appVersion" class="text-[10px] tabular-nums text-[#444] leading-none">
          v{{ appState.appVersion }}
        </span>
      </div>
    </nav>

    <!-- ═══ MAIN ═══ -->
    <div class="flex flex-1 min-w-0 flex-col">
      <!-- topbar -->
      <header
        class="flex h-[40px] shrink-0 items-center justify-between px-[16px] bg-black border-b border-[#2a2a2a] relative"
        style="--wails-draggable: drag"
      >
        <span class="font-semibold text-[13px] tracking-wide">{{ title }}</span>
        <div
          v-if="isWindows"
          class="absolute right-[10px] top-[8px] flex items-center gap-[1px] z-99999"
        >
          <button
            class="flex items-center justify-center w-[30px] h-[23px] rounded-[4px] text-[#666] hover:bg-[#1c1c1c] hover:text-white cursor-pointer"
            @click="minimizeWindow"
          >
            <span class="icon-[ic--round-minus] text-[20px]"></span>
          </button>
          <button
            class="flex items-center justify-center w-[30px] h-[23px] rounded-[4px] text-[#666] hover:bg-[#1c1c1c] hover:text-white cursor-pointer"
            @click="closeWindow"
          >
            <span class="icon-[ic--round-close] text-[20px]"></span>
          </button>
        </div>
      </header>

      <!-- content -->
      <main class="flex-1 min-h-0 overflow-hidden flex flex-col">
        <router-view />
      </main>

      <!-- statusbar (only on home) -->
      <footer
        v-if="showFooter"
        class="flex h-[28px] shrink-0 items-center justify-between px-[12px] border-t border-[#2a2a2a] bg-[#0a0a0a] text-[10px] text-[#666]"
      >
        <div class="flex items-center gap-[12px]">
          <span>{{ appState.serviceRunning ? "● 服务运行中" : "● 服务已停止" }}</span>
        </div>
        <div class="flex items-center gap-[12px]">
          <span
            v-if="proxyBadgeText"
            class="flex items-center gap-[2px]"
            :title="proxyBadgeTitle"
          >
            <span class="icon-[mdi--wifi] text-[15px]"></span>
            <span class="truncate">{{ proxyBadgeText }}</span>
          </span>
        </div>
      </footer>
    </div>
  </div>
</template>