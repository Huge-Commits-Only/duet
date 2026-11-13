package game

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestJoin(t *testing.T) {
	g := New(t.TempDir())
	for _, bad := range [][3]string{{"", "Natalia", "#ff00aa"}, {"a", "  ", "#ff00aa"}, {"a", "Natalia", "red"}, {"a", "abcdefghijklmnopqrstu", "#ff00aa"}} {
		if g.join(bad[0], bad[1], bad[2]) != ErrBad {
			t.Errorf("join%v: want ErrBad", bad)
		}
	}
	a, b := g.Subscribe("a"), g.Subscribe("b")
	if g.join("a", "Natalia", "#ff00aa") != nil || g.join("b", "Jan", "#00ffaa") != nil {
		t.Fatal("two players must fit")
	}
	if g.join("b", "Jan", "#ff00aa") != ErrColor {
		t.Error("two online players must not share a color")
	}
	if g.join("c", "Trzeci", "#0000ff") != ErrFull {
		t.Error("third player while both online: want ErrFull")
	}
	if g.join("a", "Olga", "#ff00aa") != nil || g.players[0].Name != "Olga" || len(g.players) != 2 {
		t.Error("rejoin with same id must update in place")
	}
	g.Unsubscribe(b)
	if g.join("c", "Trzeci", "#00ffaa") != nil || g.players[1].ID != "c" {
		t.Error("new phone must take the offline player's seat")
	}

	var v struct {
		Players []player
		Me      int
	}
	json.Unmarshal(<-a, &v)
	if v.Me != 0 || len(v.Players) != 2 || !v.Players[0].Online || v.Players[1].Online {
		t.Errorf("view for a: %+v", v)
	}
}

func TestWoliszFlow(t *testing.T) {
	dir := t.TempDir()
	g := New(dir)
	a := g.Subscribe("a")
	g.Subscribe("b")
	g.join("a", "Natalia", "#ff00aa")
	if g.Act("a", "start", "", "") == nil {
		t.Fatal("start needs two players")
	}
	g.join("b", "Jan", "#00ffaa")
	if g.Act("x", "start", "", "") == nil || g.Act("a", "config", "tone", "ostro") == nil {
		t.Fatal("strangers and unknown options must be rejected")
	}
	if err := g.Act("b", "config", "tone", "romantycznie"); err != nil || g.cfg["mode"] != "mix" {
		t.Fatal("the menu starts on Wieczór mix", err)
	}
	g.Act("a", "config", "mode", "wolisz")
	if err := g.Act("a", "start", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.rounds != 8 || g.cur.Tone != "romantycznie" {
		t.Fatalf("rounds=%d tone=%q", g.rounds, g.cur.Tone)
	}

	// b answered, a did not: a must not learn b's answer before the reveal
	g.Act("b", "answer", "", "1")
	var v struct {
		Answered [2]bool
		MyAnswer *int
		Answers  *[2]int
	}
	json.Unmarshal(<-a, &v)
	if !v.Answered[1] || v.Answered[0] || v.MyAnswer != nil || v.Answers != nil {
		t.Fatalf("leaky view: %+v", v)
	}
	if g.Act("a", "next", "", "") == nil || g.Act("a", "answer", "", "2") == nil {
		t.Fatal("next before reveal and bad answers must be rejected")
	}

	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		g.Act("a", "answer", "", "1")
		g.Act("b", "answer", "", string(rune('0'+i%2))) // round 1: b already said "1"; then agree on odd i
		if g.phase != "reveal" {
			t.Fatalf("round %d: phase %s", i, g.phase)
		}
		seen[g.cur.id("wolisz")] = true
		all(g, "next")
	}
	if g.phase != "end" || g.matches != 5 || len(seen) != 8 {
		t.Fatalf("phase=%s matches=%d distinct=%d", g.phase, g.matches, len(seen))
	}

	// a restarted server remembers what was asked; 12 in the pack, 8 used -> too few fresh ones, so the pool restarts
	g2 := New(dir)
	g2.players = g.players
	g2.Subscribe("a") // a couple with nobody connected does not play
	g2.cfg["tone"] = "romantycznie"
	g2.cfg["length"] = "krotka"
	if len(g2.store.asked) != 8 || g2.start() != nil {
		t.Fatalf("asked=%d", len(g2.store.asked))
	}
	if g2.rounds != 8 {
		t.Fatalf("exhausted pool must restart to a full game, got %d rounds", g2.rounds)
	}

	if g.Act("a", "quit", "", "") != nil || g.phase != "lobby" {
		t.Fatal("quit must return to the lobby")
	}
}

// started returns a game in its first question, with players a (seat 0) and b (seat 1).
func started(t *testing.T, mode string) *Game {
	g := New(t.TempDir())
	g.Subscribe("a")
	g.Subscribe("b")
	g.join("a", "Natalia", "#ff00aa")
	g.join("b", "Jan", "#00ffaa")
	g.Act("a", "stake", "", "masaż")
	g.Act("b", "stake", "", "zmywanie")
	if err := g.Act("a", "config", "mode", mode); err != nil {
		t.Fatal(err)
	}
	if err := g.Act("a", "start", "", ""); err != nil {
		t.Fatal(err)
	}
	return g
}

func endView(g *Game, id string) (v struct {
	Winner int
	Stakes [2]string
	Points [2]int
}) {
	json.Unmarshal(g.view(id), &v)
	return v
}

func TestSeveralCouples(t *testing.T) {
	g := New(t.TempDir())
	for _, id := range []string{"a", "b", "c", "d"} {
		g.Subscribe(id)
	}
	seat := func(id, name, color string, n int) error { return g.JoinAs(id, name, color, "", g.key, n) }
	if seat("a", "Natalia", "#ff0000", 0) != nil || seat("b", "Jan", "#00ff00", 0) != nil || seat("c", "Kasia", "#0000ff", 1) != nil {
		t.Fatal("joining two couples")
	}
	if seat("x", "Piąty", "#123456", 0) != ErrFull || seat("x", "Piąty", "#123456", 4) != ErrBad {
		t.Fatal("a complete couple takes no third player; there are four couples at most")
	}
	if g.Act("a", "start", "", "") != errAlone {
		t.Fatal("a player without a partner must block the start")
	}
	seat("d", "Tomek", "#ffff00", 1)
	g.Act("a", "config", "mode", "rozmowy")
	if g.Act("a", "start", "", "") != errSolo {
		t.Fatal("a talk is for one couple only")
	}
	g.Act("c", "config", "mode", "skala")
	g.Act("c", "stake", "", "masaż")
	if err := g.Act("d", "start", "", ""); err != nil || len(g.playing) != 2 {
		t.Fatalf("start with two couples: %v", err)
	}

	// each phone sees its own couple; the question is about seat 0 of every couple
	var c struct {
		Players  []player
		Me, Pair int
		Multi    bool
		Scored   bool
		Subject  int
		Pairs    []viewPair
		Answers  *[2]int
		Hit      bool
		MyAnswer *int
	}
	json.Unmarshal(g.view("c"), &c)
	if c.Players[0].Name != "Kasia" || c.Me != 0 || c.Pair != 2 || !c.Multi || c.Scored || c.Subject != 0 || len(c.Pairs) != 2 {
		t.Fatalf("view of the second couple: %+v", c)
	}

	g.Act("a", "answer", "", "50")
	g.Act("b", "answer", "", "52")
	g.Act("c", "answer", "", "10")
	if g.phase != "question" {
		t.Fatal("the round waits for every couple")
	}
	g.Act("d", "answer", "", "90")
	p0, p1 := g.pairs[0], g.pairs[1]
	if g.phase != "reveal" || p0.score != 3 || p0.points != [2]int{0, 3} || !p0.hit || p1.score != 0 || p1.hit {
		t.Fatalf("scores per couple: %d %v / %d", p0.score, p0.points, p1.score)
	}
	c.Answers = nil
	json.Unmarshal(g.view("c"), &c)
	if c.Answers == nil || *c.Answers != [2]int{10, 90} || c.Hit {
		t.Fatalf("a phone sees its own couple's reveal: %+v", c.Answers)
	}
	screen := string(g.view(""))
	if strings.Contains(screen, `"answers"`) || !strings.Contains(screen, `"detail":"Natalia: 50 · Jan: 52"`) || !strings.Contains(screen, `"detail":"Kasia: 10 · Tomek: 90"`) {
		t.Fatalf("the projector shows every couple in the ranking, not one couple's answers: %s", screen)
	}

	// someone who walks in during the game only watches
	seat("e", "Gość", "#00ffff", 2)
	if g.Act("e", "skip", "", "") == nil {
		t.Fatal("a couple that is not playing cannot act")
	}
	for g.phase != "end" {
		if g.phase == "reveal" {
			all(g, "next")
		} else {
			all(g, "skip")
		}
	}
	if strings.Contains(string(g.view("d")), "masaż") {
		t.Fatal("rewards are for a single couple only")
	}
	h := g.store.history
	if len(h) != 1 || len(h[0].Pairs) != 2 || h[0].winner() != "Natalia i Jan" || h[0].Pairs[0].Score != 3 {
		t.Fatalf("history: %+v", h)
	}

	// guests who left keep their seats, but neither play nor block the couple that stayed
	all(g, "quit")
	for ch, id := range g.subs {
		if id == "c" || id == "d" {
			g.Unsubscribe(ch)
		}
	}
	g.Act("a", "config", "mode", "rysowanie")
	if g.multi() || g.Act("a", "start", "", "") != nil || len(g.playing) != 1 || g.playing[0] != g.pairs[0] {
		t.Fatalf("with the guests gone the first couple plays alone: multi=%v playing=%d", g.multi(), len(g.playing))
	}

	// in the lobby a player can move to another couple
	all(g, "quit")
	if seat("b", "Jan", "#00ff00", 3) != nil || len(g.pairs[0].players) != 1 || g.pairs[3].players[0].Name != "Jan" {
		t.Fatal("moving to another couple in the lobby")
	}
}

func TestEmptyLobbyView(t *testing.T) {
	// the pages index into these without checking, so they must be lists even when empty
	v := string(New(t.TempDir()).view(""))
	if !strings.Contains(v, `"pairs":[]`) || !strings.Contains(v, `"players":[]`) {
		t.Fatalf("empty lobby: %s", v)
	}
}

func TestCouplesTakingTurns(t *testing.T) {
	g := New(t.TempDir())
	for i, id := range []string{"a", "b", "c", "d"} {
		g.Subscribe(id)
		if err := g.JoinAs(id, strings.ToUpper(id), fmt.Sprintf("#00000%d", i), "", g.key, i/2); err != nil {
			t.Fatal(err)
		}
	}
	g.Act("a", "config", "mode", "rysowanie")
	if err := g.Act("a", "start", "", ""); err != nil {
		t.Fatal(err)
	}
	// twice the rounds of a game for two: every player draws as often as there
	if g.rounds != 8 {
		t.Fatalf("rounds=%d", g.rounds)
	}
	ids := [2][2]string{{"a", "b"}, {"c", "d"}}
	drew := map[string]int{}
	for g.phase != "end" {
		pi := slices.Index(g.pairs[:], g.pair)
		drawer, guesser, other := ids[pi][g.subj], ids[pi][1-g.subj], ids[1-pi]
		drew[drawer]++
		word := g.cur.Q
		// the couple that watches sees no word, cannot draw or guess, and is told whose turn it is
		for _, w := range other {
			if strings.Contains(string(g.view(w)), word) || g.Act(w, "draw", "new", "1,1") == nil || g.Act(w, "answer", "", word) == nil ||
				!strings.Contains(string(g.view(w)), fmt.Sprintf(`"active":%d`, pi+1)) {
				t.Fatalf("round %d: the watching couple must only watch", g.round)
			}
		}
		if g.Act(drawer, "draw", "new", "1,1,2,2") != nil || !strings.Contains(string(g.view("")), `"strokes"`) {
			t.Fatal("the projector shows the drawing of the couple playing")
		}
		if g.Act(guesser, "answer", "", word) != nil || g.phase != "reveal" {
			t.Fatal("the partner guesses")
		}
		all(g, "next") // anyone may move the game on
	}
	if len(drew) != 4 || drew["a"] != 2 || drew["b"] != 2 || drew["c"] != 2 || drew["d"] != 2 {
		t.Fatalf("everyone draws equally often: %v", drew)
	}
	if g.pairs[0].score != 4 || g.pairs[1].score != 4 {
		t.Fatalf("a guessed drawing is a point for the couple: %d %d", g.pairs[0].score, g.pairs[1].score)
	}

	// dares and bluffs go round the table too; the mix may now use every mode but the talk
	all(g, "quit")
	g.Act("a", "config", "mode", "wyzwania")
	g.Act("a", "start", "", "")
	if g.rounds != 12 || g.Act("c", "answer", "", "0") == nil {
		t.Fatalf("dares: rounds=%d; the other couple cannot pick", g.rounds)
	}
	g.Act("a", "answer", "", "1") // a picks a dare
	g.Act("b", "answer", "", "0") // b: done
	if g.pairs[0].score != 1 || g.pair != g.pairs[1] || g.subj != 0 {
		t.Fatalf("then the next couple: score=%d", g.pairs[0].score)
	}
	// a restart keeps whose turn it is
	if r := New(g.store.dir); r.pair != r.pairs[1] || r.round != 2 {
		t.Fatalf("restored turn: round=%d", r.round)
	}
}

func TestKickAndBackup(t *testing.T) {
	g := started(t, "wolisz")
	g.Act("a", "answer", "", "0")
	g.Act("b", "answer", "", "0")
	if g.Act("a", "kick", "1", "1") == nil {
		t.Fatal("nobody is removed in the middle of a game")
	}
	data, err := g.Backup("a")
	if _, err2 := g.Backup("stranger"); err != nil || err2 == nil {
		t.Fatal("only a seated player gets the backup")
	}
	if g.Restore("a", data) == nil {
		t.Fatal("no restore while a game is on")
	}
	all(g, "quit")

	// restore into a fresh server: what was asked comes back
	fresh := New(t.TempDir())
	fresh.Subscribe("z")
	fresh.join("z", "Zoe", "#123456")
	if fresh.Restore("z", []byte(`{"some":"other json"}`)) != errNotBackup || fresh.Restore("z", []byte(`nonsense`)) != errNotBackup {
		t.Fatal("a file that is not a backup must be refused")
	}
	if err := fresh.Restore("z", data); err != nil || len(fresh.store.asked) != 1 {
		t.Fatalf("restore: %v asked=%d", err, len(fresh.store.asked))
	}
	if again := openStore(fresh.store.dir); len(again.asked) != 1 {
		t.Fatal("a restore is written to disk")
	}

	// removing a player from the menu frees the seat
	if g.Act("a", "kick", "1", "5") == nil || g.Act("a", "kick", "9", "0") == nil || g.Act("a", "kick", "1", "1") != nil || len(g.players) != 1 {
		t.Fatalf("kick: %d players left", len(g.players))
	}
	if g.Act("b", "start", "", "") == nil {
		t.Fatal("a removed player is out")
	}
}

// all makes everyone taking part do the same: moving on after a reveal, skipping and quitting
// wait for all of them. With nobody connected, player "a" acts alone.
func all(g *Game, act string) error {
	g.mu.Lock()
	ids := g.present()
	g.mu.Unlock()
	if len(ids) == 0 {
		ids = []string{"a"}
	}
	var err error
	for _, id := range ids {
		if e := g.Act(id, act, "", ""); err == nil {
			err = e
		}
	}
	return err
}
