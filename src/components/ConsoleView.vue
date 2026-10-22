<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { describeError, jenkinsApi } from "../api";
import { locale, t } from "../i18n";
import { countMatches, highlightSegments, tailLines, trimLogTail } from "../lib/console";
import { errorText } from "../lib/errors";

const props = defineProps<{
  job: string;
  number: number;
}>();

const MAX_LINES = 5000;
/** 日志文本内存上限：约 2MB，超出只留尾部（显示本来就只截 5000 行）。 */
const MAX_TEXT_CHARS = 2_000_000;
/** 增量拉取间隔。 */
const POLL_DELAY_MS = 1500;

const text = ref("");
const nextStart = ref(0);
const moreData = ref(false);
const error = ref("");
const follow = ref(true);
const query = ref("");
const copied = ref(false);
const logEl = ref<HTMLElement | null>(null);

let timer: number | undefined;
let generation = 0;
/** 正在飞行中的轮次所属 generation，-1 表示空闲；按代际记录，reset 后新轮不受旧请求阻挡。 */
let inFlight = -1;

const lines = computed(() => tailLines(text.value, MAX_LINES));
const totalLines = computed(() => {
  if (!text.value) return 0;
  const parts = text.value.split(/\r?\n/);
  return parts[parts.length - 1] === "" ? parts.length - 1 : parts.length;
});
const truncated = computed(() => totalLines.value > MAX_LINES);
const matches = computed(() => countMatches(lines.value, query.value));

function scrollToBottom(): void {
  const element = logEl.value;
  if (element) element.scrollTop = element.scrollHeight;
}

async function poll(current: number): Promise<void> {
  if (inFlight === current) return;
  inFlight = current;
  try {
    const chunk = await jenkinsApi.console(props.job, props.number, nextStart.value);
    if (current !== generation) return;
    if (chunk.text) {
      text.value = trimLogTail(text.value + chunk.text, MAX_TEXT_CHARS);
    }
    nextStart.value = chunk.nextStart;
    moreData.value = chunk.moreData;
    error.value = "";
    await nextTick();
    if (follow.value) scrollToBottom();
    // 后台不续排：回到前台由 visibilitychange 接着拉，避免后台空转。
    if (chunk.moreData && !document.hidden) {
      timer = window.setTimeout(() => void poll(current), POLL_DELAY_MS);
    }
  } catch (caught) {
    if (current !== generation) return;
    error.value = errorText(describeError(caught), locale.value);
  } finally {
    if (inFlight === current) inFlight = -1;
  }
}

/** 页面隐藏时取消已排的下一轮；回到前台且还有增量就立刻续上。 */
function handleVisibility(): void {
  if (document.hidden) {
    if (timer !== undefined) {
      window.clearTimeout(timer);
      timer = undefined;
    }
    return;
  }
  if (moreData.value && inFlight !== generation) void poll(generation);
}

function reset(): void {
  generation += 1;
  if (timer) window.clearTimeout(timer);
  text.value = "";
  nextStart.value = 0;
  moreData.value = false;
  error.value = "";
  void poll(generation);
}

async function copyLog(): Promise<void> {
  const content = text.value;
  try {
    if (window.dbxPlugin?.copy) {
      await window.dbxPlugin.copy(content);
    } else {
      await navigator.clipboard.writeText(content);
    }
    copied.value = true;
    window.setTimeout(() => (copied.value = false), 1500);
  } catch {
    copied.value = false;
  }
}

function downloadLog(): void {
  try {
    const blob = new Blob([text.value], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${props.job.replaceAll("/", "-")}-${props.number}.log`;
    anchor.click();
    URL.revokeObjectURL(url);
  } catch {
    void copyLog();
  }
}

watch(() => [props.job, props.number], reset, { immediate: true });

onMounted(() => document.addEventListener("visibilitychange", handleVisibility));

onBeforeUnmount(() => {
  document.removeEventListener("visibilitychange", handleVisibility);
  generation += 1;
  if (timer) window.clearTimeout(timer);
});
</script>

<template>
  <section class="console">
    <div class="console-bar">
      <strong>{{ t("console") }} · #{{ number }}</strong>
      <label class="toggle">
        <input v-model="follow" type="checkbox" />
        <span>{{ t("follow") }}</span>
      </label>
      <input v-model="query" type="search" class="search" :placeholder="t('searchLog')" />
      <span v-if="query" class="matches">{{ t("matchCount", { count: matches }) }}</span>
      <button type="button" class="ghost" @click="copyLog">
        {{ copied ? t("copied") : t("copy") }}
      </button>
      <button type="button" class="ghost" @click="downloadLog">{{ t("download") }}</button>
    </div>

    <p v-if="error" class="banner">{{ error }}</p>
    <p v-if="truncated" class="hint">{{ t("logTruncated") }}</p>

    <pre v-if="lines.length" ref="logEl" class="log"><template
      v-for="(line, index) in lines"
      :key="index"
    ><span
      v-for="(segment, segmentIndex) in highlightSegments(line, query)"
      :key="segmentIndex"
      :class="{ hit: segment.match }"
    >{{ segment.text }}</span>{{ "\n" }}</template></pre>
    <p v-else class="empty">{{ error ? t("consoleLoadFailed") : t("logEmpty") }}</p>

    <p v-if="moreData" class="live">●</p>
  </section>
</template>

<style scoped>
.console {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 320px;
}

.console-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
}

.console-bar strong {
  font-size: 13px;
  margin-right: auto;
}

.toggle {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: 12px;
}

.search {
  width: 180px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 8px;
  background: var(--panel-soft);
  color: var(--text);
  font: inherit;
  font-size: 12px;
  outline: none;
}

.search:focus {
  border-color: var(--accent);
}

.matches {
  color: var(--muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.ghost {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 5px 9px;
  background: transparent;
  color: var(--text);
  font-size: 12px;
}

.ghost:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.banner {
  margin: 0;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
  font-size: 12px;
}

.hint {
  margin: 0;
  padding: 6px 12px;
  color: var(--muted);
  font-size: 11px;
  border-bottom: 1px solid var(--border);
}

.log {
  margin: 0;
  padding: 12px;
  flex: 1;
  min-height: 260px;
  max-height: 56vh;
  overflow: auto;
  background: #101319;
  color: #d7dce5;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.log .hit {
  background: #ffd54f;
  color: #101319;
  border-radius: 2px;
}

.empty {
  margin: 0;
  padding: 28px 12px;
  color: var(--muted);
  font-size: 12px;
  text-align: center;
}

.live {
  margin: 0;
  padding: 4px 12px;
  color: var(--success);
  font-size: 10px;
  border-top: 1px solid var(--border);
}
</style>
