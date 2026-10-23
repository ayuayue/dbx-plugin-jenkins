<script setup lang="ts">
import { onBeforeUnmount, onMounted } from "vue";
import { t } from "../i18n";

const props = defineProps<{
  title: string;
  /** 操作对象，例如任务全名或排队项所属任务。 */
  detail?: string;
  hint?: string;
  confirmLabel: string;
  busy?: boolean;
}>();

const emit = defineEmits<{
  confirm: [];
  cancel: [];
}>();

function cancel(): void {
  if (props.busy) return;
  emit("cancel");
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") cancel();
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <div class="overlay" @click.self="cancel">
    <section class="modal" role="alertdialog" aria-modal="true">
      <header>
        <h2>{{ title }}</h2>
        <button type="button" class="close" :disabled="busy" @click="cancel">×</button>
      </header>

      <div class="body">
        <p v-if="detail" class="detail">{{ detail }}</p>
        <p v-if="hint" class="hint">{{ hint }}</p>
      </div>

      <footer>
        <button type="button" class="ghost" autofocus :disabled="busy" @click="cancel">
          {{ t("cancel") }}
        </button>
        <button type="button" class="danger" :disabled="busy" @click="emit('confirm')">
          {{ busy ? t("loading") : confirmLabel }}
        </button>
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
  width: min(420px, 100%);
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
  gap: 8px;
}

.detail {
  margin: 0;
  font-size: 13px;
  word-break: break-all;
}

.hint {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}

footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 18px;
  border-top: 1px solid var(--border);
}

.ghost,
.danger {
  border-radius: var(--radius-sm);
  padding: 8px 14px;
  font-weight: 600;
}

.ghost {
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text);
}

.danger {
  border: 0;
  background: var(--danger);
  color: #fff;
}

.ghost:disabled,
.danger:disabled,
.close:disabled {
  opacity: 0.45;
}
</style>
