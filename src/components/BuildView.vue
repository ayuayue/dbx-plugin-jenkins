<script setup lang="ts">
import { t } from "../i18n";
import { formatDuration, formatTimestamp } from "../lib/format";
import { statusMeta } from "../lib/status";
import type { BuildDetail } from "../types";
import ConsoleView from "./ConsoleView.vue";

defineProps<{
  build: BuildDetail;
  jobFullName: string;
  readOnly: boolean;
  locale: "zh" | "en";
  autoRefreshing?: boolean;
}>();

const emit = defineEmits<{
  back: [];
  stop: [number: number];
  refresh: [];
}>();
</script>

<template>
  <section class="build-view">
    <header class="build-head">
      <div class="title-block">
        <button type="button" class="back" @click="emit('back')">← {{ t("backToJob") }}</button>
        <div class="title-row">
          <h1>#{{ build.number }}</h1>
          <span class="status-pill" :data-tone="statusMeta(build.status).tone">
            {{ t(statusMeta(build.status).key) }}
          </span>
        </div>
        <p class="full-name">{{ build.fullDisplayName || jobFullName }}</p>
        <p v-if="build.causes?.length" class="cause">
          {{ t("cause") }}: {{ build.causes.map((cause) => cause.shortDescription).join(" · ") }}
        </p>
      </div>
      <div class="actions">
        <span v-if="autoRefreshing" class="auto-refresh">
          <span class="pulse" aria-hidden="true"></span>
          {{ t("autoRefreshing") }}
        </span>
        <button type="button" class="ghost" @click="emit('refresh')">
          {{ t("refresh") }}
        </button>
        <button
          v-if="build.building && !readOnly"
          type="button"
          class="danger"
          @click="emit('stop', build.number)"
        >
          {{ t("stopBuild") }}
        </button>
      </div>
    </header>

    <div class="meta-grid">
      <div class="meta">
        <span class="meta-label">{{ t("status") }}</span>
        <strong>{{ t(statusMeta(build.status).key) }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">{{ t("startedAt") }}</span>
        <strong>{{ formatTimestamp(build.timestamp, locale) }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">{{ t("duration") }}</span>
        <strong>{{ formatDuration(build.duration) }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">ETA</span>
        <strong>{{ formatDuration(build.estimatedDuration) }}</strong>
      </div>
    </div>

    <section v-if="build.changeSet?.length" class="panel">
      <div class="panel-head">
        <h2>{{ t("changes") }}</h2>
        <span class="count">{{ build.changeSet.length }}</span>
      </div>
      <table class="changes">
        <thead>
          <tr>
            <th>Commit</th>
            <th>Author</th>
            <th>Message</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in build.changeSet" :key="item.commitId">
            <td><code>{{ item.commitId.slice(0, 8) }}</code></td>
            <td>{{ item.author || "—" }}</td>
            <td class="message">{{ item.message }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <section v-if="build.artifacts?.length" class="panel">
      <div class="panel-head">
        <h2>{{ t("artifacts") }}</h2>
        <span class="count">{{ build.artifacts.length }}</span>
      </div>
      <ul class="artifacts">
        <li v-for="artifact in build.artifacts" :key="artifact.relativePath">
          <span>{{ artifact.fileName }}</span>
          <code>{{ artifact.relativePath }}</code>
        </li>
      </ul>
    </section>

    <ConsoleView :job="jobFullName" :number="build.number" />
  </section>
</template>

<style scoped>
.build-view {
  flex: 1;
  min-width: 0;
  overflow: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.build-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.back {
  border: 0;
  background: transparent;
  color: var(--muted);
  padding: 0;
  margin-bottom: 8px;
  font-size: 12px;
}

.back:hover {
  color: var(--accent);
}

.title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

h1 {
  margin: 0;
  font-size: 22px;
}

.full-name {
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.cause {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.danger {
  border: 1px solid color-mix(in srgb, var(--danger) 45%, transparent);
  border-radius: var(--radius-sm);
  padding: 8px 13px;
  background: transparent;
  color: var(--danger);
  font-weight: 600;
}

.ghost {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 13px;
  background: transparent;
  color: var(--muted);
  font-weight: 600;
}

.auto-refresh {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--success);
  font-size: 11px;
  font-weight: 600;
}

.pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--success);
  animation: auto-refresh-pulse 1.4s ease-in-out infinite;
}

@keyframes auto-refresh-pulse {
  0%,
  100% {
    opacity: 0.35;
    transform: scale(0.8);
  }
  50% {
    opacity: 1;
    transform: scale(1);
  }
}

.status-pill {
  padding: 3px 9px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  background: var(--panel-soft);
  color: var(--muted);
}

.status-pill[data-tone="success"] {
  background: color-mix(in srgb, var(--success) 14%, transparent);
  color: var(--success);
}

.status-pill[data-tone="danger"] {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

.status-pill[data-tone="warning"] {
  background: #d99b1a1f;
  color: #b47d0d;
}

.status-pill[data-tone="info"] {
  background: var(--accent-soft);
  color: var(--accent);
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel);
}

.meta-label {
  color: var(--muted);
  font-size: 11px;
}

.meta strong {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px;
}

.panel-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 10px;
}

h2 {
  margin: 0;
  font-size: 14px;
}

.count {
  color: var(--muted);
  font-size: 12px;
}

.changes {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.changes th {
  text-align: left;
  color: var(--muted);
  font-weight: 600;
  padding: 7px 8px;
  border-bottom: 1px solid var(--border);
}

.changes td {
  padding: 8px;
  border-bottom: 1px solid var(--border);
  vertical-align: top;
}

.message {
  line-height: 1.5;
}

.artifacts {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
}

.artifacts li {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 9px;
  border-radius: var(--radius-sm);
  background: var(--panel-soft);
}

.artifacts code {
  color: var(--muted);
}
</style>
