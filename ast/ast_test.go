package ast

import (
	"encoding/json"
	"reflect"
	"testing"
)

type Expectation struct {
	Path    string
	Want    interface{}
	WantErr bool
}

type TestCase struct {
	Name         string
	JSON         string
	Expectations []Expectation
}

// deepEqual 比较两个值，支持 *PlainText 与 string 的比较
func deepEqual(a, b interface{}) bool {
	// 如果 a 是 *PlainText，解引用后与 b 比较
	if pt, ok := a.(*PlainText); ok {
		a = string(*pt)
	}
	// 如果 b 是 *PlainText，解引用后与 a 比较
	if pt, ok := b.(*PlainText); ok {
		b = string(*pt)
	}
	// 如果 a 是 *Identifier，解引用后与 b 比较
	if id, ok := a.(*Identifier); ok {
		a = string(*id)
	}
	if id, ok := b.(*Identifier); ok {
		b = string(*id)
	}
	return reflect.DeepEqual(a, b)
}

func TestDocumentGetFromJSON(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "基本映射和注释",
			JSON: `{
				"type": "fuma",
				"name": "测试",
				"entries": [
					{
						"type": "map",
						"key": "用户",
						"value": [
							{"type": "map", "key": "姓名", "value": "张三", "inline": false},
							{"type": "map", "key": "年龄", "value": "25", "inline": false},
							{"type": "comment", "key": "_comments_0", "lines": ["第一行注释", "第二行注释"]}
						],
						"inline": false
					},
					{
						"type": "list",
						"key": "_index_0",
						"value": "苹果",
						"inline": false
					},
					{
						"type": "list",
						"key": "_index_1",
						"value": [
							{"type": "map", "key": "数量", "value": "3", "inline": false}
						],
						"inline": false
					},
					{
						"type": "map",
						"key": "配置",
						"value": [
							{
								"type": "map",
								"key": "路径",
								"value": [
									"/home/",
									["用户", "姓名"]
								],
								"inline": false
							}
						],
						"inline": false
					}
				]
			}`,
			Expectations: []Expectation{
				{"/0/用户/姓名", "张三", false},
				{"/0/用户/年龄", "25", false},
				{"/0/用户/_comments_0/0", "第一行注释", false},
				{"/0/用户/_comments_0/1", "第二行注释", false},
				{"/0/_index_0", "苹果", false},
				{"/0/_index_1/数量", "3", false},
				{"/0/配置/路径/0", "/home/", false},
				{"/0/配置/路径/1/0", "用户", false},
				{"/0/配置/路径/1/1", "姓名", false},
				{"/0/用户/_comments_0/2", nil, true},
				{"/0/不存在", nil, true},
				{"/0/用户/姓名/0", nil, true},
			},
		},
		{
			Name: "多片段文档",
			JSON: `[
				{"type":"fuma","name":"片段1","entries":[{"type":"map","key":"key1","value":"value1","inline":false}]},
				{"type":"fuma","name":"片段2","entries":[{"type":"map","key":"key2","value":"value2","inline":false}]}
			]`,
			Expectations: []Expectation{
				{"/0/key1", "value1", false},
				{"/1/key2", "value2", false},
				{"/2", nil, true},
			},
		},
		{
			Name: "空文档",
			JSON: `{}`,
			Expectations: []Expectation{
				{"/0", nil, true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var doc Document
			err := json.Unmarshal([]byte(tc.JSON), &doc)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			for _, exp := range tc.Expectations {
				t.Run(exp.Path, func(t *testing.T) {
					got, err := doc.Get(exp.Path)
					if (err != nil) != exp.WantErr {
						t.Errorf("Get(%q) error = %v, wantErr %v", exp.Path, err, exp.WantErr)
						return
					}
					if !exp.WantErr {
						if !deepEqual(got, exp.Want) {
							t.Errorf("Get(%q) = %v (type %T), want %v (type %T)", exp.Path, got, got, exp.Want, exp.Want)
						}
					}
				})
			}
		})
	}
}
