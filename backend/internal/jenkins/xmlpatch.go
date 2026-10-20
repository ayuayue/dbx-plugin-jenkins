package jenkins

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

// xmlElement 是 XML 里一个元素的字节区间（含起止标签）。
type xmlElement struct {
	name  string
	start int
	end   int
}

// PatchDescription 替换任务/视图配置顶层的 <description>。
func PatchDescription(xmlText, description string) (string, error) {
	replacement := "<description>" + escapeXMLText(description) + "</description>"
	patched, found, err := replaceTopLevel(xmlText, "description", replacement)
	if err != nil {
		return "", err
	}
	if found {
		return patched, nil
	}
	return insertBeforeRootEnd(xmlText, replacement)
}

// PatchDisabled 替换任务配置顶层的 <disabled>。
func PatchDisabled(xmlText string, disabled bool) (string, error) {
	replacement := "<disabled>" + strconv.FormatBool(disabled) + "</disabled>"
	patched, found, err := replaceTopLevel(xmlText, "disabled", replacement)
	if err != nil {
		return "", err
	}
	if found {
		return patched, nil
	}
	return insertBeforeRootEnd(xmlText, replacement)
}

// PatchPipelineScript 替换 Pipeline <definition>/<script> 的脚本。
func PatchPipelineScript(xmlText, script string) (string, error) {
	_, elements, err := scanElements(xmlText)
	if err != nil {
		return "", err
	}

	var definition *xmlElement
	for index := range elements {
		if elements[index].name == "definition" {
			definition = &elements[index]
			break
		}
	}
	if definition == nil {
		return "", &APIError{
			Kind:    ErrInvalidConfig,
			Message: "job config has no <definition>; the job is not a Pipeline job",
		}
	}

	inner := xmlText[definition.start:definition.end]
	replacement := "<script>" + escapeXMLText(script) + "</script>"
	patchedInner, found, err := replaceTopLevel(inner, "script", replacement)
	if err != nil {
		return "", err
	}
	if !found {
		return "", &APIError{
			Kind:    ErrInvalidConfig,
			Message: "pipeline definition has no <script>; only scripted Pipeline is supported",
		}
	}

	return xmlText[:definition.start] + patchedInner + xmlText[definition.end:], nil
}

// PatchListView 改写 ListView 配置的 description/jobNames/includeRegex/recurse，
// 其余节点（columns、properties、filter 等）原样保留。
func PatchListView(xmlText string, cfg ListViewConfig) (string, error) {
	root, elements, err := scanElements(xmlText)
	if err != nil {
		return "", err
	}
	if root != listViewClass && root != "listView" {
		return "", &APIError{
			Kind:    ErrInvalidConfig,
			Message: "view is not a ListView; edit its XML directly",
		}
	}

	present := make(map[string]bool, len(elements))
	for _, element := range elements {
		present[element.name] = true
	}

	replacements := map[string]string{
		"description": "<description>" + escapeXMLText(cfg.Description) + "</description>",
		"jobNames":    jobNamesXML(cfg.JobNames),
		"recurse":     "<recurse>" + strconv.FormatBool(cfg.Recurse) + "</recurse>",
	}
	if cfg.IncludeRegex == "" {
		replacements["includeRegex"] = ""
	} else {
		replacements["includeRegex"] = "<includeRegex>" +
			escapeXMLText(cfg.IncludeRegex) + "</includeRegex>"
	}

	// 从后往前替换，避免切片偏移失效。
	for index := len(elements) - 1; index >= 0; index-- {
		element := elements[index]
		replacement, ok := replacements[element.name]
		if !ok {
			continue
		}
		xmlText = xmlText[:element.start] + replacement + xmlText[element.end:]
	}

	// 缺失的元素按固定顺序补到根结束标签之前。
	for _, name := range []string{"description", "jobNames", "includeRegex", "recurse"} {
		if present[name] || replacements[name] == "" {
			continue
		}
		xmlText, err = insertBeforeRootEnd(xmlText, replacements[name])
		if err != nil {
			return "", err
		}
	}
	return xmlText, nil
}

// scanElements 返回根元素名与所有"根的直接子元素"的字节区间。
func scanElements(xmlText string) (string, []xmlElement, error) {
	decoder := xml.NewDecoder(strings.NewReader(normalizeXMLVersion(xmlText)))
	level := 0
	root := ""
	var elements []xmlElement
	var current *xmlElement

	for {
		tokenStart := int(decoder.InputOffset())
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, invalidXMLError(err.Error())
		}

		switch typed := token.(type) {
		case xml.StartElement:
			level++
			switch level {
			case 1:
				root = typed.Name.Local
			case 2:
				start := tokenStart
				for start < len(xmlText) && isXMLSpace(xmlText[start]) {
					start++
				}
				current = &xmlElement{name: typed.Name.Local, start: start}
			}
		case xml.EndElement:
			if level == 2 && current != nil {
				current.end = int(decoder.InputOffset())
				elements = append(elements, *current)
				current = nil
			}
			level--
		}
	}

	if root == "" {
		return "", nil, invalidXMLError("empty document")
	}
	return root, elements, nil
}

// replaceTopLevel 替换第一个匹配的直接子元素；返回是否命中。
func replaceTopLevel(xmlText, name, replacement string) (string, bool, error) {
	_, elements, err := scanElements(xmlText)
	if err != nil {
		return "", false, err
	}
	for index := len(elements) - 1; index >= 0; index-- {
		element := elements[index]
		if element.name != name {
			continue
		}
		return xmlText[:element.start] + replacement + xmlText[element.end:], true, nil
	}
	return xmlText, false, nil
}

// insertBeforeRootEnd 在根结束标签前插入一段 XML。
func insertBeforeRootEnd(xmlText, fragment string) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(normalizeXMLVersion(xmlText)))
	level := 0
	for {
		tokenStart := int(decoder.InputOffset())
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", invalidXMLError(err.Error())
		}

		switch token.(type) {
		case xml.StartElement:
			level++
		case xml.EndElement:
			if level != 1 {
				level--
				continue
			}
			separator := "\n  "
			if strings.HasSuffix(xmlText[:tokenStart], "\n") {
				separator = "  "
			}
			return xmlText[:tokenStart] + separator + fragment + "\n" + xmlText[tokenStart:], nil
		}
	}
	return "", invalidXMLError("missing root element")
}

func jobNamesXML(names []string) string {
	var builder strings.Builder
	builder.WriteString("<jobNames>")
	builder.WriteString("\n    <comparator class=\"hudson.util.CaseInsensitiveComparator\"/>")
	for _, name := range names {
		builder.WriteString("\n    <string>")
		builder.WriteString(escapeXMLText(name))
		builder.WriteString("</string>")
	}
	builder.WriteString("\n  </jobNames>")
	return builder.String()
}

func isXMLSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

// normalizeXMLVersion 把声明里的 1.1 改成 1.0 供 Go 解析。
// Go 的 encoding/xml 不支持 XML 1.1；替换等长，不会影响基于原文的字节偏移。
func normalizeXMLVersion(xmlText string) string {
	if !strings.HasPrefix(xmlText, "<?xml") {
		return xmlText
	}
	end := strings.Index(xmlText, "?>")
	if end < 0 {
		return xmlText
	}
	declaration := xmlText[:end]
	declaration = strings.Replace(declaration, `version="1.1"`, `version="1.0"`, 1)
	declaration = strings.Replace(declaration, `version='1.1'`, `version='1.0'`, 1)
	return declaration + xmlText[end:]
}

func invalidXMLError(detail string) *APIError {
	return &APIError{Kind: ErrInvalidConfig, Message: "invalid XML: " + detail}
}
