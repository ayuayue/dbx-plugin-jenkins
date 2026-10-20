<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { t } from "../i18n";
import { deleteConfirmationMatches } from "../lib/forms";

const props = defineProps<{
  target: string;
  kind: "job" | "view";
  fullName?: string;
  busy: boolean;
}>();

const emit = defineEmits<{
  confirm: [];
  cancel: [];
}>();

const input = ref("");

const matches = computed(() => deleteConfirmationMatches(input.value, props.target));
const warning = computed(() =>
  props.kind === "job" ? t("deleteWarning") : t("deleteViewWarning"),
);

function confirm(): void {
  if (!matches.value) return;
  emit("confirm");
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") emit("cancel");
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <div class="overlay" @click.self="emit('cancel')">
    <section class="modal">
      <header>
        <h2>{{ t("deleteTarget", { name: target }) }}</h2>
        <button type="button" class="close" @click="emit('cancel')">×</button>
      </header>

      <div class="body">
        <p class="warning">{{ warning }}</p>
        <p v-if="fullName && fullName !== target" class="subtle">{{ fullName }}</p>

        <label class="field">
          <span class="label">{{ t("deleteTypeName") }}</span>
          <input
            v-model="input"
            type="text"
            :placeholder="target"
            autofocus
            @keyup.enter="confirm"
          />
          <span v-if="input && !matches" class="mismatch">{{ t("deleteNameMismatch") }}</span>
        </label>
      </div>

      <footer>
        <button type="button" class="ghost" @click="emit('cancel')">{{ t("cancel") }}</button>
        <button type="button" class="danger" :disabled="!matches || busy" @click="confirm">
          {{ busy ? t("loading") : t("deleteJob") }}
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
  width: min(460px, 100%);
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
  gap: 12px;
}

.warning {
  margin: 0;
  color: var(--danger);
  font-size: 12px;
  line-height: 1.6;
}

.subtle {
  margin: 0;
  color: var(--muted);
  font-size: 11px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label {
  font-size: 12px;
  font-weight: 600;
}

input {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  background: var(--panel-soft);
  color: var(--text);
  font: inherit;
  font-size: 13px;
  outline: none;
}

input:focus {
  border-color: var(--accent);
}

.mismatch {
  color: var(--danger);
  font-size: 11px;
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

.danger:disabled {
  opacity: 0.45;
}
</style>
