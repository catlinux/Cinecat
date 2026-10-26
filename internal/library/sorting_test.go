package library

import "testing"

func TestSortKeys(t *testing.T) {
	cases := []struct {
		title      string
		wantTitle  string
		wantLetter string
	}{
		{"El Padrino", "Padrino", "P"},
		{"la lista de Schindler", "Lista de Schindler", "L"},
		{"Los Goonies", "Goonies", "G"},
		{"Las amistades peligrosas", "Amistades peligrosas", "A"},
		{"Un monstruo viene a verme", "Monstruo viene a verme", "M"},
		{"Una noche en la ópera", "Noche en la ópera", "N"},
		{"Unos días", "Días", "D"},
		{"Unas vacaciones", "Vacaciones", "V"},
		{"The Matrix", "Matrix", "M"},
		{"A Beautiful Mind", "Beautiful Mind", "B"},
		{"An Education", "Education", "E"},
		{"L'Armata", "Armata", "A"},
		{"¡Ándale!", "Andale!", "A"},
		{"¿El Padrino?", "Padrino?", "P"},
		{"\"El Padrino\"", "Padrino\"", "P"},
		{"  (The Matrix)", "Matrix)", "M"},
		{"Ñandú", "Nandú", "N"},
		{"1917", "1917", "#"},
		{"#Hashtag", "#Hashtag", "#"},
		{"", "", "#"},
		{"   ", "", "#"},
		{"Águila", "Aguila", "A"},
		{"Élite", "Elite", "E"},
		{"Ícaro", "Icaro", "I"},
		{"Óscar", "Oscar", "O"},
		{"Último", "Ultimo", "U"},
		{"Über", "Uber", "U"},
		{"the", "The", "T"},
		{"a", "A", "A"},
	}
	for _, test := range cases {
		title, letter := sortKeys(test.title)
		if title != test.wantTitle || letter != test.wantLetter {
			t.Errorf("sortKeys(%q) = %q, %q; se esperaba %q, %q",
				test.title, title, letter, test.wantTitle, test.wantLetter)
		}
	}
}

func TestParseLetter(t *testing.T) {
	for _, test := range []struct {
		value string
		want  string
		ok    bool
	}{
		{"a", "A", true},
		{"Ñ", "N", true},
		{"ñ", "N", true},
		{"Á", "A", true},
		{"#", "#", true},
		{"", "", true},
		{"AB", "", false},
		{"1", "", false},
		{"?", "", false},
	} {
		got, err := parseLetter(test.value)
		if test.ok && (err != nil || got != test.want) {
			t.Errorf("parseLetter(%q) = %q, %v", test.value, got, err)
		}
		if !test.ok && err == nil {
			t.Errorf("parseLetter(%q) debía fallar", test.value)
		}
	}
}
