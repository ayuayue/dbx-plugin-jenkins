<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive } from "vue";
import { t } from "../i18n";
import type { JobDetail } from "../types";

const props = defineProps<{ job: JobDetail }>();

const emit = defineEmits<{
  submit: [params: Record<string, unknown>];
  cancel: [];
}>();

const values = reactive<Record<string, any>>({});

for (const parameter of props.job.parameters ?? []) {
  const isBoolean = parameter.type === "BooleanParameterDefinition";
  values[parameter.name] = isBoolean
    ? parameter.defaultValue === "true"
    : parameter.defaultValue ?? "";
}

function submit(): void {
  emit("submit", { ...values });
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
        <label v-for="parameter in job.parameters ?? []" :key="parameter.name" class="field">
          <span class="label">
            <code>{{ parameter.name }}</code>
            <span class="type">{{ parameter.type.replace("ParameterDefinition", "") }}</span>
          </span>
          <span v-if="parameter.description" class="hint">{{ parameter.description }}</span>

          <select
            v-if="parameter.type === 'ChoiceParameterDefinition'"
            v-model="values[parameter.name]"
          >
            <option v-for="choice in parameter.choices ?? []" :key="choice" :value="choice">
              {{ choice }}
            </option>
          </select>

          <textarea
            v-else-if="parameter.type === 'TextParameterDefinition'"
            v-model="values[parameter.name]"
            rows="3"
          />

          <input
            v-else-if="parameter.type === 'BooleanParameterDefinition'"
            v-model="values[parameter.name]"
            type="checkbox"
            class="checkbox"
          />

          <input
            v-else-if="parameter.type === 'PasswordParameterDefinition'"
            v-model="values[parameter.name]"
            type="password"
          />

          <input v-else v-model="values[parameter.name]" type="text" />
        </label>
      </div>

      <footer>
        <button type="button" class="ghost" @click="emit('cancel')">{{ t("cancel") }}</button>
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
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.label code {
  font-size: 12px;
  font-weight: 700;
}

.type {
  color: var(--muted);
  font-size: 10px;
}

.hint {
  color: var(--muted);
  font-size: 11px;
  line-height: 1.5;
}

input:not(.checkbox),
select,
textarea {
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

input:focus,
select:focus,
textarea:focus {
  border-color: var(--accent);
}

.checkbox {
  width: 16px;
  height: 16px;
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
