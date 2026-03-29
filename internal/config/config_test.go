package config

import (
	"reflect"
	"testing"
)

func TestConfigDefaultModelTag(t *testing.T) {
	field, ok := reflect.TypeOf(Config{}).FieldByName("OpenAIModel")
	if !ok {
		t.Fatal("OpenAIModel field not found")
	}
	if got := field.Tag.Get("default"); got != "gpt-4o-mini" {
		t.Fatalf("unexpected default tag: %q", got)
	}
}
