package migrations

import "testing"

func TestLoadOrdersByVersionAndParsesEveryFile(t *testing.T) {
	migs, err := load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(migs) == 0 {
		t.Fatal("load returned no migrations; expected at least the embedded *.sql files")
	}

	for i, m := range migs {
		if m.sql == "" {
			t.Errorf("migration %04d_%s has empty SQL", m.version, m.name)
		}
		if i > 0 && migs[i-1].version >= m.version {
			t.Errorf("migrations not strictly increasing: %04d_%s then %04d_%s",
				migs[i-1].version, migs[i-1].name, m.version, m.name)
		}
	}
}

func TestParseName(t *testing.T) {
	cases := []struct {
		filename    string
		wantVersion int
		wantName    string
		wantOK      bool
	}{
		{"0001_init.sql", 1, "init", true},
		{"0002_indexes.sql", 2, "indexes", true},
		{"init.sql", 0, "", false},
		{"abcd_init.sql", 0, "", false},
	}

	for _, c := range cases {
		version, name, ok := parseName(c.filename)
		if ok != c.wantOK {
			t.Errorf("parseName(%q) ok = %v, want %v", c.filename, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if version != c.wantVersion || name != c.wantName {
			t.Errorf("parseName(%q) = (%d, %q), want (%d, %q)",
				c.filename, version, name, c.wantVersion, c.wantName)
		}
	}
}
