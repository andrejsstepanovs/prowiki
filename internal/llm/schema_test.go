package llm

import "testing"

func TestParseStrict(t *testing.T) {
	type Dummy struct {
		Name string `json:"name"`
	}

	rawValid := `{"name":"test"}`
	rawFenced := "Here is the JSON:\n```json\n{\"name\":\"fenced\"}\n```\nHope it helps!"

	res1, err := ParseStrict[Dummy](rawValid)
	if err != nil || res1.Name != "test" {
		t.Fatalf("failed valid parse: %v, res: %+v", err, res1)
	}

	res2, err := ParseStrict[Dummy](rawFenced)
	if err != nil || res2.Name != "fenced" {
		t.Fatalf("failed fenced parse: %v, res: %+v", err, res2)
	}
}
