package markup

import "testing"

func TestEscapeForMarkdown(t *testing.T) {
	src := "_hello_ [world](test)!"
	want := "\\_hello\\_ \\[world\\]\\(test\\)\\!"

	if got := EscapeForMarkdown(src); got != want {
		t.Fatalf("unexpected escaped markdown: got %q want %q", got, want)
	}
}
