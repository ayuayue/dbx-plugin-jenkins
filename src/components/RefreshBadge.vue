<script setup lang="ts">
import { computed } from "vue";
import { t } from "../i18n";
import { formatClockTime, refreshBadge } from "../lib/refreshStatus";

const props = defineProps<{
  mode: string | null;
  paused: boolean;
  inFlight: boolean;
  lastAt: number | null;
  intervalMs: number;
}>();

const tone = computed(() => refreshBadge(props.mode, props.paused, props.inFlight));

const label = computed(() =>
  tone.value === "paused" ? t("autoRefreshPausedShort") : t("autoRefreshing"),
);

/** 暂停时不显示时间：轮询已经停了，时间戳只会误导。 */
const clock = computed(() =>
  tone.value === "paused" ? "" : formatClockTime(props.lastAt),
);

const title = computed(() =>
  tone.value === "paused"
    ? t("autoRefreshPaused")
    : t("autoRefreshHint", { seconds: Math.round(props.intervalMs / 1000) }),
);
</script>

<template>
  <span
    v-if="tone !== 'idle'"
    class="refresh-badge"
    :data-tone="tone"
    :title="title"
  >
    <span class="dot" aria-hidden="true" />
    <span>{{ label }}</span>
    <span v-if="clock" class="clock">· {{ clock }}</span>
  </span>
</template>

<style scoped>
.refresh-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex: none;
  color: var(--success);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
}

.refresh-badge[data-tone="paused"] {
  color: var(--warning);
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  flex: none;
}

/* 等待下一轮：半透明呼吸，表示还活着但没在请求 */
.refresh-badge[data-tone="active"] .dot {
  opacity: 0.4;
  animation: refresh-breathe 2s ease-in-out infinite;
}

/* 请求中：实心快脉冲 */
.refresh-badge[data-tone="busy"] .dot {
  animation: refresh-pulse 0.9s ease-in-out infinite;
}

/* 暂停：常亮琥珀，不闪 */
.refresh-badge[data-tone="paused"] .dot {
  opacity: 1;
}

.clock {
  color: var(--muted);
  font-weight: 400;
  font-variant-numeric: tabular-nums;
}

@keyframes refresh-breathe {
  0%,
  100% {
    opacity: 0.25;
  }
  50% {
    opacity: 0.6;
  }
}

@keyframes refresh-pulse {
  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.45;
    transform: scale(0.7);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dot {
    animation: none !important;
  }
}
</style>
