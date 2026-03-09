package ast

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Document 文档，包含零个或多个片段
type Document struct {
	Fragments []*Fragment `json:"fragments,omitempty"`
}

// Fragment 片段
type Fragment struct {
	Type    string    `json:"type"`              // "fuma" 或 "summary"
	Name    string    `json:"name,omitempty"`    // 仅当 Type 为 "fuma" 时有意义
	Entries EntryList `json:"entries,omitempty"` // 条目列表
}

// Entry 是条目的接口
type Entry interface {
	isEntry()
	json.Marshaler
	json.Unmarshaler
}

// MapEntry 映射条目
type MapEntry struct {
	Type   string      `json:"type"`   // 固定为 "map"
	Key    interface{} `json:"key"`    // 可以是 PlainText, *TextTemplate, *SystemCall
	Value  interface{} `json:"value"`  // 可以是 NullValue, PlainText, *TextTemplate, EntryList
	Inline bool        `json:"inline"` // 是否内联
}

func (*MapEntry) isEntry() {}

// MarshalJSON 实现 json.Marshaler
func (m *MapEntry) MarshalJSON() ([]byte, error) {
	type aux MapEntry
	return json.Marshal((*aux)(m))
}

// UnmarshalJSON 实现 json.Unmarshaler
func (m *MapEntry) UnmarshalJSON(data []byte) error {
	type rawMapEntry struct {
		Type   string          `json:"type"`
		Key    json.RawMessage `json:"key"`
		Value  json.RawMessage `json:"value"`
		Inline bool            `json:"inline"`
	}
	var raw rawMapEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Type != "map" {
		return fmt.Errorf("expected type map, got %s", raw.Type)
	}
	m.Type = "map"
	m.Inline = raw.Inline

	key, err := decodeValue(raw.Key)
	if err != nil {
		return err
	}
	m.Key = key

	val, err := decodeValue(raw.Value)
	if err != nil {
		return err
	}
	m.Value = val
	return nil
}

// ListEntry 列表条目
type ListEntry struct {
	Type   string      `json:"type"`   // 固定为 "list"
	Key    string      `json:"key"`    // 自动生成的键，如 "_index_0"
	Value  interface{} `json:"value"`  // 同 MapEntry 的值类型
	Inline bool        `json:"inline"` // 是否内联
}

func (*ListEntry) isEntry() {}

// MarshalJSON 实现 json.Marshaler
func (l *ListEntry) MarshalJSON() ([]byte, error) {
	type aux ListEntry
	return json.Marshal((*aux)(l))
}

// UnmarshalJSON 实现 json.Unmarshaler
func (l *ListEntry) UnmarshalJSON(data []byte) error {
	type rawListEntry struct {
		Type   string          `json:"type"`
		Key    string          `json:"key"`
		Value  json.RawMessage `json:"value"`
		Inline bool            `json:"inline"`
	}
	var raw rawListEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Type != "list" {
		return fmt.Errorf("expected type list, got %s", raw.Type)
	}
	l.Type = "list"
	l.Key = raw.Key
	l.Inline = raw.Inline

	val, err := decodeValue(raw.Value)
	if err != nil {
		return err
	}
	l.Value = val
	return nil
}

// CommentBlock 注释块
type CommentBlock struct {
	Type  string   `json:"type"`  // 固定为 "comment"
	Key   string   `json:"key"`   // 自动生成的键，如 "_comments_0"
	Lines []string `json:"lines"` // 注释行内容
}

func (*CommentBlock) isEntry() {}

// MarshalJSON 实现 json.Marshaler
func (c *CommentBlock) MarshalJSON() ([]byte, error) {
	type aux CommentBlock
	return json.Marshal((*aux)(c))
}

// UnmarshalJSON 实现 json.Unmarshaler
func (c *CommentBlock) UnmarshalJSON(data []byte) error {
	type aux CommentBlock
	return json.Unmarshal(data, (*aux)(c))
}

// EntryList 条目列表，用于嵌套
type EntryList []Entry

// NullValue 表示空白值
type NullValue struct{}

func (NullValue) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}
func (NullValue) isTemplateElement() {}

// PlainText 普通文本
type PlainText string

func (p PlainText) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(p))
}
func (p *PlainText) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*p = PlainText(s)
	return nil
}
func (PlainText) isTemplateElement() {}

// TextTemplate 文本模板，包含多个元素
type TextTemplate struct {
	Elements []TemplateElement `json:"elements"`
}

// TemplateElement 是模板元素的接口
type TemplateElement interface {
	isTemplateElement()
	json.Marshaler
	json.Unmarshaler
}

// VariableReference 变量引用
type VariableReference struct {
	Segments []Segment `json:"segments"`
}

func (*VariableReference) isTemplateElement() {}
func (*VariableReference) isSegment()         {}

// Segment 是变量引用中段的接口
type Segment interface {
	isSegment()
	json.Marshaler
	json.Unmarshaler
}

type Identifier string

func (i Identifier) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(i))
}
func (i *Identifier) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*i = Identifier(s)
	return nil
}
func (Identifier) isSegment() {}

// SystemCall 系统调用，只能作为键
type SystemCall struct {
	Name interface{} `json:"name"` // 可以是 PlainText 或 *TextTemplate
}

func (*SystemCall) isTemplateElement() {}

// ==================== JSON 序列化辅助 ====================

func (f *Fragment) MarshalJSON() ([]byte, error) {
	type aux Fragment
	return json.Marshal((*aux)(f))
}

func (f *Fragment) UnmarshalJSON(data []byte) error {
	type aux Fragment
	return json.Unmarshal(data, (*aux)(f))
}

func (d *Document) MarshalJSON() ([]byte, error) {
	if len(d.Fragments) == 0 {
		return []byte("{}"), nil
	}
	if len(d.Fragments) == 1 {
		return json.Marshal(d.Fragments[0])
	}
	return json.Marshal(d.Fragments)
}

func (d *Document) UnmarshalJSON(data []byte) error {
	if string(data) == "{}" {
		d.Fragments = nil
		return nil
	}
	// 尝试解析为数组
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err == nil {
		frags := make([]*Fragment, len(arr))
		for i, raw := range arr {
			var f Fragment
			if err := json.Unmarshal(raw, &f); err != nil {
				return err
			}
			frags[i] = &f
		}
		d.Fragments = frags
		return nil
	}
	// 单片段
	var f Fragment
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	d.Fragments = []*Fragment{&f}
	return nil
}

// decodeValue 将 JSON raw 解码为 interface{}，正确处理各种类型
func decodeValue(raw json.RawMessage) (interface{}, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		ps := PlainText(s)
		return &ps, nil
	}
	// 尝试解析为数组
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		if len(arr) == 0 {
			return EntryList{}, nil
		}
		// 尝试作为 EntryList
		var el EntryList
		errList := json.Unmarshal(raw, &el)
		if errList == nil {
			return el, nil
		}
		// 尝试作为 TextTemplate
		var tt TextTemplate
		errTmpl := json.Unmarshal(raw, &tt)
		if errTmpl == nil {
			return &tt, nil
		}
		return nil, fmt.Errorf("cannot decode array (as EntryList: %v, as TextTemplate: %v)", errList, errTmpl)
	}
	// 尝试解析为对象
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		// 如果有 "type" 字段，可能是条目
		if typeVal, ok := obj["type"]; ok {
			switch typeVal {
			case "map":
				var me MapEntry
				if err := json.Unmarshal(raw, &me); err == nil {
					return &me, nil
				}
			case "list":
				var le ListEntry
				if err := json.Unmarshal(raw, &le); err == nil {
					return &le, nil
				}
			case "comment":
				var cb CommentBlock
				if err := json.Unmarshal(raw, &cb); err == nil {
					return &cb, nil
				}
			}
		}
		// 否则可能是 TextTemplate 或 VariableReference
		var tt TextTemplate
		if err := json.Unmarshal(raw, &tt); err == nil {
			return &tt, nil
		}
		var vr VariableReference
		if err := json.Unmarshal(raw, &vr); err == nil {
			return &vr, nil
		}
		return nil, fmt.Errorf("cannot decode object")
	}
	return nil, fmt.Errorf("cannot decode value")
}

// EntryList 的序列化
func (el EntryList) MarshalJSON() ([]byte, error) {
	return json.Marshal([]Entry(el))
}

func (el *EntryList) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	entries := make([]Entry, len(raw))
	for i, r := range raw {
		// 先解析出一个 map 获取 type 字段
		var obj map[string]interface{}
		if err := json.Unmarshal(r, &obj); err != nil {
			return err
		}
		typeVal, ok := obj["type"]
		if !ok {
			return fmt.Errorf("missing type field in entry")
		}
		typeStr, ok := typeVal.(string)
		if !ok {
			return fmt.Errorf("type field is not string")
		}
		switch typeStr {
		case "map":
			var me MapEntry
			if err := json.Unmarshal(r, &me); err != nil {
				return err
			}
			entries[i] = &me
		case "list":
			var le ListEntry
			if err := json.Unmarshal(r, &le); err != nil {
				return err
			}
			entries[i] = &le
		case "comment":
			var cb CommentBlock
			if err := json.Unmarshal(r, &cb); err != nil {
				return err
			}
			entries[i] = &cb
		default:
			return fmt.Errorf("unknown entry type: %s", typeStr)
		}
	}
	*el = entries
	return nil
}

// TextTemplate 的序列化
func (tt *TextTemplate) MarshalJSON() ([]byte, error) {
	return json.Marshal(tt.Elements)
}

func (tt *TextTemplate) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	elements := make([]TemplateElement, len(raw))
	for i, r := range raw {
		// 判断是 PlainText 还是 VariableReference
		var s string
		if err := json.Unmarshal(r, &s); err == nil {
			ps := PlainText(s)
			elements[i] = &ps
			continue
		}
		var vr VariableReference
		if err := json.Unmarshal(r, &vr); err == nil {
			elements[i] = &vr
			continue
		}
		return fmt.Errorf("unknown template element")
	}
	tt.Elements = elements
	return nil
}

// VariableReference 的序列化
func (vr *VariableReference) MarshalJSON() ([]byte, error) {
	return json.Marshal(vr.Segments)
}

func (vr *VariableReference) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	segments := make([]Segment, len(raw))
	for i, r := range raw {
		var s string
		if err := json.Unmarshal(r, &s); err == nil {
			id := Identifier(s)
			segments[i] = &id
			continue
		}
		var vr2 VariableReference
		if err := json.Unmarshal(r, &vr2); err == nil {
			segments[i] = &vr2
			continue
		}
		return fmt.Errorf("unknown segment")
	}
	vr.Segments = segments
	return nil
}

// SystemCall 的序列化
func (sc *SystemCall) MarshalJSON() ([]byte, error) {
	type aux SystemCall
	return json.Marshal((*aux)(sc))
}

func (sc *SystemCall) UnmarshalJSON(data []byte) error {
	type aux SystemCall
	return json.Unmarshal(data, (*aux)(sc))
}

// ==================== 辅助函数 ====================

func getEntryKey(e Entry) string {
	switch entry := e.(type) {
	case *MapEntry:
		if key, ok := entry.Key.(*PlainText); ok {
			return string(*key)
		}
	case *ListEntry:
		return entry.Key
	case *CommentBlock:
		return entry.Key
	}
	return ""
}

func isLeafValue(v interface{}) bool {
	switch v.(type) {
	case *PlainText, NullValue:
		return true
	default:
		return false
	}
}

// isEntryLeaf 判断条目是否叶子
func isEntryLeaf(e Entry) bool {
	switch entry := e.(type) {
	case *MapEntry:
		return isLeafValue(entry.Value)
	case *ListEntry:
		return isLeafValue(entry.Value)
	case *CommentBlock:
		return false
	default:
		return false
	}
}

// leafValue 获取条目的叶子值
func leafValue(e Entry) interface{} {
	switch entry := e.(type) {
	case *MapEntry:
		if pt, ok := entry.Value.(*PlainText); ok {
			return string(*pt)
		}
		return nil
	case *ListEntry:
		if pt, ok := entry.Value.(*PlainText); ok {
			return string(*pt)
		}
		return nil
	default:
		return nil
	}
}

// ==================== 路径查询 ====================

func (d *Document) Get(path string) (interface{}, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, errors.New("path must start with /")
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 {
		return d, nil
	}
	idx, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, errors.New("first segment must be a number")
	}
	if idx < 0 || idx >= len(d.Fragments) {
		return nil, errors.New("fragment index out of range")
	}
	current := interface{}(d.Fragments[idx])

	for i := 1; i < len(parts); i++ {
		part := parts[i]

		// 展开 current：如果它是 MapEntry 或 ListEntry，则进入其值，直到遇到非条目或叶子
		for {
			switch v := current.(type) {
			case *MapEntry:
				current = v.Value
				if isLeafValue(current) {
					break
				}
				continue
			case *ListEntry:
				current = v.Value
				if isLeafValue(current) {
					break
				}
				continue
			default:
				break
			}
			break
		}

		// 检查叶子值后是否有剩余段
		if leaf, ok := current.(*PlainText); ok {
			if i < len(parts)-1 {
				return nil, errors.New("cannot go deeper from plain text")
			}
			return string(*leaf), nil
		}
		if leaf, ok := current.(*Identifier); ok {
			if i < len(parts)-1 {
				return nil, errors.New("cannot go deeper from identifier")
			}
			return string(*leaf), nil
		}
		if leaf, ok := current.(string); ok {
			if i < len(parts)-1 {
				return nil, errors.New("cannot go deeper from string")
			}
			return leaf, nil
		}

		// 处理其他类型
		switch node := current.(type) {
		case *Fragment:
			found := false
			for _, entry := range node.Entries {
				if getEntryKey(entry) == part {
					// 找到条目，先检查是否是叶子
					if isEntryLeaf(entry) {
						if i+1 < len(parts) {
							return nil, errors.New("cannot go deeper from leaf value")
						}
						return leafValue(entry), nil
					}
					current = entry
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("entry not found: %s", part)
			}
		case EntryList:
			found := false
			for _, entry := range node {
				if getEntryKey(entry) == part {
					if isEntryLeaf(entry) {
						if i+1 < len(parts) {
							return nil, errors.New("cannot go deeper from leaf value")
						}
						return leafValue(entry), nil
					}
					current = entry
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("entry not found in list: %s", part)
			}
		case *CommentBlock:
			lineIdx, err := strconv.Atoi(part)
			if err != nil {
				return nil, errors.New("comment block segment must be a number")
			}
			if lineIdx < 0 || lineIdx >= len(node.Lines) {
				return nil, errors.New("comment line index out of range")
			}
			return node.Lines[lineIdx], nil
		case *TextTemplate:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, errors.New("text template segment must be a number")
			}
			if idx < 0 || idx >= len(node.Elements) {
				return nil, errors.New("text template element index out of range")
			}
			current = node.Elements[idx]
		case *VariableReference:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, errors.New("variable reference segment must be a number")
			}
			if idx < 0 || idx >= len(node.Segments) {
				return nil, errors.New("variable reference segment index out of range")
			}
			current = node.Segments[idx]
		default:
			return nil, fmt.Errorf("unsupported node type: %T", node)
		}
	}

	// 循环结束，处理最终节点
	switch v := current.(type) {
	case *MapEntry:
		if isLeafValue(v.Value) {
			if pt, ok := v.Value.(*PlainText); ok {
				return string(*pt), nil
			}
			return nil, nil // NullValue
		}
		return v, nil
	case *ListEntry:
		if isLeafValue(v.Value) {
			if pt, ok := v.Value.(*PlainText); ok {
				return string(*pt), nil
			}
			return nil, nil
		}
		return v, nil
	case *PlainText:
		return string(*v), nil
	case *Identifier:
		return string(*v), nil
	case string:
		return v, nil
	default:
		return current, nil
	}
}
