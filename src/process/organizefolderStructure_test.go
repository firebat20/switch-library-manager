package process

import (
	"robpike.io/nihongo"
	"strings"
	"testing"
)

//var folderIllegalCharsRegex = regexp.MustCompile(`[./\\?%*:;=|"<>]`)

func TestRename(t *testing.T) {
	name := "Pokémon™: Let’s Go, Eevee! 포탈 나이츠"
	name = folderIllegalCharsRegex.ReplaceAllString(name, "")
	safe := cjk.FindAllString(name, -1)
	name = strings.Join(safe, "")
	name = nihongo.RomajiString(name)
}

func TestIsSplitPart(t *testing.T) {
	tests := []struct {
		fileName string
		prefix   string
		expected bool
	}{
		{"game.nsp.00", "game.nsp.", true},
		{"game.nsp.01", "game.nsp.", true},
		{"game.nsp.10", "game.nsp.", true},
		{"another_game.nsp.00", "game.nsp.", false},
		{"game.nsp.00.bak", "game.nsp.", false},
		{"00", "", true},
		{"01", "", true},
		{"readme.txt", "", false},
		{"game.nsp.00", "another.", false},
	}

	for _, tc := range tests {
		result := isSplitPart(tc.fileName, tc.prefix)
		if result != tc.expected {
			t.Errorf("isSplitPart(%q, %q) = %v; want %v", tc.fileName, tc.prefix, result, tc.expected)
		}
	}
}
