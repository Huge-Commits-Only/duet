package game

import (
	"regexp"
	"testing"
)

func TestGender(t *testing.T) {
	for _, c := range []struct{ in, you, other, want string }{
		{"Co ostatnio wyszukałeś/aś?", "f", "m", "Co ostatnio wyszukałaś?"},
		{"Co ostatnio wyszukałeś/aś?", "m", "f", "Co ostatnio wyszukałeś?"},
		{"Co chciał(a)byś, żebym robił(a) częściej?", "f", "m", "Co chciałabyś, żebym robił częściej?"},
		{"Co chciał(a)byś, żebym robił(a) częściej?", "m", "f", "Co chciałbyś, żebym robiła częściej?"},
		{"Co chciał(a)byś, żebym wiedział(a), a nigdy tego nie powiedziałeś/aś?", "m", "f", "Co chciałbyś, żebym wiedziała, a nigdy tego nie powiedziałeś?"},
		{"Co powinienem/powinnam robić częściej?", "f", "m", "Co powinienem robić częściej?"},
		{"Jak bardzo jesteś uparty/a?", "f", "", "Jak bardzo jesteś uparta?"},
		{"Kiedy czujesz się najbardziej pewny/a siebie?", "m", "", "Kiedy czujesz się najbardziej pewny siebie?"},
		{"Czego potrzebujesz, żeby czuć się kochanym/ą?", "f", "", "Czego potrzebujesz, żeby czuć się kochaną?"},
		{"Co robisz, gdy jesteś sam(a) w domu?", "f", "", "Co robisz, gdy jesteś sama w domu?"},
		{"Rozdał(a)bym bliskim", "f", "m", "Rozdałabym bliskim"},
		{"Jaki film mógłbyś/mogłabyś oglądać bez końca?", "f", "", "Jaki film mogłabyś oglądać bez końca?"},
		{"O tym, jak wyobrażałeś/aś sobie partnera jako nastolatek/ka", "f", "", "O tym, jak wyobrażałaś sobie partnera jako nastolatka"},
		// a person whose form is unknown keeps the neutral spelling
		{"Co chciał(a)byś, żebym robił(a) częściej?", "", "f", "Co chciał(a)byś, żebym robiła częściej?"},
		{"Co chciał(a)byś, żebym robił(a) częściej?", "m", "", "Co chciałbyś, żebym robił(a) częściej?"},
		{"Kawa / herbata, 50/50", "m", "f", "Kawa / herbata, 50/50"},
	} {
		if got := gender(c.in, c.you, c.other); got != c.want {
			t.Errorf("gender(%q, %q, %q)\n got %q\nwant %q", c.in, c.you, c.other, got, c.want)
		}
	}

	// with both forms known, nothing in the built-in packs may keep a double spelling
	left := regexp.MustCompile(`\p{L}/\p{L}|\(a\)`)
	for mode, qs := range loadQuestions(t.TempDir()) {
		for _, q := range qs {
			for _, forms := range [][2]string{{"m", "f"}, {"f", "m"}} {
				g := q.gendered(forms[0], forms[1])
				for _, s := range append([]string{g.Q, g.A, g.B}, g.Options...) {
					if left.MatchString(s) {
						t.Errorf("%s: unresolved form in %q", mode, s)
					}
				}
			}
			if len(q.Options) > 0 && &q.gendered("m", "f").Options[0] == &q.Options[0] {
				t.Fatal("gendered must not share the pack's options slice")
			}
		}
	}

	// who a question addresses: the player it is about, else the phone's owner; the projector only when both match
	g := started(t, "znasz")
	g.players[0].Form, g.players[1].Form = "f", "m"
	if you, other := g.forms(g.pair, 1); you != "f" || other != "m" {
		t.Errorf("subject mode: everyone reads it for the subject, got %q %q", you, other)
	}
	g.subj = -1
	if you, other := g.forms(g.pair, 1); you != "m" || other != "f" {
		t.Errorf("phone: its owner, got %q %q", you, other)
	}
	if you, _ := g.forms(g.pair, -1); you != "" {
		t.Errorf("projector with different forms stays neutral, got %q", you)
	}
}

func TestPartnerka(t *testing.T) {
	for in, want := range map[string]string{
		"Pocałuj partnera tak, jak za pierwszym razem.":       "Pocałuj partnerkę tak, jak za pierwszym razem.",
		"Całuj i liż szyję partnera przez minutę.":            "Całuj i liż szyję partnerki przez minutę.",
		"Zrób partnerowi fryzurę.":                            "Zrób partnerce fryzurę.",
		"Zatańcz z partnerem wolny taniec bez muzyki.":        "Zatańcz z partnerką wolny taniec bez muzyki.",
		"Masturbacja przy partnerze?":                         "Masturbacja przy partnerce?",
		"Patrzeć, jak partner robi sobie dobrze":              "Patrzeć, jak partnerka robi sobie dobrze",
		"Kto bardziej lubi patrzeć na nagiego partnera?":      "Kto bardziej lubi patrzeć na nagą partnerkę?",
		"Zaśpiewaj refren ulubionej piosenki partnera.":       "Zaśpiewaj refren ulubionej piosenki partnerki.",
		"Naśladuj trzy zwierzęta — partner zgaduje jakie.":    "Naśladuj trzy zwierzęta — partnerka zgaduje jakie.",
		"Zrób partnerowi malinkę w miejscu, które wybierzesz": "Zrób partnerce malinkę w miejscu, które wybierzesz",
	} {
		if got := gender(in, "m", "f"); got != want {
			t.Errorf("\n got %q\nwant %q", got, want)
		}
		if got := gender(in, "f", "m"); got != in {
			t.Errorf("a male partner keeps the text: %q", got)
		}
	}
	// nothing in the packs may be left in the masculine when the partner is addressed as a woman
	left := regexp.MustCompile(`partner(a|owi|em|ze)?($|[^\p{L}])`)
	for mode, qs := range loadQuestions(t.TempDir()) {
		for _, q := range qs {
			g := q.gendered("m", "f")
			for _, s := range append([]string{g.Q, g.A, g.B}, g.Options...) {
				if left.MatchString(s) {
					t.Errorf("%s: %q", mode, s)
				}
			}
		}
	}
}
