package helpers

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Purrquinox Digital":      "purrquinox-digital",
		"The Copper Spoon!":       "the-copper-spoon",
		"  Multiple   Spaces  ":   "multiple-spaces",
		"Café & Bar":              "caf-bar",
		"TaskFlow":                "taskflow",
		"123 Main St.":            "123-main-st",
		"--already--slugged--":    "already-slugged",
		"!!!":                     "",
		"":                        "",
		"Ben & Jerry's Ice Cream": "ben-jerry-s-ice-cream",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
