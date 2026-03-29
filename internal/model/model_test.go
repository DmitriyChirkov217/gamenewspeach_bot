package model

import "testing"

func TestArticleTagFields(t *testing.T) {
	tag := ArticleTag{Tag: "rpg", Weight: 0.9}
	if tag.Tag != "rpg" || tag.Weight != 0.9 {
		t.Fatalf("unexpected article tag: %+v", tag)
	}
}
