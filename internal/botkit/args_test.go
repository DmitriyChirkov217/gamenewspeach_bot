package botkit

import "testing"

func TestParseJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	t.Run("parses valid json", func(t *testing.T) {
		got, err := ParseJSON[payload](`{"name":"steam","id":7}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "steam" || got.ID != 7 {
			t.Fatalf("unexpected payload: %+v", got)
		}
	})

	t.Run("returns zero value on invalid json", func(t *testing.T) {
		got, err := ParseJSON[payload](`{"name":`)
		if err == nil {
			t.Fatal("expected error")
		}
		if got != (payload{}) {
			t.Fatalf("expected zero value, got %+v", got)
		}
	})
}
