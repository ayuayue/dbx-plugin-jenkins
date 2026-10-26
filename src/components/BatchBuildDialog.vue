<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { t } from "../i18n";
import { batchProgress, batchReasonKey, summarizeBatch } from "../lib/batchBuild";
import { statusMeta } from "../lib/status";
import type { BatchBuildResult, BatchTargetInfo, ParameterDefinition } from "../types";
import ParamFields from "./ParamFields.vue";

const props = withDefaults(
  defineProps<{
    /** 预检结果（降级时为缓存快照）。 */
    targets: BatchTargetInfo[];
    phase: "inspecting" | "confirm" | "running" | "done";
    skipBusy: boolean;
    results: BatchBuildResult[];
    /** 执行中的进度：已返回条数 / 总数。 */
    progress?: { done: number; total: number } | null;
    /** 整个批量请求失败时的错误文案（单条失败走 results）。 */
    error?: string;
    /** 共享参数（并集，仅 Boolean 开关）。 */
    parameters?: ParameterDefinition[];
    paramValues?: Record<string, unknown>;
  }>(),
  {
    progress: null,
    error: "",
    parameters: () => [],
    paramValues: () => ({}),
  },
);

const emit = defineEmits<{
  confirm: [skipBusy: boolean, values: Record<string, unknown>];
  close: [];
  retry: [];
  remove: [fullName: string];
  "update:paramValues": [value: Record<string, unknown>];
}>();

const skipBusy = ref(props.skipBusy);
const summary = computed(() => summarizeBatch(props.results));
/** 结果页只列非成功项：跳过 + 失败，成功项用汇总行带过。 */
const issues = computed(() => props.results.filter((item) => !item.success));
/** 可重试的只有真失败（跳过的不算，重试也会再被跳过）。 */
const retryCount = computed(
  () => props.results.filter((item) => !item.success && !item.skipped).length,
);
const done = computed(() => props.progress?.done ?? 0);
const total = computed(() => props.progress?.total ?? props.targets.length);
const percent = computed(() => batchProgress(done.value, total.value));
const busy = computed(() => props.phase === "inspecting" || props.phase === "running");

function close(): void {
  if (busy.value) return;
  emit("close");
}

function confirm(): void {
  emit("confirm", skipBusy.value, { ...props.paramValues });
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") close();
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <div class="overlay" @click.self="close">
    <section class="modal" role="dialog" aria-modal="true">
      <header>
        <h2>{{ t("batchTitle") }}</h2>
        <button type="button" class="close" :disabled="busy" @click="close">×</button>
      </header>

      <div class="body">
        <template v-if="phase === 'inspecting'">
          <p class="title">{{ t("batchInspecting") }}</p>
          <div class="running-bar indeterminate"><span /></div>
        </template>

        <template v-else-if="phase === 'confirm'">
          <p class="title">{{ t("batchConfirmTitle", { count: targets.length }) }}</p>
          <p v-if="error" class="hint warn-text">{{ t("batchInspectFailed") }}</p>
          <ul class="targets">
            <li v-for="target in targets" :key="target.fullName">
              <span class="name">{{ target.name }}</span>
              <span class="full" :title="target.fullName">{{ target.fullName }}</span>
              <span class="pill" :data-tone="statusMeta(target.status).tone">
                {{ t(statusMeta(target.status).key) }}
              </span>
              <button
                type="button"
                class="remove"
                :title="t('batchRemoveTarget', { name: target.name })"
                :aria-label="t('batchRemoveTarget', { name: target.name })"
                @click="emit('remove', target.fullName)"
              >
                ×
              </button>
            </li>
          </ul>

          <div v-if="parameters.length" class="params">
            <div class="params-head">
              <span>{{ t("batchParams") }}</span>
              <span class="count">{{ parameters.length }}</span>
            </div>
            <ParamFields
              :parameters="parameters"
              :model-value="paramValues"
              @update:model-value="emit('update:paramValues', $event)"
            />
            <p class="hint">{{ t("batchParamsHint") }}</p>
          </div>

          <label v-if="targets.length" class="option">
            <input v-model="skipBusy" type="checkbox" />
            <span>{{ t("batchSkipBusy") }}</span>
          </label>
        </template>

        <template v-else-if="phase === 'running'">
          <p class="title">{{ t("batchProgress", { done, total }) }}</p>
          <div class="running-bar"><span :style="{ width: `${percent}%` }" /></div>
          <p class="hint">{{ t("batchConfirmHint") }}</p>
        </template>

        <template v-else>
          <p class="title" :data-tone="summary.failed ? 'danger' : 'success'">
            {{
              t("batchResultSummary", {
                queued: summary.queued,
                skipped: summary.skipped,
                failed: summary.failed,
              })
            }}
          </p>
          <p v-if="error" class="hint">{{ error }}</p>
          <ul v-if="issues.length" class="issues">
            <li v-for="item in issues" :key="item.job">
              <span class="name">{{ item.job }}</span>
              <span class="reason" :data-skipped="item.skipped">
                {{ t(batchReasonKey(item.reason)) }}
              </span>
              <span v-if="item.error" class="err" :title="item.error">{{ item.error }}</span>
            </li>
          </ul>
        </template>
      </div>

      <footer>
        <template v-if="phase === 'inspecting'">
          <button type="button" class="primary" disabled>{{ t("batchInspecting") }}</button>
        </template>
        <template v-else-if="phase === 'confirm'">
          <button type="button" class="ghost" autofocus @click="close">
            {{ t("cancel") }}
          </button>
          <button
            type="button"
            class="primary"
            :disabled="!targets.length"
            @click="confirm"
          >
            {{ t("batchBuildAction", { count: targets.length }) }}
          </button>
        </template>
        <template v-else-if="phase === 'running'">
          <button type="button" class="primary" disabled>{{ t("batchRunning") }}</button>
        </template>
        <template v-else>
          <button v-if="retryCount" type="button" class="ghost retry" @click="emit('retry')">
            {{ t("batchRetryFailed", { count: retryCount }) }}
          </button>
          <button type="button" class="primary" autofocus @click="close">
            {{ t("batchClose") }}
          </button>
        </template>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: grid;
  place-items: center;
  padding: 24px;
  background: color-mix(in srgb, #000 48%, transparent);
}

.modal {
  width: min(560px, 100%);
  max-height: min(86vh, 820px);
  display: flex;
  flex-direction: column;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--border);
}

h2 {
  margin: 0;
  font-size: 15px;
}

.close {
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: 22px;
  line-height: 1;
  padding: 0 4px;
}

.body {
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  overflow: auto;
}

.title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.title[data-tone="success"] {
  color: var(--success);
}

.title[data-tone="danger"] {
  color: var(--danger);
}

.targets,
.issues {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 260px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel-soft);
}

.targets li,
.issues li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}

.targets li:last-child,
.issues li:last-child {
  border-bottom: 0;
}

.name {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.targets .name {
  flex: none;
  max-width: 180px;
}

.full {
  flex: 1;
  min-width: 0;
  color: var(--muted);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pill {
  flex: none;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
  background: var(--panel);
  color: var(--muted);
}

.pill[data-tone="success"] {
  background: color-mix(in srgb, var(--success) 14%, transparent);
  color: var(--success);
}

.pill[data-tone="danger"] {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

.pill[data-tone="warning"] {
  background: #d99b1a1f;
  color: #b47d0d;
}

.pill[data-tone="info"] {
  background: var(--accent-soft);
  color: var(--accent);
}

.params {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel-soft);
}

.params-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  font-size: 12px;
  font-weight: 700;
}

.params-head .count {
  color: var(--muted);
  font-weight: 600;
}

.option {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text);
}

.option input {
  width: 14px;
  height: 14px;
  accent-color: var(--accent);
}

.hint {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}

.warn-text {
  color: #b47d0d;
}

.issues .name {
  flex: none;
  max-width: 220px;
}

.issues .reason {
  flex: none;
  color: var(--danger);
  font-size: 11px;
}

.issues .reason[data-skipped="true"] {
  color: var(--muted);
}

.err {
  flex: 1;
  min-width: 0;
  color: var(--muted);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.running-bar {
  height: 4px;
  border-radius: 999px;
  background: var(--panel-soft);
  overflow: hidden;
}

.running-bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--accent);
  transition: width 0.25s ease;
}

.running-bar.indeterminate span {
  width: 35%;
  animation: batch-slide 1.2s ease-in-out infinite;
}

@keyframes batch-slide {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(300%);
  }
}

footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 18px;
  border-top: 1px solid var(--border);
}

.ghost,
.primary {
  border-radius: var(--radius-sm);
  padding: 8px 14px;
  font-weight: 600;
}

.ghost {
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text);
}

.primary {
  border: 0;
  background: var(--accent);
  color: #fff;
}

.retry {
  border-color: var(--accent);
  color: var(--accent);
}

.ghost:disabled,
.primary:disabled,
.close:disabled {
  opacity: 0.45;
}
</style>
