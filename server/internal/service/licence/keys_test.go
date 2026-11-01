package licence

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

var keyPattern = regexp.MustCompile(`^FA-[` + Alphabet + `]{4}-[` + Alphabet + `]{4}-[` + Alphabet + `]{4}-[` + Alphabet + `]{4}$`)

func TestNewKey(t *testing.T) {
	seen := map[string]bool{}
	for range 500 {
		key, err := NewKey()
		if err != nil {
			t.Fatal(err)
		}
		if !keyPattern.MatchString(key) {
			t.Fatalf("key %q does not match the expected shape", key)
		}
		if strings.ContainsAny(key[3:], "0O1I") {
			t.Fatalf("key %q has an ambiguous character", key)
		}
		if seen[key] {
			t.Fatalf("duplicate key %q", key)
		}
		seen[key] = true
		if got, err := NormaliseKey(key); err != nil || got != key {
			t.Fatalf("NormaliseKey(%q) = %q, %v; want it unchanged", key, got, err)
		}
	}
}

func TestNormaliseKey(t *testing.T) {
	const want = "FA-ABCD-EFGH-JKLM-N234"
	tests := []struct {
		name  string
		input string
		want  string
		err   error
	}{
		{"canonical", want, want, nil},
		{"lower case", "fa-abcd-efgh-jklm-n234", want, nil},
		{"spaces and no dashes", "  FA ABCD EFGH JKLM N234 ", want, nil},
		{"no prefix", "ABCDEFGHJKLMN234", want, nil},
		{"no prefix with dashes", "abcd-efgh-jklm-n234", want, nil},
		{"mixed separators", "fa-ABCD efgh-JKLM\tn234", want, nil},
		{"ambiguous zero", "FA-ABCD-EFGH-JKLM-N230", "", ErrMalformedKey},
		{"ambiguous letter O", "FA-ABCD-EFGH-JKLM-NO34", "", ErrMalformedKey},
		{"too short", "FA-ABCD-EFGH-JKLM", "", ErrMalformedKey},
		{"too long", "FA-ABCD-EFGH-JKLM-N234-X", "", ErrMalformedKey},
		{"wrong prefix", "FB-ABCD-EFGH-JKLM-N234", "", ErrMalformedKey},
		{"empty", "", "", ErrMalformedKey},
		{"punctuation", "FA-ABCD-EFGH-JKLM-N23!", "", ErrMalformedKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormaliseKey(tt.input)
			if !errors.Is(err, tt.err) || got != tt.want {
				t.Fatalf("NormaliseKey(%q) = %q, %v; want %q, %v", tt.input, got, err, tt.want, tt.err)
			}
		})
	}
}

func TestHashAndHint(t *testing.T) {
	a, _ := NormaliseKey("fa abcd efgh jklm n234")
	b, _ := NormaliseKey("FA-ABCD-EFGH-JKLM-N234")
	if HashKey(a) != HashKey(b) || len(HashKey(a)) != 64 {
		t.Fatalf("hashes of the same key differ or have the wrong length")
	}
	if got := Hint(a); got != "N234" {
		t.Fatalf("Hint() = %q, want N234", got)
	}
}

func TestInstallToken(t *testing.T) {
	token, err := NewInstallToken()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, TokenPrefix) || len(token) != len(TokenPrefix)+43 {
		t.Fatalf("token %q has the wrong shape", token)
	}
	if HashToken(token) == HashToken(token+"x") {
		t.Fatal("different tokens hashed the same")
	}
}

func TestNewUUID(t *testing.T) {
	for range 100 {
		if id := NewUUID(); !IsUUID(id) || id[14] != '4' {
			t.Fatalf("NewUUID() = %q is not a v4 uuid", id)
		}
	}
}
