<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { t } from "../i18n";
import { groupJobs } from "../lib/jobs";
import { statusMeta } from "../lib/status";
import { tabWheelDelta } from "../lib/tabs";
import { isNamedView } from "../lib/viewRefresh";
import type { JobSummary, ViewInfo } from "../types";

const props = defineProps<{
  jobs: JobSummary[];
  selected: string;
  folder: string;
  loading: boolean;
  views: ViewInfo[];
  activeView: string;
  readOnly: boolean;
}>();

const emit = defineEmits<{
  select: [job: JobSummary];
  openFolder: [job: JobSummary];
  navigate: [path: string];
  selectView: [name: string];
  refresh: [];
  createJob: [];
  createView: [];
  editView: [];
  deleteView: [];
}>();

const filter = ref("");
const tabsEl = ref<HTMLElement | null>(null);
const hasOverflow = ref(false);
const canScrollLeft = ref(false);
const canScrollRight = ref(false);
const menuOpen = ref(false);

const canManageView = computed(() => isNamedView(props.activeView));

const crumbs = computed(() => (props.folder ? props.folder.split("/") : []));
const rootLabel = computed(() =>
  props.activeView && props.activeView.toLowerCase() !== "all"
    ? props.activeView
    : t("rootFolder"),
);

function viewLabel(view: ViewInfo): string {
  return view.name && view.name.toLowerCase() !== "all"
    ? view.name
    : t("rootFolder");
}

const visibleJobs = computed(() => {
  const needle = filter.value.trim().toLowerCase();
  if (!needle) return props.jobs;
  return props.jobs.filter((job) => job.name.toLowerCase().includes(needle));
});

const groups = computed(() => groupJobs(visibleJobs.value));

function goToDepth(depth: number): void {
  emit("navigate", crumbs.value.slice(0, depth).join("/"));
}

function updateScrollState(): void {
  const element = tabsEl.value;
  if (!element) {
    hasOverflow.value = false;
    canScrollLeft.value = false;
    canScrollRight.value = false;
    return;
  }
  const maxScroll = element.scrollWidth - element.clientWidth;
  hasOverflow.value = maxScroll > 1;
  canScrollLeft.value = element.scrollLeft > 1;
  canScrollRight.value = element.scrollLeft < maxScroll - 1;
}

function scrollTabs(direction: -1 | 1): void {
  const element = tabsEl.value;
  if (!element) return;
  element.scrollBy({
    left: direction * Math.max(96, element.clientWidth * 0.7),
    behavior: "smooth",
  });
}

function centerActiveTab(): void {
  const element = tabsEl.value;
  if (!element) return;
  const active = element.querySelector<HTMLElement>('[aria-selected="true"]');
  active?.scrollIntoView({
    inline: "center",
    block: "nearest",
    behavior: "smooth",
  });
  window.setTimeout(updateScrollState, 350);
}

/** 鼠标滚轮纵向滚动时，把 Tab 条横向滚起来；到边界时放行，交回浏览器默认滚动。 */
function onTabsWheel(event: WheelEvent): void {
  const element = tabsEl.value;
  if (!element) return;
  const delta = tabWheelDelta({
    deltaX: event.deltaX,
    deltaY: event.deltaY,
    deltaMode: event.deltaMode,
    scrollLeft: element.scrollLeft,
    scrollWidth: element.scrollWidth,
    clientWidth: element.clientWidth,
  });
  if (!delta) return;
  event.preventDefault();
  element.scrollLeft += delta;
  updateScrollState();
}

let wheelTarget: HTMLElement | null = null;

function unbindWheel(): void {
  wheelTarget?.removeEventListener("wheel", onTabsWheel);
  wheelTarget = null;
}

/** Tab 条由 v-if 控制，视图加载完成后才出现，所以每次 ref 变化都要重绑。 */
function bindWheel(): void {
  unbindWheel();
  const element = tabsEl.value;
  if (!element) return;
  element.addEventListener("wheel", onTabsWheel, { passive: false });
  wheelTarget = element;
}

onMounted(() => {
  void nextTick(() => {
    updateScrollState();
    centerActiveTab();
    bindWheel();
  });
  window.addEventListener("resize", updateScrollState);
  document.addEventListener("click", onDocumentClick);
});

onBeforeUnmount(() => {
  unbindWheel();
  window.removeEventListener("resize", updateScrollState);
  document.removeEventListener("click", onDocumentClick);
});

watch(
  () => props.activeView,
  () => {
    filter.value = "";
    menuOpen.value = false;
    void nextTick(centerActiveTab);
  },
);

function toggleMenu(): void {
  menuOpen.value = !menuOpen.value;
}

function closeMenu(): void {
  menuOpen.value = false;
}

function onDocumentClick(event: MouseEvent): void {
  if (!menuOpen.value) return;
  const target = event.target as HTMLElement | null;
  if (target?.closest(".view-menu-wrap")) return;
  menuOpen.value = false;
}

function requestEditView(): void {
  closeMenu();
  emit("editView");
}

function requestDeleteView(): void {
  closeMenu();
  emit("deleteView");
}

watch(
  () => props.views.length,
  () => void nextTick(centerActiveTab),
);

watch(tabsEl, () => bindWheel());
</script>

<template>
  <aside class="job-list">
    <div class="list-head">
      <input v-model="filter" type="search" :placeholder="t('searchJobs')" />
      <button
        v-if="!readOnly"
        type="button"
        class="icon-button"
        :title="t('createJob')"
        @click="emit('createJob')"
      >
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path
            d="M8 3v10M3 8h10"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
          />
        </svg>
      </button>
      <button type="button" class="icon-button" :title="t('refresh')" @click="emit('refresh')">
        <svg viewBox="0 0 16 16" aria-hidden="true">
          <path
            d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9M13.5 2.5V6h-3.5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
    </div>

    <div
      v-if="views.length"
      class="view-tabs-shell"
      :class="{ overflowing: hasOverflow }"
    >
      <button
        v-if="hasOverflow"
        type="button"
        class="tab-arrow"
        :disabled="!canScrollLeft"
        :aria-label="t('scrollViewsLeft')"
        @click="scrollTabs(-1)"
      >
        ‹
      </button>

      <div
        ref="tabsEl"
        class="view-tabs"
        role="tablist"
        :aria-label="t('views')"
        @scroll="updateScrollState"
      >
        <button
          v-for="view in views"
          :key="view.name || '__all'"
          type="button"
          role="tab"
          class="view-tab"
          :class="{ active: view.name === activeView }"
          :title="view.description || view.name || t('rootFolder')"
          :aria-selected="view.name === activeView"
          @click="emit('selectView', view.name)"
        >
          {{ viewLabel(view) }}
        </button>
      </div>

      <button
        v-if="hasOverflow"
        type="button"
        class="tab-arrow"
        :disabled="!canScrollRight"
        :aria-label="t('scrollViewsRight')"
        @click="scrollTabs(1)"
      >
        ›
      </button>
    </div>

    <div class="crumbs">
      <div class="crumb-trail">
        <button type="button" @click="emit('navigate', '')">{{ rootLabel }}</button>
        <template v-for="(crumb, index) in crumbs" :key="index">
          <span class="sep">/</span>
          <button type="button" @click="goToDepth(index + 1)">{{ crumb }}</button>
        </template>
      </div>

      <div v-if="!readOnly" class="view-menu-wrap">
        <button
          type="button"
          class="crumb-action"
          :title="t('createView')"
          @click.stop="emit('createView')"
        >
          ＋
        </button>
        <button
          v-if="canManageView"
          type="button"
          class="crumb-action"
          :title="t('manageView')"
          @click.stop="toggleMenu"
        >
          ⋯
        </button>
        <div v-if="menuOpen && canManageView" class="view-menu">
          <button type="button" @click="requestEditView">{{ t("editView") }}</button>
          <button type="button" class="danger" @click="requestDeleteView">
            {{ t("deleteView") }}
          </button>
        </div>
      </div>
    </div>

    <div class="list-body">
      <template v-if="groups.folders.length">
        <div class="group-head">
          <span>{{ t("folders") }}</span>
          <span class="group-count">{{ groups.folders.length }}</span>
        </div>
        <button
          v-for="job in groups.folders"
          :key="job.fullName"
          type="button"
          class="job-row"
          :class="{ selected: job.fullName === selected }"
          :title="job.fullName"
          @click="emit('openFolder', job)"
        >
          <span class="dot" :data-tone="statusMeta(job.status).tone" />
          <span class="name">{{ job.name }}</span>
          <span class="chev">›</span>
        </button>
      </template>

      <template v-if="groups.jobs.length">
        <div class="group-head">
          <span>{{ t("jobsGroup") }}</span>
          <span class="group-count">{{ groups.jobs.length }}</span>
        </div>
        <button
          v-for="job in groups.jobs"
          :key="job.fullName"
          type="button"
          class="job-row"
          :class="{ selected: job.fullName === selected }"
          :title="job.fullName"
          @click="emit('select', job)"
        >
          <span class="dot" :data-tone="statusMeta(job.status).tone" />
          <span class="name">{{ job.name }}</span>
          <span class="last">#{{ job.lastBuild?.number ?? "—" }}</span>
        </button>
      </template>

      <p v-if="loading" class="empty">{{ t("loading") }}</p>
      <p v-else-if="!visibleJobs.length" class="empty">
        {{ filter ? t("noMatches") : props.folder ? t("noJobs") : t("noViewJobs") }}
      </p>
    </div>
  </aside>
</template>

<style scoped>
.job-list {
  width: 320px;
  flex: none;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-right: 1px solid var(--border);
  background: var(--panel);
}

.list-head {
  display: flex;
  gap: 8px;
  padding: 12px;
}

.list-head input {
  flex: 1;
  min-width: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 7px 10px;
  background: var(--panel-soft);
  color: var(--text);
  font: inherit;
  outline: none;
}

.list-head input:focus {
  border-color: var(--accent);
}

.icon-button {
  flex: none;
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
}

.icon-button svg {
  width: 16px;
  height: 16px;
}

.icon-button:hover {
  color: var(--accent);
  border-color: var(--accent);
}

.view-tabs-shell {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 6px 8px;
  border-bottom: 1px solid var(--border);
}

.view-tabs-shell.overflowing .view-tabs {
  mask-image: linear-gradient(
    to right,
    transparent 0,
    #000 18px,
    #000 calc(100% - 18px),
    transparent 100%
  );
}

.view-tabs {
  flex: 1;
  min-width: 0;
  display: flex;
  gap: 4px;
  padding: 0 4px;
  overflow-x: auto;
  scrollbar-width: none;
}

.view-tabs::-webkit-scrollbar {
  display: none;
}

.tab-arrow {
  flex: none;
  width: 22px;
  height: 24px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--panel-soft);
  color: var(--text);
  font-size: 15px;
  line-height: 1;
}

.tab-arrow:hover:not(:disabled) {
  border-color: var(--accent);
  background: var(--accent-soft);
  color: var(--accent);
}

.tab-arrow:disabled {
  opacity: 0.3;
  cursor: default;
}

.view-tab {
  flex: none;
  border: 0;
  border-radius: 999px;
  padding: 5px 10px;
  background: transparent;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.view-tab:focus {
  outline: none;
}

.view-tab:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}

.view-tab:hover {
  background: var(--panel-soft);
  color: var(--text);
}

.view-tab.active {
  background: var(--accent-soft);
  color: var(--accent);
}

.crumbs {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 9px 12px;
  font-size: 12px;
  color: var(--muted);
}

.crumb-trail {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px;
  min-width: 0;
}

.crumb-trail button {
  border: 0;
  background: transparent;
  padding: 2px 3px;
  color: inherit;
  border-radius: 4px;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.crumb-trail button:hover {
  color: var(--accent);
  background: var(--accent-soft);
}

.view-menu-wrap {
  position: relative;
  flex: none;
  display: flex;
  gap: 2px;
}

.crumb-action {
  width: 22px;
  height: 22px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--panel-soft);
  color: var(--text);
  font-size: 13px;
  line-height: 1;
}

.crumb-action:hover {
  border-color: var(--accent);
  background: var(--accent-soft);
  color: var(--accent);
}

.view-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 4px);
  z-index: 10;
  min-width: 132px;
  display: flex;
  flex-direction: column;
  padding: 4px;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.18);
}

.view-menu button {
  border: 0;
  background: transparent;
  color: var(--text);
  text-align: left;
  padding: 7px 9px;
  border-radius: 5px;
  font-size: 12px;
  max-width: none;
}

.view-menu button:hover {
  background: var(--accent-soft);
  color: var(--accent);
}

.view-menu button.danger {
  color: var(--danger);
}

.view-menu button.danger:hover {
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
}

.sep {
  opacity: 0.5;
}

.list-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 0 8px 12px;
}

.group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 9px 4px;
  color: var(--muted);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.group-count {
  font-weight: 600;
  letter-spacing: 0;
}

.job-row {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 8px 9px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  text-align: left;
}

.job-row:hover {
  background: var(--panel-soft);
}

.job-row.selected {
  background: var(--accent-soft);
  color: var(--accent);
  font-weight: 600;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: none;
  background: var(--muted);
}

.dot[data-tone="success"] {
  background: var(--success);
}

.dot[data-tone="danger"] {
  background: var(--danger);
}

.dot[data-tone="warning"] {
  background: #d99b1a;
}

.dot[data-tone="info"] {
  background: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chev {
  color: var(--muted);
  font-size: 16px;
  line-height: 1;
}

.last {
  color: var(--muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.empty {
  margin: 18px 8px;
  color: var(--muted);
  font-size: 12px;
  text-align: center;
}
</style>
