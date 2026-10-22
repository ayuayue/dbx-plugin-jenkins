<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { t } from "../i18n";
import {
  buildCreateViewPayload,
  buildViewUpdateFields,
  uniqueList,
  validateName,
} from "../lib/forms";
import type {
  CreateViewPayload,
  ViewConfigResponse,
  ViewInfo,
  ViewUpdateFields,
} from "../types";

const props = defineProps<{
  mode: "create" | "edit";
  view?: ViewInfo | null;
  config?: ViewConfigResponse | null;
  availableJobs: string[];
  existingNames?: string[];
  busy: boolean;
}>();

const emit = defineEmits<{
  create: [payload: CreateViewPayload];
  update: [fields: ViewUpdateFields];
  saveXml: [xml: string];
  cancel: [];
}>();

const tab = ref<"form" | "xml">("form");
const name = ref(props.config?.fields?.name ?? props.view?.name ?? "");
const description = ref(props.config?.fields?.description ?? "");
const selected = ref<string[]>(props.config?.fields?.jobNames ?? []);
const includeRegex = ref(props.config?.fields?.includeRegex ?? "");
const recurse = ref(props.config?.fields?.recurse ?? false);
const xml = ref(props.config?.xml ?? "");
const filter = ref("");
const error = ref("");

const isEdit = computed(() => props.mode === "edit");
const isListView = computed(() => !isEdit.value || Boolean(props.config?.fields));

const original = computed(() => ({
  name: props.config?.fields?.name ?? props.view?.name ?? "",
  description: props.config?.fields?.description ?? "",
  jobNames: props.config?.fields?.jobNames ?? [],
  includeRegex: props.config?.fields?.includeRegex ?? "",
  recurse: props.config?.fields?.recurse ?? false,
}));

const visibleJobs = computed(() => {
  const needle = filter.value.trim().toLowerCase();
  const list = uniqueList(props.availableJobs);
  if (!needle) return list;
  return list.filter((job) => job.toLowerCase().includes(needle));
});

const selectedSet = computed(() => new Set(selected.value));

function toggleJob(job: string): void {
  if (selectedSet.value.has(job)) {
    selected.value = selected.value.filter((item) => item !== job);
  } else {
    selected.value = [...selected.value, job];
  }
}

function selectAll(): void {
  selected.value = uniqueList([...selected.value, ...visibleJobs.value]);
}

function clearAll(): void {
  selected.value = [];
}

function submit(): void {
  error.value = "";
  if (!isEdit.value) {
    const nameError = validateName(name.value);
    if (nameError) {
      error.value = t(nameError);
      return;
    }
    emit(
      "create",
      buildCreateViewPayload({
        name: name.value,
        description: description.value,
        jobs: selected.value,
        includeRegex: includeRegex.value,
        recurse: recurse.value,
      }),
    );
    return;
  }

  if (tab.value === "xml") {
    if (!xml.value.trim()) {
      error.value = t("xmlRequired");
      return;
    }
    emit("saveXml", xml.value);
    return;
  }

  const fields = buildViewUpdateFields(
    {
      name: name.value,
      description: description.value,
      jobNames: selected.value,
      includeRegex: includeRegex.value,
      recurse: recurse.value,
    },
    original.value,
  );
  if (fields.name) {
    const nameError = validateName(fields.name);
    if (nameError) {
      error.value = t(nameError);
      return;
    }
    const taken = (props.existingNames ?? []).some((item) => item === fields.name);
    if (taken) {
      error.value = t("viewNameExists", { name: fields.name });
      return;
    }
  }
  if (Object.keys(fields).length === 0) {
    emit("cancel");
    return;
  }
  emit("update", fields);
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
          <h2>{{ isEdit ? t("editViewTitle") : t("newViewTitle") }}</h2>
          <p class="subtle">{{ isEdit ? view?.name : t("views") }}</p>
        </div>
        <button type="button" class="close" @click="emit('cancel')">×</button>
      </header>

      <div v-if="isEdit" class="tabs">
        <button
          v-if="isListView"
          type="button"
          :class="{ active: tab === 'form' }"
          @click="tab = 'form'"
        >
          {{ t("basicTab") }}
        </button>
        <button
          type="button"
          :class="{ active: tab === 'xml' || !isListView }"
          @click="tab = 'xml'"
        >
          {{ t("xmlTab") }}
        </button>
      </div>

      <div class="body">
        <template v-if="!isEdit">
          <label class="field">
            <span class="label">{{ t("viewName") }}</span>
            <input v-model="name" type="text" :placeholder="t('viewName')" autofocus />
          </label>
        </template>

        <template v-if="!isEdit || (tab === 'form' && isListView)">
          <label v-if="isEdit" class="field">
            <span class="label">{{ t("viewName") }}</span>
            <input v-model="name" type="text" />
            <span class="hint">{{ t("renameHint") }}</span>
          </label>

          <label class="field">
            <span class="label">{{ t("description") }}</span>
            <textarea v-model="description" rows="2" />
          </label>

          <div class="field">
            <div class="jobs-head">
              <span class="label">{{ t("jobsInView") }} · {{ selected.length }}</span>
              <div class="jobs-tools">
                <input v-model="filter" type="search" :placeholder="t('filterJobsInView')" />
                <button type="button" class="mini" @click="selectAll">{{ t("selectAll") }}</button>
                <button type="button" class="mini" @click="clearAll">{{ t("clearSelection") }}</button>
              </div>
            </div>

            <div class="job-picker">
              <p v-if="!visibleJobs.length" class="hint">{{ t("noJobsToPick") }}</p>
              <label v-for="job in visibleJobs" :key="job" class="job-option">
                <input
                  type="checkbox"
                  class="checkbox"
                  :checked="selectedSet.has(job)"
                  @change="toggleJob(job)"
                />
                <span>{{ job }}</span>
              </label>
            </div>
          </div>

          <label class="field">
            <span class="label">{{ t("includeRegex") }}</span>
            <input v-model="includeRegex" type="text" placeholder="^release-.*" />
            <span class="hint">{{ t("includeRegexHint") }}</span>
          </label>

          <label class="field checkbox-field">
            <input v-model="recurse" type="checkbox" class="checkbox" />
            <span>{{ t("recurse") }}</span>
          </label>
        </template>

        <template v-else-if="isEdit">
          <p class="hint">{{ isListView ? t("xmlHint") : t("viewNotListView") }}</p>
          <textarea v-model="xml" rows="18" class="code" spellcheck="false" />
        </template>
      </div>

      <footer>
        <span v-if="error" class="error">{{ error }}</span>
        <button type="button" class="ghost" @click="emit('cancel')">{{ t("cancel") }}</button>
        <button type="button" class="primary" :disabled="busy" @click="submit">
          {{ busy ? t("loading") : isEdit ? t("save") : t("create") }}
        </button>
      </footer>
    </section>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 30;
  display: grid;
  place-items: center;
  padding: 24px;
  background: color-mix(in srgb, #000 42%, transparent);
}

.modal {
  width: min(680px, 100%);
  max-height: min(88vh, 860px);
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

.tabs {
  display: flex;
  gap: 4px;
  padding: 10px 18px 0;
  border-bottom: 1px solid var(--border);
}

.tabs button {
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--muted);
  padding: 7px 12px 9px;
  font-size: 12px;
  font-weight: 600;
}

.tabs button.active {
  border-bottom-color: var(--accent);
  color: var(--accent);
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

.checkbox-field {
  flex-direction: row;
  align-items: center;
  gap: 9px;
}

.label {
  font-size: 12px;
  font-weight: 600;
}

.hint {
  color: var(--muted);
  font-size: 11px;
  line-height: 1.5;
  margin: 0;
}

.jobs-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.jobs-tools {
  display: flex;
  align-items: center;
  gap: 6px;
}

.jobs-tools input {
  width: 150px;
}

.mini {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  padding: 4px 8px;
  font-size: 11px;
}

.mini:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.job-picker {
  max-height: 200px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: var(--panel-soft);
}

.job-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 6px;
  border-radius: 5px;
  font-size: 12px;
}

.job-option:hover {
  background: var(--accent-soft);
}

.checkbox {
  width: 15px;
  height: 15px;
}

input:not(.checkbox),
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
textarea:focus {
  border-color: var(--accent);
}

.code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre;
  tab-size: 2;
}

footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 18px;
  border-top: 1px solid var(--border);
}

.error {
  margin-right: auto;
  color: var(--danger);
  font-size: 12px;
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

.primary:disabled {
  opacity: 0.6;
}
</style>
