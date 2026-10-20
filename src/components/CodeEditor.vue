<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { t } from "../i18n";
import { gutterText, indentSelection } from "../lib/editor";
import { highlight, type CodeLang } from "../lib/highlight";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    lang?: CodeLang;
    rows?: number;
  }>(),
  { lang: "groovy", rows: 10 },
);

const emit = defineEmits<{ "update:modelValue": [value: string] }>();

/** 必须是像素值，textarea 与高亮层用同一行高才能逐行对齐。 */
const LINE_HEIGHT = 18.6;
const PADDING_TOP = 10;

const textarea = ref<HTMLTextAreaElement | null>(null);
const highlightBox = ref<HTMLElement | null>(null);
const gutterBox = ref<HTMLElement | null>(null);
const expanded = ref(false);
const scrollTop = ref(0);
const currentLine = ref(1);

// 末尾补一个换行，保证以 \n 结尾时高亮层和 textarea 高度一致。
const html = computed(() => `${highlight(props.modelValue, props.lang)}\n`);
const gutter = computed(() => gutterText(props.modelValue));
const lineCount = computed(() => props.modelValue.split("\n").length);
const digits = computed(() => String(Math.max(2, String(lineCount.value).length)));

const editorStyle = computed<Record<string, string>>(() => ({
  "--rows": String(props.rows),
  "--digits": digits.value,
  "--active-top": `${PADDING_TOP + (currentLine.value - 1) * LINE_HEIGHT - scrollTop.value}px`,
}));

function syncScroll(): void {
  const el = textarea.value;
  if (!el) return;
  scrollTop.value = el.scrollTop;
  if (highlightBox.value) {
    highlightBox.value.scrollTop = el.scrollTop;
    highlightBox.value.scrollLeft = el.scrollLeft;
  }
  if (gutterBox.value) gutterBox.value.scrollTop = el.scrollTop;
}

function syncCaret(): void {
  const el = textarea.value;
  if (!el) return;
  currentLine.value = el.value.slice(0, el.selectionStart).split("\n").length;
}

function onInput(event: Event): void {
  const el = event.target as HTMLTextAreaElement;
  emit("update:modelValue", el.value);
  syncCaret();
  syncScroll();
}

function onKeydown(event: KeyboardEvent): void {
  if (event.key !== "Tab" || event.ctrlKey || event.altKey || event.metaKey) return;
  const el = event.target as HTMLTextAreaElement;
  event.preventDefault();
  const result = indentSelection(
    el.value,
    el.selectionStart,
    el.selectionEnd,
    event.shiftKey,
  );
  if (result.value === el.value) return;
  emit("update:modelValue", result.value);
  nextTick(() => {
    el.selectionStart = result.selectionStart;
    el.selectionEnd = result.selectionEnd;
    syncCaret();
    syncScroll();
  });
}

function toggle(): void {
  const el = textarea.value;
  const caret = el ? { start: el.selectionStart, end: el.selectionEnd } : null;
  expanded.value = !expanded.value;
  nextTick(() => {
    if (!el) return;
    el.focus({ preventScroll: true });
    if (caret) {
      el.selectionStart = caret.start;
      el.selectionEnd = caret.end;
    }
    syncScroll();
    syncCaret();
  });
}

/** 展开时 Esc 只收起编辑层，不关闭外层的任务弹窗。 */
function onWindowKeydown(event: KeyboardEvent): void {
  if (event.key !== "Escape" || !expanded.value) return;
  event.preventDefault();
  event.stopPropagation();
  expanded.value = false;
}

watch(expanded, () => {
  nextTick(() => {
    syncScroll();
    textarea.value?.focus();
  });
});

onMounted(() => {
  window.addEventListener("keydown", onWindowKeydown, true);
  // 未聚焦的 textarea 在绑值后光标默认停在末尾，先归零，避免「当前行」一开始就落在最后一行。
  const el = textarea.value;
  if (el) {
    el.selectionStart = 0;
    el.selectionEnd = 0;
  }
  currentLine.value = 1;
});

onBeforeUnmount(() => window.removeEventListener("keydown", onWindowKeydown, true));
</script>

<template>
  <div class="code-editor" :class="{ expanded }" :style="editorStyle">
    <div class="bar">
      <span class="lang">{{ lang === "xml" ? "XML" : "GROOVY" }}</span>
      <span class="count">{{ t("editorLineCount", { count: lineCount }) }}</span>
      <button type="button" class="expand" @click="toggle">
        {{ expanded ? t("editorCollapse") : t("editorExpand") }}
      </button>
    </div>
    <div class="body">
      <pre ref="gutterBox" class="gutter" aria-hidden="true">{{ gutter }}</pre>
      <div class="surface">
        <div class="active-line" aria-hidden="true"></div>
        <pre ref="highlightBox" class="highlight" aria-hidden="true"><code v-html="html"></code></pre>
        <textarea
          ref="textarea"
          :value="modelValue"
          spellcheck="false"
          wrap="off"
          autocomplete="off"
          autocapitalize="off"
          :aria-label="lang === 'xml' ? 'XML' : 'Groovy'"
          @input="onInput"
          @scroll="syncScroll"
          @keydown="onKeydown"
          @click="syncCaret"
          @keyup="syncCaret"
          @select="syncCaret"
        ></textarea>
      </div>
    </div>
  </div>
</template>

<style scoped>
.code-editor {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel-soft);
  overflow: hidden;
}

.code-editor:focus-within {
  border-color: var(--accent);
}

.code-editor.expanded {
  position: fixed;
  inset: 0;
  z-index: 60;
  border: 0;
  border-radius: 0;
  background: var(--panel);
}

.bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 6px 4px 10px;
  border-bottom: 1px solid var(--border);
  font-size: 10px;
  letter-spacing: 0.04em;
  color: var(--muted);
}

.expanded .bar {
  padding: 10px 14px;
  font-size: 11px;
}

.lang {
  font-family: var(--mono);
  font-weight: 700;
}

.count {
  margin-right: auto;
}

.expand {
  border: 1px solid var(--border);
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  padding: 2px 8px;
  font-size: 10px;
  line-height: 1.5;
}

.expand:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.body {
  display: flex;
  min-height: 0;
  height: calc(var(--rows) * 18.6px + 20px);
}

.expanded .body {
  flex: 1;
  height: auto;
}

.gutter {
  flex: none;
  width: calc(var(--digits) * 1ch + 18px);
  margin: 0;
  padding: 10px 6px 10px 0;
  border-right: 1px solid var(--border);
  background: color-mix(in srgb, var(--text) 3%, transparent);
  color: var(--muted);
  font-family: var(--mono);
  font-size: 12px;
  line-height: 18.6px;
  text-align: right;
  white-space: pre;
  overflow: hidden;
  user-select: none;
}

.surface {
  position: relative;
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.active-line {
  position: absolute;
  left: 0;
  right: 0;
  top: var(--active-top);
  height: 18.6px;
  background: var(--accent-soft);
  pointer-events: none;
}

.highlight,
textarea {
  position: absolute;
  inset: 0;
  margin: 0;
  border: 0;
  padding: 10px 12px;
  font-family: var(--mono);
  font-size: 12px;
  line-height: 18.6px;
  letter-spacing: normal;
  tab-size: 2;
  white-space: pre;
  overflow: auto;
  word-break: normal;
  overflow-wrap: normal;
}

.highlight {
  pointer-events: none;
  color: var(--text);
  scrollbar-width: none;
}

.highlight::-webkit-scrollbar {
  display: none;
}

.highlight code {
  font: inherit;
}

textarea {
  resize: none;
  outline: none;
  background: transparent;
  color: transparent;
  caret-color: var(--text);
}

textarea::selection {
  background: color-mix(in srgb, var(--accent) 32%, transparent);
  color: transparent;
}

textarea::-webkit-scrollbar {
  width: 10px;
  height: 10px;
}

textarea::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: color-mix(in srgb, var(--text) 24%, transparent);
  background-clip: content-box;
}

textarea::-webkit-scrollbar-thumb:hover {
  background: color-mix(in srgb, var(--text) 42%, transparent);
  background-clip: content-box;
}

textarea::-webkit-scrollbar-track,
textarea::-webkit-scrollbar-corner {
  background: transparent;
}

.highlight :deep(.tok-comment) {
  color: var(--muted);
  font-style: italic;
}

.highlight :deep(.tok-keyword) {
  color: #a626a4;
  font-weight: 600;
}

.highlight :deep(.tok-builtin) {
  color: var(--accent);
}

.highlight :deep(.tok-string) {
  color: #1a9e6d;
}

.highlight :deep(.tok-number) {
  color: #b76b01;
}

.highlight :deep(.tok-var) {
  color: #986801;
  font-weight: 600;
}

.highlight :deep(.tok-attr) {
  color: #b76b01;
}

.highlight :deep(.tok-tag) {
  color: var(--accent);
}

.highlight :deep(.tok-meta) {
  color: var(--muted);
}

:root[data-theme="dark"] .highlight :deep(.tok-keyword) {
  color: #c678dd;
}

:root[data-theme="dark"] .highlight :deep(.tok-string) {
  color: #98c379;
}

:root[data-theme="dark"] .highlight :deep(.tok-number),
:root[data-theme="dark"] .highlight :deep(.tok-attr) {
  color: #d19a66;
}

:root[data-theme="dark"] .highlight :deep(.tok-var) {
  color: #e5c07b;
}
</style>
