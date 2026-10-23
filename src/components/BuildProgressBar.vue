<script setup lang="ts">
import { computed } from "vue";
import type { BuildProgress } from "../lib/progress";

const props = defineProps<{ progress: BuildProgress | null }>();

/** 不确定态固定 40% 宽的滑块循环滑动。 */
const fillWidth = computed(() =>
  props.progress?.indeterminate ? "40%" : `${props.progress?.percent ?? 0}%`,
);
</script>

<template>
  <span
    v-if="progress"
    class="build-progress"
    :class="{ indeterminate: progress.indeterminate }"
  >
    <span class="track">
      <span class="fill" :style="{ width: fillWidth }" />
    </span>
    <span class="text">{{ progress.indeterminate ? "…" : `${progress.percent}%` }}</span>
  </span>
  <span v-else class="build-progress idle">—</span>
</template>

<style scoped>
.build-progress {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.build-progress.idle {
  color: var(--muted);
}

.track {
  position: relative;
  display: block;
  width: 76px;
  height: 6px;
  border-radius: 999px;
  background: var(--panel-soft);
  overflow: hidden;
}

.fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--accent);
  transition: width 0.6s ease;
}

/* 没有估算耗时：滑块循环滑动，表示「进行中但不知道还剩多少」。 */
.build-progress.indeterminate .fill {
  position: absolute;
  left: 0;
  top: 0;
  animation: progress-slide 1.2s ease-in-out infinite;
}

@keyframes progress-slide {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(250%);
  }
}

.text {
  min-width: 30px;
  color: var(--muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  transition: color 0.2s ease;
}

.build-progress:not(.idle) .text {
  color: var(--text);
}
</style>
