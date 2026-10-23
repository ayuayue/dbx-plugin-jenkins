<script setup lang="ts">
import { t } from "../i18n";
import { formatTimestamp } from "../lib/format";
import type { RefreshUi } from "../lib/refreshStatus";
import type { QueueItem } from "../types";
import RefreshBadge from "./RefreshBadge.vue";

defineProps<{
  items: QueueItem[];
  loading: boolean;
  readOnly: boolean;
  locale: "zh" | "en";
  refresh: RefreshUi;
}>();

const emit = defineEmits<{
  cancel: [id: number];
  refresh: [];
}>();
</script>

<template>
  <section class="queue-view">
    <header class="head">
      <div>
        <h1>{{ t("queue") }}</h1>
        <p class="subtle">{{ items.length }} {{ t("inQueue") }}</p>
      </div>
      <div class="head-actions">
        <RefreshBadge
          :mode="refresh.mode"
          :paused="refresh.paused"
          :in-flight="refresh.inFlight"
          :last-at="refresh.lastAt"
          :interval-ms="refresh.intervalMs"
        />
        <button
          type="button"
          class="ghost"
          :class="{ done: refresh.refreshed }"
          :disabled="refresh.refreshing"
          @click="emit('refresh')"
        >
          {{
            refresh.refreshing
              ? t("refreshing")
              : refresh.refreshed
                ? t("refreshed")
                : t("refresh")
          }}
        </button>
      </div>
    </header>

    <p v-if="loading" class="empty">{{ t("loading") }}</p>
    <p v-else-if="!items.length" class="empty">{{ t("queueEmpty") }}</p>

    <table v-else class="table">
      <thead>
        <tr>
          <th>#</th>
          <th>{{ t("jobs") }}</th>
          <th>{{ t("why") }}</th>
          <th>{{ t("waitingSince") }}</th>
          <th>{{ t("cause") }}</th>
          <th />
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="item.id">
          <td class="number">{{ item.id }}</td>
          <td class="job">
            {{ item.jobName }}
            <span v-if="item.stuck" class="badge danger">stuck</span>
            <span v-else-if="item.blocked" class="badge">blocked</span>
          </td>
          <td class="why">{{ item.why || "—" }}</td>
          <td>{{ formatTimestamp(item.inQueueSince, locale) }}</td>
          <td class="cause">{{ item.cause || "—" }}</td>
          <td class="action">
            <button
              v-if="!readOnly"
              type="button"
              class="danger-link"
              @click="emit('cancel', item.id)"
            >
              {{ t("cancel") }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<style scoped>
.queue-view {
  flex: 1;
  min-width: 0;
  overflow: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.head-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: none;
}

h1 {
  margin: 0;
  font-size: 20px;
}

.subtle {
  margin: 5px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.ghost {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 7px 12px;
  background: transparent;
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
}

.ghost:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.ghost:disabled {
  opacity: 0.55;
  cursor: default;
  border-color: var(--border);
  color: var(--muted);
}

.ghost.done {
  border-color: var(--success);
  color: var(--success);
}

.empty {
  color: var(--muted);
  font-size: 13px;
}

.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

.table th {
  text-align: left;
  color: var(--muted);
  font-weight: 600;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--panel-soft);
}

.table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  vertical-align: top;
}

.table tbody tr:last-child td {
  border-bottom: 0;
}

.number {
  font-variant-numeric: tabular-nums;
  color: var(--muted);
}

.job {
  font-weight: 600;
  white-space: nowrap;
}

.why,
.cause {
  max-width: 320px;
  color: var(--muted);
  line-height: 1.5;
}

.badge {
  margin-left: 6px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--panel-soft);
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
}

.badge.danger {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

.action {
  text-align: right;
}

.danger-link {
  border: 0;
  background: transparent;
  color: var(--danger);
  font-size: 12px;
  font-weight: 600;
  padding: 2px 4px;
}

.danger-link:hover {
  text-decoration: underline;
}
</style>
