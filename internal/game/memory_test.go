package game

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// at moves the game's clock to a fixed moment for the rest of the test.
func at(t *testing.T, when time.Time) {
	old := clock
	clock = func() time.Time { return when }
	t.Cleanup(func() { clock = old })
}

// playThrough answers every round with what answer returns for it, to the end of the game.
func playThrough(g *Game, answer func(round int) (a, b string)) {
	for g.phase == "question" {
		a, b := answer(g.round)
		g.Act("a", "answer", "", a)
		g.Act("b", "answer", "", b)
		all(g, "next")
	}
}

func same(int) (string, string) { return "0", "0" }

var march = time.Date(2026, 3, 10, 20, 0, 0, 0, time.Local)

func TestQuestionComesBack(t *testing.T) {
	at(t, march)
	g := started(t, "wolisz")
	playThrough(g, same)
	all(g, "quit")

	// a month later it is too early for a question to come back
	at(t, march.AddDate(0, 1, 0))
	g.Act("a", "start", "", "")
	for _, c := range g.deck {
		if c.before != nil {
			t.Fatal("a question came back after a month")
		}
	}
	all(g, "quit")

	at(t, march.AddDate(0, 2, 1))
	g.Act("a", "start", "", "")
	again := -1
	for i, c := range g.deck {
		if c.before != nil {
			if again >= 0 {
				t.Fatal("more than one question came back in one game")
			}
			again = i
		}
	}
	if again < 0 {
		t.Fatal("after two months one question must come back")
	}
	for g.round < again+1 {
		g.Act("a", "answer", "", "0")
		g.Act("b", "answer", "", "0")
		all(g, "next")
	}
	var v struct{ Before *viewBefore }
	json.Unmarshal(g.view("b"), &v)
	if v.Before == nil || v.Before.When != "w marcu" || v.Before.Said != [2]string{} {
		t.Fatalf("question view: %+v, want when without the old answers", v.Before)
	}
	g.Act("a", "answer", "", "1") // Natalia changed her mind
	g.Act("b", "answer", "", "0")
	json.Unmarshal(g.view(""), &v)
	was := g.deck[again].before.Said[0]
	if v.Before == nil || v.Before.Said[0] != was || v.Before.Said[1] != was {
		t.Fatalf("reveal: %+v, want what both said in March", v.Before)
	}
	all(g, "next")
	playThrough(g, same)

	p := g.store.profile(clock())
	if len(p.Changed) != 1 || p.Changed[0].Who != "Natalia" || p.Changed[0].From != was || p.Changed[0].When != "w marcu" {
		t.Fatalf("changed = %+v, want Natalia's new answer", p.Changed)
	}
}

func TestAnniversary(t *testing.T) {
	at(t, march)
	g := started(t, "skala")
	all(g, "quit")
	if err := g.Act("a", "config", "mode", "rocznica"); err != nil {
		t.Fatal(err)
	}
	if err := g.Act("a", "start", "", ""); err != errHistory {
		t.Fatalf("an anniversary without history: %v", err)
	}
	g.Act("a", "config", "mode", "skala")
	g.Act("a", "start", "", "")
	gaps := []string{"5", "60", "15", "90"}
	playThrough(g, func(r int) (string, string) { return "0", gaps[r%len(gaps)] })
	g.Act("b", "letter", "", "Za rok gramy dalej")
	all(g, "quit")

	at(t, march.AddDate(1, 0, 0))
	g.Act("a", "letters", "", "") // opened in the menu…
	g.Act("a", "letters", "", "")
	g.Act("a", "config", "mode", "rocznica")
	if err := g.Act("a", "start", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.rounds != 8 {
		t.Fatalf("%d rounds, want the 8 from history", g.rounds)
	}
	for i, c := range g.deck {
		if c.before == nil || c.mode != "skala" {
			t.Fatalf("card %d is not from history: %+v", i, c)
		}
	}
	// a question about one of them is about the same person as the first time
	if who := g.before.Names[g.before.Subj]; g.players[g.subj].Name != who {
		t.Fatalf("the question is about %s, was about %s", g.players[g.subj].Name, who)
	}
	playThrough(g, same)
	var v struct {
		Letters []viewLetter
		Scored  bool
	}
	json.Unmarshal(g.view("a"), &v)
	if len(v.Letters) != 1 || !v.Scored { // …and back at the end of the anniversary
		t.Fatalf("end of the anniversary: %+v", v)
	}
}

func TestTimeCapsule(t *testing.T) {
	at(t, march)
	g := started(t, "wolisz")
	if g.Act("a", "letter", "", "za wcześnie") == nil {
		t.Fatal("a letter is written at the end of a game")
	}
	playThrough(g, same)
	if g.Act("a", "letter", "", "  Pierwsza   wersja ") != nil || g.Act("a", "letter", "", "Kocham Cię") != nil || g.Act("b", "letter", "", "Do zobaczenia za rok") != nil {
		t.Fatal("both can leave a letter")
	}
	if g.Act("a", "letter", "", strings.Repeat("x", maxLetter+1)) == nil || g.Act("a", "letter", "", "   ") == nil {
		t.Fatal("an empty or too long letter must be rejected")
	}
	var end struct {
		CanWrite bool
		MyLetter string
	}
	json.Unmarshal(g.view("a"), &end)
	if !end.CanWrite || end.MyLetter != "Kocham Cię" || len(g.store.letters) != 2 {
		t.Fatalf("end: %+v, %d letters (writing again replaces the letter)", end, len(g.store.letters))
	}
	if strings.Contains(string(g.view("b")), "Kocham") || strings.Contains(string(g.view("")), "Kocham") {
		t.Fatal("a letter must stay secret until it opens")
	}
	all(g, "quit")

	var lobby struct {
		LettersDue int
		Letters    []viewLetter
	}
	at(t, march.AddDate(0, 11, 0))
	json.Unmarshal(g.view(""), &lobby)
	if lobby.LettersDue != 0 || g.Act("a", "letters", "", "") == nil {
		t.Fatal("a letter must wait a whole year")
	}
	at(t, march.AddDate(1, 0, 1))
	json.Unmarshal(g.view(""), &lobby)
	if lobby.LettersDue != 2 {
		t.Fatalf("%d letters due, want 2", lobby.LettersDue)
	}
	if err := g.Act("b", "letters", "", ""); err != nil {
		t.Fatal(err)
	}
	lobby.LettersDue, lobby.Letters = 0, nil
	json.Unmarshal(g.view(""), &lobby)
	if len(lobby.Letters) != 2 || lobby.Letters[0].Text != "Kocham Cię" || lobby.Letters[0].When != "w marcu 2026" || lobby.LettersDue != 0 {
		t.Fatalf("opened: %+v", lobby)
	}
	if g.Act("a", "letters", "", "") != nil || g.Act("a", "letters", "", "") == nil {
		t.Fatal("letters hide, and there is nothing left to open")
	}
}

func TestMonthSummary(t *testing.T) {
	at(t, time.Date(2026, 10, 10, 20, 0, 0, 0, time.Local))
	g := started(t, "wolisz")
	playThrough(g, same)
	all(g, "quit")
	var v struct{ Month *monthView }
	json.Unmarshal(g.view(""), &v)
	if v.Month != nil {
		t.Fatal("no summary of a month still going on")
	}
	at(t, time.Date(2026, 11, 2, 19, 0, 0, 0, time.Local))
	json.Unmarshal(g.view(""), &v)
	if v.Month == nil || v.Month.Name != "październik" || v.Month.Games != 1 || v.Month.Evenings != 1 || v.Month.Compat != 100 || v.Month.Best == nil {
		t.Fatalf("summary of October: %+v", v.Month)
	}
	if err := g.Act("a", "month", "", ""); err != nil {
		t.Fatal(err)
	}
	v.Month = nil
	json.Unmarshal(g.view(""), &v)
	if v.Month != nil {
		t.Fatal("the summary goes away once seen")
	}
}

func TestCorrectedQuestionKeepsItsHistory(t *testing.T) {
	old := question{Q: "Wyjazd w ciemno — kierunek losowany"}.id("tak")
	if old != "141fea96026a883d" {
		t.Fatalf("the old id in questions/tak.json does not match: %s", old)
	}
	s := openStore(t.TempDir())
	s.asked[old], s.banned[old] = true, true
	s.rounds = []roundRec{{ID: old, Mode: "tak"}}
	s.migrate(loadQuestions(s.dir))
	id := question{Q: "Wyjazd w ciemno, kierunek losowany"}.id("tak")
	if !s.asked[id] || !s.banned[id] || s.asked[old] || s.rounds[0].ID != id {
		t.Fatal("asked, banned and rounds must move to the corrected question")
	}
}
