package parser

import "testing"

func TestParse(t *testing.T) {
    _, err := Parse(`【fuma】： test`)
    if err != nil {
        t.Errorf("Parse failed: %v", err)
    }
}
