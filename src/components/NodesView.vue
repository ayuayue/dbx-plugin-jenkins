<script setup lang="ts">
import { t } from "../i18n";
import type { NodeInfo } from "../types";

defineProps<{
  nodes: NodeInfo[];
  loading: boolean;
}>();

const emit = defineEmits<{ refresh: [] }>();
</script>

<template>
  <section class="nodes-view">
    <header class="head">
      <div>
        <h1>{{ t("nodes") }}</h1>
        <p class="subtle">{{ nodes.length }} {{ t("nodes") }}</p>
      </div>
      <button type="button" class="ghost" @click="emit('refresh')">{{ t("refresh") }}</button>
    </header>

    <p v-if="loading" class="empty">{{ t("loading") }}</p>
    <p v-else-if="!nodes.length" class="empty">{{ t("noNodes") }}</p>

    <table v-else class="table">
      <thead>
        <tr>
          <th>{{ t("nodeName") }}</th>
          <th>{{ t("nodeStatus") }}</th>
          <th>{{ t("executorsCol") }}</th>
          <th>{{ t("labels") }}</th>
          <th>{{ t("offlineReason") }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="node in nodes" :key="node.name">
          <td class="name">{{ node.name }}</td>
          <td>
            <span class="pill" :data-offline="node.offline">
              {{ node.offline ? t("offline") : t("online") }}
            </span>
          </td>
          <td class="executors">
            <span class="busy">{{ node.busy }}</span>
            <span class="of">/{{ node.numExecutors }}</span>
          </td>
          <td>
            <span v-for="label in node.labels ?? []" :key="label" class="label">{{ label }}</span>
            <span v-if="!node.labels?.length" class="muted">—</span>
          </td>
          <td class="reason">{{ node.offlineReason || "—" }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<style scoped>
.nodes-view {
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
  vertical-align: middle;
}

.table tbody tr:last-child td {
  border-bottom: 0;
}

.name {
  font-weight: 600;
}

.pill {
  padding: 3px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--success) 14%, transparent);
  color: var(--success);
  font-size: 10px;
  font-weight: 700;
}

.pill[data-offline="true"] {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

.executors {
  font-variant-numeric: tabular-nums;
}

.busy {
  font-weight: 700;
}

.of {
  color: var(--muted);
}

.label {
  display: inline-block;
  margin: 1px 4px 1px 0;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--panel-soft);
  color: var(--muted);
  font-size: 10px;
}

.muted,
.reason {
  color: var(--muted);
}
</style>
