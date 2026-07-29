<script setup>
import Button from "@/components/ui/Button.vue";
import Card from "@/components/ui/Card.vue";
import Switch from "@/components/ui/Switch.vue";
import HomeMetricsCard from "@/components/HomeMetricsCard.vue";
import { useMessage } from "@/composables/useMessage";
import { showModal } from "@/composables/useModal";
import {
  appState,
  appViewState,
  openConfigWindow,
  openModelConfigWindow,
  saveRoutingMode,
  syncHomeMetrics,
  syncServiceState,
  toUserError,
  toggleService,
} from "@/state/appState";
import { computed, onMounted } from "vue";

const directModeEnabled = computed(() => appState.routingMode === "upstream");
const message = useMessage();

async function showActionError(title, error) {
  await showModal({
    title,
    content: String(error || "服务错误").trim() || "服务错误",
  });
}

async function handleToggleService() {
  const result = await toggleService();
  if (!result.ok) {
    await showActionError("服务操作失败", result.error);
  }
}

async function handleRefreshState() {
  const [serviceStateResult] = await Promise.allSettled([
    syncServiceState(),
    syncHomeMetrics(),
  ]);
  if (serviceStateResult.status === "rejected") {
    await showActionError("刷新失败", toUserError(serviceStateResult.reason));
  }
}

async function handleRefreshMetrics() {
  await syncHomeMetrics().catch(() => {});
}

async function handleOpenConfig() {
  try {
    await openConfigWindow();
  } catch (error) {
    await showActionError("打开失败", toUserError(error));
  }
}

async function handleOpenModelConfig() {
  try {
    await openModelConfigWindow();
  } catch (error) {
    await showActionError("打开失败", toUserError(error));
  }
}

async function handleDirectModeChange(enabled) {
  const result = await saveRoutingMode(enabled ? "upstream" : "local");
  if (!result.ok) {
    await showActionError("切换失败", result.error);
    return;
  }
  message.success(enabled ? "已切换到直连 Cursor 模式" : "已切换到本地服务模式");
}

onMounted(() => {
  void handleRefreshState();
});
</script>

<template>
  <div class="flex flex-col gap-[16px] p-[20px] text-white overflow-y-auto h-full">
    <!-- service status -->
    <Card>
      <div class="flex flex-col gap-[12px]">
        <div class="flex items-start justify-between gap-4">
          <div class="flex flex-col gap-1">
            <div class="text-sm" :class="appViewState.serviceStatusClass">
              {{ appViewState.serviceStatusText }}
            </div>
          </div>
          <Button variant="primary" :disabled="appState.serviceBusy" @click="handleToggleService">
            <span class="icon-[mdi--pause] text-[16px]" v-if="appState.serviceRunning"></span>
            <span class="icon-[mdi--play] text-[16px]" v-else></span>
            <span> {{ appViewState.serviceButtonText }}</span>
          </Button>
        </div>

        <div v-if="appState.serviceLastError"
          class="rounded-[4px] border border-[#ff4444]/30 bg-[#ff4444]/5 px-3 py-2 text-sm text-[#ff4444]"
        >
          {{ appState.serviceLastError }}
        </div>

        <Switch
          label="直连模式"
          description="开启后，Cursor将直接接通官方，请勿开启"
          enabled-text="当前为直连模式"
          disabled-text="当前为本地服务模式"
          :enabled="directModeEnabled"
          :busy="appState.configSaving"
          :disabled="appState.configSaving"
          @change="handleDirectModeChange"
        />
      </div>
    </Card>

    <!-- metrics -->
    <HomeMetricsCard
      :metrics="appState.homeMetrics"
      :loading="appState.homeMetricsLoading"
      :error="appState.homeMetricsError"
      @refresh="handleRefreshMetrics"
    />

    <!-- quick actions -->
    <Card>
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="text-[14px] font-medium text-white">本地配置</h2>
          <div class="text-[12px] text-[#a8a8a8]">打开设置目录，或单独管理模型配置</div>
        </div>
        <div class="flex items-center gap-2">
          <Button variant="default" @click="handleOpenConfig">设置文件夹</Button>
          <Button variant="primary" @click="handleOpenModelConfig">模型配置</Button>
        </div>
      </div>
    </Card>
  </div>
</template>