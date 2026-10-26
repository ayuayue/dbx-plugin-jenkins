<script setup lang="ts">
import { computed } from "vue";
import { t } from "../i18n";
import type { RefreshUi } from "../lib/refreshStatus";
import type { ExecutorInfo, Overview } from "../types";
import ExecutorList from "./ExecutorList.vue";
import RefreshBadge from "./RefreshBadge.vue";

const props = defineProps<{
  overview: Overview | null;
  loading: boolean;
  executors: ExecutorInfo[];
  loadingExecutors: boolean;
  refresh: RefreshUi;
}>();

const emit = defineEmits<{ refresh: []; openExecutor: [row: ExecutorInfo] }>();

const executorPercent = computed(() => {
  if (!props.overview || props.overview.executorsTotal <= 0) return 0;
  return Math.min(
    100,
    Math.round((props.overview.executorsBusy / props.overview.executorsTotal) * 100),
  );
});
</script>

<template>
  <section class="overview">
    <header class="head">
      <div>
        <h1>{{ t("overview") }}</h1>
        <p v-if="overview" class="subtle">
          {{ overview.nodeDescription || "Jenkins" }} · {{ t("version") }}
          {{ overview.version || "?" }}
        </p>
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

    <template v-else-if="overview">
      <div class="cards">
        <article class="card">
          <span class="card-label">{{ t("executors") }}</span>
          <strong class="card-value">
            {{ overview.executorsBusy }}<span class="of">/{{ overview.executorsTotal }}</span>
          </strong>
          <div class="bar"><i :style="{ width: `${executorPercent}%` }" /></div>
          <span class="card-hint">{{ t("busy") }}</span>
        </article>

        <article class="card">
          <span class="card-label">{{ t("inQueue") }}</span>
          <strong class="card-value">{{ overview.queueLength }}</strong>
          <span class="card-hint">{{ t("queue") }}</span>
        </article>

        <article class="card">
          <span class="card-label">{{ t("nodesOnline") }}</span>
          <strong class="card-value">
            {{ overview.nodesOnline }}<span class="of">/{{ overview.nodesTotal }}</span>
          </strong>
          <span class="card-hint">{{ t("nodes") }}</span>
        </article>

        <article class="card">
          <span class="card-label">{{ t("mode") }}</span>
          <strong class="card-value mode" :data-quiet="overview.quietingDown">
            {{ overview.quietingDown ? t("quietMode") : t("normalMode") }}
          </strong>
          <span class="card-hint">{{ overview.mode || "NORMAL" }}</span>
        </article>
      </div>

      <ExecutorList
        :rows="executors"
        :loading="loadingExecutors"
        @open="emit('openExecutor', $event)"
      />

      <section class="panel">
        <h2>{{ t("connection") }}</h2>
        <dl class="facts">
          <div><dt>{{ t("user") }}</dt><dd>{{ overview.user || t("anonymous") }}</dd></div>
          <div><dt>{{ t("version") }}</dt><dd>{{ overview.version || "—" }}</dd></div>
          <div><dt>{{ t("mode") }}</dt><dd>{{ overview.mode || "—" }}</dd></div>
          <div><dt>{{ t("nodeDescription") }}</dt><dd>{{ overview.nodeDescription || "—" }}</dd></div>
        </dl>
      </section>
    </template>
  </section>
</template>

<style scoped>
.overview {
  flex: 1;
  min-width: 0;
  overflow: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
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
  gap: 10px;
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

.cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background:
    radial-gradient(120% 130% at 100% 0%, var(--accent-soft), transparent 60%),
    var(--panel);
}

.card-label {
  color: var(--muted);
  font-size: 11px;
}

.card-value {
  font-size: 26px;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

.card-value.mode {
  font-size: 18px;
}

.card-value.mode[data-quiet="true"] {
  color: #b47d0d;
}

.of {
  margin-left: 3px;
  color: var(--muted);
  font-size: 14px;
  font-weight: 500;
}

.card-hint {
  color: var(--muted);
  font-size: 11px;
}

.bar {
  height: 5px;
  border-radius: 999px;
  background: var(--panel-soft);
  overflow: hidden;
}

.bar i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--accent), var(--success));
}

.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px;
}

h2 {
  margin: 0 0 12px;
  font-size: 14px;
}

.facts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin: 0;
}

.facts div {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  background: var(--panel-soft);
}

dt {
  color: var(--muted);
  font-size: 11px;
}

dd {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}
</style>
