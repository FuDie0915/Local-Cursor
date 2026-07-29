<script setup>
import Button from "@/components/ui/Button.vue";
import Card from "@/components/ui/Card.vue";
import Input from "@/components/ui/Input.vue";
import LocaleSelect from "@/components/LocaleSelect.vue";
import Select from "@/components/ui/Select.vue";
import { showModal } from "@/composables/useModal";
import {
  appState,
  openModelConfigWindow,
  persistUserConfig,
  reloadUserConfig,
  ROUTE_MODE_OPTIONS,
  saveTabServerConfig,
  toUserError,
} from "@/state/appState";
import { onMounted } from "vue";

const routeModeOptions = ROUTE_MODE_OPTIONS;

async function showActionError(title, error) {
  await showModal({
    title,
    content: String(error || "服务错误").trim() || "服务错误",
  });
}

async function handleSaveConfig() {
  const result = await persistUserConfig();
  if (!result.ok) {
    await showActionError("保存失败", result.error);
    return;
  }
  await showModal({
    title: "提示",
    content: "本地配置已保存",
  });
}

async function handleSaveTabServerConfig() {
  const result = await saveTabServerConfig(
    appState.tabServerBaseURL,
    appState.tabServerToken,
  );
  if (!result.ok) {
    await showActionError("保存失败", result.error);
    return;
  }
  await showModal({
    title: "提示",
    content: "Tab 补全配置已保存，重启服务后生效",
  });
}

async function handleOpenModelConfig() {
  try {
    await openModelConfigWindow();
  } catch (error) {
    await showActionError("打开失败", toUserError(error));
  }
}

onMounted(async () => {
  await reloadUserConfig().catch(() => {});
});
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-4 overflow-y-auto p-4 pt-0 text-white">
    <Card>
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="text-base font-medium text-white">本地配置</h2>
          <div class="text-sm text-[#a8a8a8]">
            可配置运行模式和模型渠道；运行日志位于
            <code>~/.cursor-local-assistant-v2/logs/</code>
          </div>
        </div>
        <Button variant="primary" :disabled="appState.configSaving" @click="handleSaveConfig">
          {{ appState.configSaving ? "保存中..." : "保存配置" }}
        </Button>
      </div>
    </Card>

    <Card>
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="text-base font-medium text-white">运行模式</h2>
          <div class="text-sm text-[#a8a8a8]">
            控制白名单主链路请求走本地服务，还是回到原始 Cursor 上游地址
          </div>
        </div>
        <div class="w-[220px] max-w-full">
          <Select
            v-model="appState.routingMode"
            :options="routeModeOptions"
            placeholder="选择模式"
          />
        </div>
      </div>
    </Card>

    <Card>
      <div class="flex flex-col gap-4">
        <div class="flex items-center justify-between gap-4">
          <div>
            <h2 class="text-base font-medium text-white">Tab 补全服务</h2>
            <div class="text-sm text-[#a8a8a8]">
              填写自定义 Tab 补全后端地址和 Token，留空则禁用 Tab 补全
            </div>
          </div>
          <Button
            variant="primary"
            :disabled="appState.configSaving"
            @click="handleSaveTabServerConfig"
          >
            {{ appState.configSaving ? "保存中..." : "保存" }}
          </Button>
        </div>
        <div class="flex flex-col gap-2">
          <label class="text-sm text-[#a8a8a8]">服务地址</label>
          <Input
            v-model="appState.tabServerBaseURL"
            placeholder="例如：http://127.0.0.1:18041"
          />
        </div>
        <div class="flex flex-col gap-2">
          <label class="text-sm text-[#a8a8a8]">访问 Token</label>
          <Input
            v-model="appState.tabServerToken"
            type="password"
            allow-visibility-toggle
            placeholder="从 Cursor 的 state.vscdb 中获取 accessToken"
          />
        </div>
      </div>
    </Card>

    <Card>
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="text-base font-medium text-white">界面语言</h2>
          <div class="text-sm text-[#a8a8a8]">
            切换当前界面显示语言，设置会立即生效并保存在本机
          </div>
        </div>
        <LocaleSelect wrapper-class="w-[220px] max-w-full" />
      </div>
    </Card>

    <Card>
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="text-base font-medium text-white">模型配置</h2>
          <div class="text-sm text-[#a8a8a8]">
            已配置 {{ appState.modelAdapters.length }} 个模型适配器
          </div>
        </div>
        <Button variant="primary" @click="handleOpenModelConfig">打开模型配置</Button>
      </div>
    </Card>
  </div>
</template>
