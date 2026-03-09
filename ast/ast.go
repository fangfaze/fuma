package ast

// Document 表示一个Fuma文档（包含多个片段）
type Document struct {
    Fragments []*Fragment
}

// Fragment 表示一个片段
type Fragment struct {
    Name  string
    Nodes []*Node
}

// Node 表示一个节点
type Node struct {
    ID    string // 可选，为空表示匿名节点
    Items []Item // 子项列表（键值对、列表项、注释）
}

// Item 是节点子项的接口
type Item interface {
    isItem()
}

// KeyValue 表示一个键值对子项
type KeyValue struct {
    Key   string
    Value Value
}

func (kv *KeyValue) isItem() {}

// ListItem 表示一个列表元素子项
type ListItem struct {
    Value Value
}

func (li *ListItem) isItem() {}

// Comment 表示注释
type Comment struct {
    Text string
}

func (c *Comment) isItem() {}

// Value 是所有值的接口
type Value interface {
    isValue()
}

// StringValue 表示普通字符串值
type StringValue struct {
    Text string
}

func (sv *StringValue) isValue() {}
