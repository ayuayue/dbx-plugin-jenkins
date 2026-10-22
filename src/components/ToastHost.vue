<script setup lang="ts">
import { t } from "../i18n";
import type { ToastItem } from "../lib/toast";

defineProps<{
  items: ToastItem[];
}>();

const emit = defineEmits<{
  dismiss: [id: number];
}>();
</script>

<template>
  <div class="toast-host" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <div
        v-for="item in items"
        :key="item.id"
        class="toast"
        :class="item.tone"
        @click="emit('dismiss', item.id)"
      >
        <svg class="icon" viewBox="0 0 16 16" aria-hidden="true">
          <template v-if="item.tone === 'info'">
            <circle cx="8" cy="8" r="6.4" fill="none" stroke="currentColor" stroke-width="1.4" />
            <path d="M8 7.1v4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            <circle cx="8" cy="4.9" r="0.85" fill="currentColor" />
          </template>
          <path
            v-else
            d="M3.4 8.4l3 3 6.2-6.6"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
        <span class="text">{{ item.text }}</span>
        <button
          type="button"
          class="close"
          :aria-label="t('closeToast')"
          @click.stop="emit('dismiss', item.id)"
        >
          ×
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
/* 浮层不占布局，只在上层展示；压在顶栏（56px）下方，避免挡住连接状态与刷新按钮。 */
.toast-host {
  position: fixed;
  top: 68px;
  right: 12px;
  z-index: 70;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  width: max-content;
  min-width: 180px;
  max-width: min(340px, calc(100vw - 48px));
  padding: 9px 10px 9px 12px;
  border: 1px solid var(--border);
  border-left-width: 3px;
  border-radius: var(--radius-sm);
  background: var(--panel);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.16);
  font-size: 12px;
  line-height: 1.55;
  color: var(--text);
  pointer-events: auto;
  cursor: pointer;
}

.toast.success {
  border-left-color: var(--success);
}

.toast.success .icon {
  color: var(--success);
}

.toast.info {
  border-left-color: var(--accent);
}

.toast.info .icon {
  color: var(--accent);
}

.icon {
  width: 14px;
  height: 14px;
  flex: none;
  margin-top: 2px;
}

.text {
  flex: 1;
  min-width: 0;
  word-break: break-word;
}

.close {
  flex: none;
  margin: -2px -2px 0 0;
  padding: 0 2px;
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: 15px;
  line-height: 1.2;
}

.close:hover {
  color: var(--text);
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(12px);
}
</style>
