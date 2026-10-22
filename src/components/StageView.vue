<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import { t } from "../i18n";
import { formatTimestamp } from "../lib/format";
import {
  commitMessageSummary,
  formatStageDuration,
  popoverPosition,
  shortCommitId,
  stageIntensity,
  stageTone,
} from "../lib/stage";
import { statusMeta } from "../lib/status";
import type { StageBuildInfo, StageInfo, StageMatrix } from "../types";

const props = defineProps<{
  job: string;
  matrix: StageMatrix | null;
  loading: boolean;
  locale: "zh" | "en";
  autoRefreshing: boolean;
}>();

const emit = defineEmits<{
  back: [];
  refresh: [];
  openBuild: [number: number];
}>();

const POPOVER_WIDTH = 360;
const POPOVER_MAX_HEIGHT = 320;
const POPOVER_GAP = 8;
const CLOSE_DELAY_MS = 120;

const stageNames = computed(() => props.matrix?.stageNames ?? []);

const maxDurations = computed(() =>
  stageNames.value.map((name) =>
    (props.matrix?.builds ?? []).reduce((max, build) => {
      const stage = stageOf(build, name);
      return Math.max(max, stage?.durationMillis ?? 0);
    }, 0),
  ),
);

const popoverBuild = ref<StageBuildInfo | null>(null);
const popoverEl = ref<HTMLElement | null>(null);
const popoverStyle = ref<Record<string, string>>({ top: "0px", left: "0px" });

let activeTrigger: HTMLElement | null = null;
let closeTimer: number | undefined;
let openedByClick = false;

function stageOf(build: StageBuildInfo, name: string): StageInfo | undefined {
  return (build.stages ?? []).find((stage) => stage.name === name);
}

function intensity(index: number, build: StageBuildInfo): number {
  const stage = stageOf(build, stageNames.value[index]);
  if (!stage || stage.durationMillis <= 0) return 0.1;
  return stageIntensity(stage.durationMillis, maxDurations.value[index]);
}

function average(index: number): number {
  return props.matrix?.averages[index]?.duration ?? 0;
}

function commitLabel(count: number): string {
  if (count <= 0) return t("noChanges");
  return count === 1
    ? t("commitOne", { count })
    : t("commitMany", { count });
}

function cellTitle(name: string, build: StageBuildInfo): string {
  const stage = stageOf(build, name);
  if (!stage) return `${name}: skipped`;
  return `${name}: ${stage.status} ${formatStageDuration(stage.durationMillis)}`;
}

function clearClose(): void {
  if (closeTimer !== undefined) {
    window.clearTimeout(closeTimer);
    closeTimer = undefined;
  }
}

function detachPopoverListeners(): void {
  window.removeEventListener("scroll", closePopover, true);
  window.removeEventListener("resize", closePopover);
  window.removeEventListener("keydown", onPopoverKeydown);
  document.removeEventListener("click", onDocumentClick, true);
}

function closePopover(): void {
  clearClose();
  detachPopoverListeners();
  popoverBuild.value = null;
  activeTrigger = null;
  openedByClick = false;
}

function onPopoverKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") closePopover();
}

function onDocumentClick(event: MouseEvent): void {
  const target = event.target as Node | null;
  if (!target) return;
  if (popoverEl.value?.contains(target) || activeTrigger?.contains(target)) return;
  closePopover();
}

function scheduleClose(): void {
  clearClose();
  closeTimer = window.setTimeout(() => closePopover(), CLOSE_DELAY_MS);
}

function positionPopover(trigger: HTMLElement): void {
  const triggerRect = trigger.getBoundingClientRect();
  const cellRect = trigger.closest("td")?.getBoundingClientRect();
  const anchorLeft = cellRect ? cellRect.right : triggerRect.right;
  const height = Math.min(popoverEl.value?.offsetHeight || POPOVER_MAX_HEIGHT, POPOVER_MAX_HEIGHT);
  const position = popoverPosition(
    { top: triggerRect.top, left: anchorLeft, width: 0, height: triggerRect.height },
    { width: POPOVER_WIDTH, height },
    { width: window.innerWidth, height: window.innerHeight },
    POPOVER_GAP,
  );
  popoverStyle.value = { top: `${position.top}px`, left: `${position.left}px` };
}

async function openPopover(build: StageBuildInfo, event: Event, byClick = false): Promise<void> {
  if (!build.changes?.length) return;
  const trigger = (event.currentTarget as HTMLElement | null) ?? activeTrigger;
  if (!trigger) return;

  clearClose();
  if (popoverBuild.value?.number !== build.number) {
    popoverBuild.value = build;
    activeTrigger = trigger;
    openedByClick = false;
    window.addEventListener("scroll", closePopover, true);
    window.addEventListener("resize", closePopover);
    window.addEventListener("keydown", onPopoverKeydown);
    document.addEventListener("click", onDocumentClick, true);
  }
  if (byClick) openedByClick = true;

  await nextTick();
  positionPopover(trigger);
}

function handleTriggerClick(build: StageBuildInfo, event: MouseEvent): void {
  if (popoverBuild.value?.number === build.number && openedByClick) {
    closePopover();
    return;
  }
  void openPopover(build, event, true);
}

function handleTriggerBlur(): void {
  clearClose();
  closeTimer = window.setTimeout(() => {
    const active = document.activeElement as Node | null;
    if (active && (popoverEl.value?.contains(active) || activeTrigger?.contains(active))) return;
    closePopover();
  }, CLOSE_DELAY_MS);
}

function onPopoverFocusOut(event: FocusEvent): void {
  const next = event.relatedTarget as Node | null;
  if (next && (popoverEl.value?.contains(next) || activeTrigger?.contains(next))) return;
  closePopover();
}

function openBuildFromPopover(): void {
  const build = popoverBuild.value;
  closePopover();
  if (build) emit("openBuild", build.number);
}

watch(
  () => props.matrix,
  (matrix) => {
    const current = popoverBuild.value;
    if (!current) return;
    const fresh = matrix?.builds.find((build) => build.number === current.number);
    if (fresh) {
      popoverBuild.value = fresh;
    } else {
      closePopover();
    }
  },
);

onBeforeUnmount(() => closePopover());
</script>

<template>
  <section class="stage-view">
    <header class="stage-head">
      <div class="title-block">
        <button type="button" class="back" @click="emit('back')">
          ← {{ t("backToJob") }}
        </button>
        <div class="title-row">
          <h1>{{ t("stageView") }}</h1>
          <span class="job-name">{{ job }}</span>
        </div>
        <p v-if="matrix" class="subtle">
          {{ t("recentBuilds", { count: matrix.builds.length }) }} ·
          {{ t("averageRun") }} ≈ {{ formatStageDuration(matrix.averageRun) }}
        </p>
      </div>
      <div class="stage-actions">
        <span v-if="autoRefreshing" class="auto-refresh">
          <span class="pulse" />
          {{ t("autoRefreshing") }}
        </span>
        <button type="button" class="ghost" @click="emit('refresh')">
          {{ t("refresh") }}
        </button>
      </div>
    </header>

    <p v-if="loading" class="empty">{{ t("loading") }}</p>
    <p v-else-if="!matrix || !stageNames.length" class="empty">
      {{ t("noStageData") }}
    </p>

    <div v-else class="stage-scroll">
      <table class="stage-table">
        <thead>
          <tr class="average-row">
            <th class="row-head">
              <span class="average-label">{{ t("averageStageTimes") }}</span>
            </th>
            <th v-for="(name, index) in stageNames" :key="name" class="average-cell">
              {{ formatStageDuration(average(index)) }}
            </th>
          </tr>
          <tr>
            <th class="row-head build-head">#</th>
            <th v-for="name in stageNames" :key="name" class="stage-head-cell">
              {{ name }}
            </th>
          </tr>
        </thead>

        <tbody>
          <tr
            v-for="build in matrix.builds"
            :key="build.number"
            class="build-row"
            :title="`#${build.number}`"
            @click="emit('openBuild', build.number)"
          >
            <td class="row-head build-cell">
              <div class="build-line">
                <span class="build-dot" :data-tone="statusMeta(build.status).tone" />
                <strong>#{{ build.number }}</strong>
              </div>
              <span class="build-time">{{ formatTimestamp(build.timestamp, locale) }}</span>
              <button
                v-if="build.changes?.length"
                type="button"
                class="commit-trigger"
                :aria-expanded="popoverBuild?.number === build.number"
                aria-haspopup="dialog"
                :title="t('viewChangesHint')"
                @mouseenter="openPopover(build, $event)"
                @mouseleave="scheduleClose()"
                @focus="openPopover(build, $event)"
                @blur="handleTriggerBlur()"
                @click.stop="handleTriggerClick(build, $event)"
              >
                {{ commitLabel(build.commitCount) }}
              </button>
              <span v-else class="build-commits">{{ commitLabel(build.commitCount) }}</span>
            </td>
            <td
              v-for="(name, index) in stageNames"
              :key="name"
              class="stage-cell"
              :data-tone="stageTone(stageOf(build, name)?.status)"
              :style="{ '--stage-alpha': intensity(index, build) }"
              :title="cellTitle(name, build)"
            >
              <span v-if="stageOf(build, name)">
                {{ formatStageDuration(stageOf(build, name)?.durationMillis ?? 0) }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <Teleport to="body">
      <div
        v-if="popoverBuild"
        ref="popoverEl"
        class="commit-pop"
        role="dialog"
        :aria-label="t('changes')"
        :style="popoverStyle"
        @mouseenter="clearClose()"
        @mouseleave="scheduleClose()"
        @focusout="onPopoverFocusOut"
      >
        <header class="commit-pop-head">
          {{ commitLabel(popoverBuild.changes?.length ?? 0) }}
        </header>
        <ul class="commit-list">
          <li v-for="item in popoverBuild.changes" :key="`${item.commitId}-${item.message}`">
            <code class="commit-id">{{ shortCommitId(item.commitId) }}</code>
            <div class="commit-body">
              <span class="commit-message" :title="item.message">
                {{ commitMessageSummary(item.message) || "—" }}
              </span>
              <span class="commit-author">{{ item.author || "—" }}</span>
            </div>
          </li>
        </ul>
        <button type="button" class="commit-detail" @click="openBuildFromPopover">
          {{ t("seeDetailPage") }}
        </button>
      </div>
    </Teleport>
  </section>
</template>

<style scoped>
.stage-view {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px;
  overflow: hidden;
}

.stage-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex: none;
}

.back {
  border: 0;
  background: transparent;
  color: var(--muted);
  padding: 0;
  margin-bottom: 6px;
  font-size: 12px;
}

.back:hover {
  color: var(--accent);
}

.title-row {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}

h1 {
  margin: 0;
  font-size: 20px;
}

.job-name {
  color: var(--muted);
  font-size: 13px;
}

.subtle {
  margin: 5px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.stage-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: none;
}

.auto-refresh {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--success);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.pulse {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--success);
  animation: stage-auto-refresh-pulse 1.4s ease-in-out infinite;
}

@keyframes stage-auto-refresh-pulse {
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

.ghost {
  flex: none;
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
  margin: 24px 0;
  color: var(--muted);
  font-size: 13px;
}

.stage-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
}

.stage-table {
  width: max-content;
  min-width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  font-size: 12px;
}

.stage-table th,
.stage-table td {
  border-right: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  padding: 7px 10px;
  text-align: center;
  white-space: nowrap;
}

.stage-table th:last-child,
.stage-table td:last-child {
  border-right: 0;
}

.stage-table thead th {
  position: sticky;
  top: 0;
  z-index: 3;
  background: var(--panel);
}

.average-row th {
  top: 0;
  height: 34px;
  z-index: 3;
  color: var(--muted);
  font-weight: 600;
  font-size: 11px;
}

.stage-table thead tr:last-child th {
  top: 34px;
}

.average-row th.row-head {
  z-index: 5;
  background: var(--panel-soft);
}

.average-label {
  display: block;
  text-align: left;
  white-space: normal;
}

.average-cell {
  min-width: 96px;
  color: var(--muted);
}

.stage-head-cell {
  min-width: 96px;
  max-width: 150px;
  font-weight: 700;
  vertical-align: bottom;
}

.build-head {
  vertical-align: middle;
}

.row-head {
  position: sticky;
  left: 0;
  z-index: 2;
  min-width: 190px;
  max-width: 190px;
  background: var(--panel);
  text-align: left !important;
}

thead .row-head {
  z-index: 5;
}

.build-row {
  cursor: pointer;
}

.build-row:hover td {
  filter: brightness(0.985);
}

.build-cell {
  padding: 8px 12px !important;
}

.build-line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.build-line strong {
  font-size: 12px;
}

.build-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--muted);
  flex: none;
}

.build-dot[data-tone="success"] {
  background: var(--success);
}

.build-dot[data-tone="danger"] {
  background: var(--danger);
}

.build-dot[data-tone="warning"] {
  background: #d99b1a;
}

.build-dot[data-tone="info"] {
  background: var(--accent);
}

.build-time,
.build-commits {
  display: block;
  margin-top: 2px;
  color: var(--muted);
  font-size: 10px;
}

.commit-trigger {
  display: block;
  margin-top: 2px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  font: inherit;
  font-size: 10px;
  text-align: left;
  text-decoration: underline dotted;
  text-underline-offset: 2px;
  cursor: pointer;
}

.commit-trigger:hover,
.commit-trigger:focus-visible,
.commit-trigger[aria-expanded="true"] {
  color: var(--accent);
  outline: none;
}

.stage-cell {
  min-width: 96px;
  height: 42px;
  font-variant-numeric: tabular-nums;
}

.stage-cell[data-tone="success"] {
  background: rgb(46 160 67 / var(--stage-alpha));
}

.stage-cell[data-tone="danger"] {
  background: rgb(207 34 46 / 0.16);
  color: var(--danger);
  font-weight: 600;
}

.stage-cell[data-tone="info"] {
  background: repeating-linear-gradient(
    135deg,
    rgb(35 114 210 / 0.16) 0 8px,
    rgb(35 114 210 / 0.28) 8px 16px
  );
  background-size: 22px 22px;
  animation: stage-stripes 1.1s linear infinite;
  color: var(--accent);
  font-weight: 600;
}

.stage-cell[data-tone="muted"] {
  background: var(--panel-soft);
  color: var(--muted);
}

@keyframes stage-stripes {
  from {
    background-position: 0 0;
  }
  to {
    background-position: 22px 0;
  }
}

.commit-pop {
  position: fixed;
  z-index: 1200;
  display: flex;
  flex-direction: column;
  width: 360px;
  max-height: 320px;
  padding: 10px 12px 8px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  box-shadow: 0 14px 36px rgb(15 23 42 / 0.18);
  color: var(--text);
  font-size: 12px;
  text-align: left;
}

.commit-pop-head {
  flex: none;
  color: var(--muted);
  font-size: 11px;
  font-weight: 700;
}

.commit-list {
  flex: 1;
  min-height: 0;
  margin: 8px 0;
  padding: 0;
  overflow: auto;
  list-style: none;
}

.commit-list li {
  display: flex;
  gap: 8px;
  padding: 7px 0;
  border-top: 1px solid var(--border);
}

.commit-list li:first-child {
  padding-top: 0;
  border-top: 0;
}

.commit-id {
  flex: none;
  align-self: flex-start;
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--panel-soft);
  color: var(--muted);
  font-family: var(--mono);
  font-size: 10px;
}

.commit-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.commit-message {
  overflow: hidden;
  color: var(--text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.commit-author {
  color: var(--muted);
  font-size: 10px;
}

.commit-detail {
  flex: none;
  padding: 8px 0 0;
  border: 0;
  border-top: 1px solid var(--border);
  background: transparent;
  color: var(--accent);
  font: inherit;
  font-size: 11px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}

.commit-detail:hover {
  text-decoration: underline;
}
</style>
