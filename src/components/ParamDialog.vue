<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { t } from "../i18n";
import { buildParamValues } from "../lib/batchBuild";
import type { JobDetail } from "../types";
import ParamFields from "./ParamFields.vue";

const props = defineProps<{ job: JobDetail }>();

const emit = defineEmits<{
  submit: [params: Record<string, unknown>];
  cancel: [];
}>();

const values = ref<Record<string, unknown>>(
  buildParamValues(props.job.parameters ?? []),
);

function submit(): void {
  emit("submit", { ...values.value });
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
        <div>
          <h2>{{ t("buildWithParams") }}</h2>
          <p class="subtle">{{ job.fullName }}</p>
        </div>
        <button type="button" class="close" @click="emit('cancel')">×</button>
      </header>

      <div class="body">
        <ParamFields
          :parameters="job.parameters ?? []"
          :model-value="values"
          @update:model-value="values = $event"
        />
      </div>

      <footer>
        <button type="button" class="ghost" @click="emit('cancel')">
          {{ t("cancel") }}
        </button>
        <button type="button" class="primary" @click="submit">{{ t("start") }}</button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: grid;
  place-items: center;
  padding: 24px;
  background: color-mix(in srgb, #000 42%, transparent);
}

.modal {
  width: min(560px, 100%);
  max-height: min(80vh, 720px);
  display: flex;
  flex-direction: column;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.28);
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
  font-size: 16px;
}

.subtle {
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.close {
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: 22px;
  line-height: 1;
  padding: 0 4px;
}

.close:hover {
  color: var(--text);
}

.body {
  padding: 16px 18px;
  overflow: auto;
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
</style>
