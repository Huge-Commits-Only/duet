package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestJoin(t *testing.T) {
	g := New(t.TempDir())
	for _, bad := range [][3]string{{"", "Ola", "#ff00aa"}, {"a", "  ", "#ff00aa"}, {"a", "Ola", "red"}, {"a", "abcdefghijklmnopqrstu", "#ff00aa"}} {
		if g.join(bad[0], bad[1], bad[2]) != ErrBad {
			t.Errorf("join%v: want ErrBad", bad)
		}
	}
	a, b := g.Subscribe("a"), g.Subscribe("b")
	if g.join("a", "Ola", "#ff00aa") != nil || g.join("b", "Jan", "#00ffaa") != nil {
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
	g.join("a", "Ola", "#ff00aa")
	if g.Act("a", "start", "", "") == nil {
		t.Fatal("start needs two players")
	}
	g.join("b", "Jan", "#00ffaa")
	if g.Act("x", "start", "", "") == nil || g.Act("a", "config", "tone", "ostro") == nil {
		t.Fatal("strangers and unknown options must be rejected")
	}
	if err := g.Act("b", "config", "tone", "romantycznie"); err != nil {
		t.Fatal(err)
	}
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
		g.Act("b", "next", "", "")
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

func TestBuiltinPacks(t *testing.T) {
	packs := loadQuestions(t.TempDir())
	for _, mode := range menu["mode"] {
		if mode.ID == "mix" {
			continue // draws from the other modes' packs
		}
		for _, tone := range menu["tone"] {
			n := 0
			for _, q := range packs[mode.ID] {
				if q.Tone == tone.ID && q.valid(mode.ID) {
					n++
				}
			}
			if n < roundsFor["krotka"] {
				t.Errorf("%s/%s: %d valid questions, need at least %d", mode.ID, tone.ID, n, roundsFor["krotka"])
			}
			for level := 1; mode.ID == "rozmowy" && level <= 3; level++ {
				if !slices.ContainsFunc(packs[mode.ID], func(q question) bool { return q.Tone == tone.ID && q.Level == level }) {
					t.Errorf("rozmowy/%s: no card on level %d", tone.ID, level)
				}
			}
		}
	}
}

// started returns a game in its first question, with players a (seat 0) and b (seat 1).
func started(t *testing.T, mode string) *Game {
	g := New(t.TempDir())
	g.Subscribe("a")
	g.Subscribe("b")
	g.join("a", "Ola", "#ff00aa")
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

func TestSkalaScoring(t *testing.T) {
	g := started(t, "skala")
	// round 1 is about seat 0, so seat 1 guesses; then they alternate
	for i, r := range []struct{ a, b, gain, pts0, pts1 int }{
		{50, 55, 3, 0, 3}, // off by 5
		{10, 25, 2, 2, 3}, // off by 15, seat 0 guessed
		{0, 30, 1, 2, 4},
		{100, 69, 0, 2, 4},
	} {
		if g.subject() != i%2 {
			t.Fatalf("round %d: subject %d", i+1, g.subject())
		}
		g.Act("a", "answer", "", strconv.Itoa(r.a))
		g.Act("b", "answer", "", strconv.Itoa(r.b))
		if g.gain != r.gain || g.points != [2]int{r.pts0, r.pts1} {
			t.Fatalf("round %d: gain=%d points=%v", i+1, g.gain, g.points)
		}
		g.Act("a", "next", "", "")
	}
	if g.Act("a", "answer", "", "101") == nil || g.Act("a", "answer", "", "-1") == nil || g.Act("a", "answer", "", "x") == nil {
		t.Fatal("answers outside 0-100 must be rejected")
	}
	if g.matches != 2 {
		t.Fatalf("matches=%d, want 2 (rounds within 15)", g.matches)
	}

	// the stake stays secret from the partner until the end, then only the winner's is revealed
	var mid struct {
		MyStake  string
		StakeSet [2]bool
		Stakes   *[2]string
	}
	json.Unmarshal(g.view("a"), &mid)
	if mid.MyStake != "masaż" || mid.StakeSet != [2]bool{true, true} || mid.Stakes != nil {
		t.Fatalf("mid-game view: %+v", mid)
	}
	g.phase = "end"
	if v := endView(g, ""); v.Winner != 1 || v.Stakes != [2]string{"", "zmywanie"} {
		t.Fatalf("end view: %+v", v)
	}
	g.points = [2]int{4, 4}
	if v := endView(g, ""); v.Winner != -1 || v.Stakes != [2]string{"masaż", "zmywanie"} {
		t.Fatalf("draw must reveal both: %+v", v)
	}
}

func TestZnaszAndKto(t *testing.T) {
	g := started(t, "znasz")
	if g.rounds%2 != 0 {
		t.Fatalf("subject modes need an even number of rounds, got %d", g.rounds)
	}
	last := strconv.Itoa(len(g.cur.Options) - 1)
	if g.Act("a", "answer", "", strconv.Itoa(len(g.cur.Options))) == nil {
		t.Fatal("option index out of range must be rejected")
	}
	g.Act("a", "answer", "", last)
	g.Act("b", "answer", "", last)
	if g.points != [2]int{0, 1} || g.matches != 1 {
		t.Fatalf("guesser (seat 1) must score: points=%v matches=%d", g.points, g.matches)
	}
	g.Act("a", "next", "", "")
	g.Act("a", "answer", "", "0")
	g.Act("b", "answer", "", "1")
	if g.points != [2]int{0, 1} {
		t.Fatalf("a miss gives nothing: %v", g.points)
	}

	g = started(t, "kto")
	var v struct{ Q viewQ }
	json.Unmarshal(g.view(""), &v)
	if v.Q.Options[0] != "Ola" || v.Q.Options[1] != "Jan" || g.subject() != -1 {
		t.Fatalf("kto: the choices are the players: %+v", v.Q)
	}
	g.Act("a", "answer", "", "1")
	g.Act("b", "answer", "", "1")
	if g.matches != 1 || g.points != [2]int{} {
		t.Fatalf("kto: matches=%d points=%v", g.matches, g.points)
	}
}

func TestTelepatia(t *testing.T) {
	g := started(t, "telepatia")
	if g.Act("a", "answer", "", "  ") == nil || g.Act("a", "answer", "", strings.Repeat("x", 31)) == nil {
		t.Fatal("empty and overlong words must be rejected")
	}
	g.Act("a", "answer", "", " Plaża")
	var v struct {
		MyText string
		Texts  *[2]string
	}
	json.Unmarshal(g.view("b"), &v)
	if v.MyText != "" || v.Texts != nil {
		t.Fatalf("partner's word leaked before the reveal: %+v", v)
	}
	g.Act("b", "answer", "", "plaza")
	if g.phase != "reveal" || !g.hit || g.matches != 1 {
		t.Fatalf("same word ignoring case and diacritics must match: phase=%s hit=%v", g.phase, g.hit)
	}
	g.Act("a", "next", "", "")
	g.Act("a", "answer", "", "morze")
	g.Act("b", "answer", "", "góry")
	if g.hit || g.matches != 1 {
		t.Fatal("different words must not match")
	}
}

func TestWspolneTak(t *testing.T) {
	g := started(t, "tak")
	dir := g.store.dir
	first := g.cur.Q
	g.Act("a", "answer", "", "0") // yes
	// a said yes, b says no: the card just passes, and nobody's answer is ever in a view
	g.Act("b", "answer", "", "1")
	if g.phase != "question" || g.round != 2 || g.matches != 0 || len(g.store.yes) != 0 {
		t.Fatalf("a non-match must move on silently: phase=%s round=%d", g.phase, g.round)
	}
	if strings.Contains(string(g.view("b")), `"answers"`) || strings.Contains(string(g.view("")), `"answers"`) {
		t.Fatal("answers must never be sent in this mode")
	}
	second := g.cur.Q
	g.Act("a", "answer", "", "0")
	g.Act("b", "answer", "", "0")
	if g.matches != 1 || len(g.found) != 1 || g.found[0] != second || second == first {
		t.Fatalf("mutual yes must be collected: %v", g.found)
	}
	if got := openStore(dir).yes; len(got) != 1 || got[0].Text != second || got[0].Done {
		t.Fatalf("mutual yes must be saved for later evenings: %v", got)
	}

	// the saved list is only sent while a phone asked to show it
	g.Act("a", "quit", "", "")
	if strings.Contains(string(g.view("")), second) {
		t.Fatal("list must stay hidden until asked for")
	}
	g.Act("b", "list", "", "")
	if !strings.Contains(string(g.view("")), second) {
		t.Fatal("list must be shown on request")
	}

	// ticking off and removing, and both survive a restart
	if g.Act("a", "yesdone", "", "nie ma takiego") == nil || g.Act("a", "yesdone", "", second) != nil {
		t.Fatal("only ideas on the list can be ticked off")
	}
	if got := openStore(dir).yes; len(got) != 1 || !got[0].Done {
		t.Fatalf("done must be saved: %+v", got)
	}
	if g.Act("b", "yesdel", "", second) != nil || len(openStore(dir).yes) != 0 {
		t.Fatal("removing must be saved")
	}
	// the list written by older versions was a plain array of texts
	os.WriteFile(filepath.Join(dir, "yes.json"), []byte(`["Stary wpis",{"text":"Nowy wpis","done":true}]`), 0o644)
	if got := openStore(dir).yes; len(got) != 2 || got[0] != (yesItem{"Stary wpis", false}) || got[1] != (yesItem{"Nowy wpis", true}) {
		t.Fatalf("old and new formats must both load: %+v", got)
	}
}

func TestRozmowy(t *testing.T) {
	g := started(t, "rozmowy")
	if g.rounds != 3 || g.Act("a", "answer", "", "0") == nil {
		t.Fatalf("a short talk is 3 cards with nothing to answer, got %d", g.rounds)
	}
	for want := 1; want <= 3; want++ {
		if g.phase != "question" || g.cur.Level != want {
			t.Fatalf("card %d: phase=%s level=%d", want, g.phase, g.cur.Level)
		}
		g.Act("b", "next", "", "")
	}
	if g.phase != "end" || len(g.store.asked) != 3 {
		t.Fatalf("phase=%s asked=%d", g.phase, len(g.store.asked))
	}
	if strings.Contains(string(g.view("")), "masaż") {
		t.Fatal("a conversation reveals no rewards")
	}
	g = started(t, "wolisz")
	g.phase = "end"
	if strings.Contains(string(g.view("")), "masaż") {
		t.Fatal("a game without points has no winner to reward")
	}
}

func TestWyzwania(t *testing.T) {
	g := started(t, "wyzwania")
	if g.rounds != 6 || g.subject() != 0 {
		t.Fatalf("rounds=%d subject=%d", g.rounds, g.subject())
	}
	var q struct{ Q viewQ }
	json.Unmarshal(g.view(""), &q)
	if q.Q.Kind != "pick" || g.Act("b", "answer", "", "0") == nil {
		t.Fatalf("first the one in the spotlight picks truth or dare: %+v", q.Q)
	}
	g.Act("a", "answer", "", "1")
	if g.cur.Kind != "wyzwanie" || g.pending != nil {
		t.Fatalf("picked a dare, got %q", g.cur.Kind)
	}
	if g.Act("a", "answer", "", "0") == nil {
		t.Fatal("the one in the spotlight must not judge themselves")
	}
	g.Act("b", "answer", "", "0") // done
	if g.points != [2]int{1, 0} || g.round != 2 || g.subject() != 1 || g.phase != "question" {
		t.Fatalf("points=%v round=%d subject=%d", g.points, g.round, g.subject())
	}
	g.Act("b", "answer", "", "0") // b picks a truth
	if g.cur.Kind != "prawda" {
		t.Fatalf("picked a truth, got %q", g.cur.Kind)
	}
	g.Act("a", "answer", "", "1") // gave up
	if g.points != [2]int{1, 0} || g.round != 3 {
		t.Fatalf("giving up scores nothing: %v", g.points)
	}
	if len(g.store.asked) != 2 {
		t.Fatalf("only the picked cards are used up, got %d", len(g.store.asked))
	}
	g.Act("a", "skip", "", "") // skipped before picking: nothing was shown
	if len(g.store.asked) != 2 || g.round != 4 {
		t.Fatalf("skip while picking: asked=%d round=%d", len(g.store.asked), g.round)
	}
}

func TestBlef(t *testing.T) {
	g := started(t, "blef")
	three := "Byłem w Japonii\n Mam brata \nBoję się gęsi"
	if g.Act("b", "answer", "1", three) == nil || g.Act("a", "answer", "3", three) == nil ||
		g.Act("a", "answer", "1", "tylko dwa\nzdania") == nil || g.Act("b", "answer", "", "0") == nil {
		t.Fatal("only the bluffer writes: exactly three statements, lie index 0-2; no guessing before that")
	}
	if err := g.Act("a", "answer", "1", three); err != nil {
		t.Fatal(err)
	}
	var v struct {
		Q        viewQ
		MyAnswer *int
	}
	json.Unmarshal(g.view("b"), &v)
	if v.Q.Kind != "choice" || v.Q.Options[1] != "Mam brata" || v.MyAnswer != nil || strings.Contains(string(g.view("b")), `"answers"`) {
		t.Fatalf("guesser must see the statements but not the lie: %+v", v)
	}
	g.Act("b", "answer", "", "1")
	if g.phase != "reveal" || !g.hit || g.points != [2]int{0, 1} {
		t.Fatalf("spotting the lie scores for the guesser: %v", g.points)
	}
	g.Act("a", "next", "", "")
	g.Act("b", "answer", "0", three) // now b bluffs
	g.Act("a", "answer", "", "2")
	if g.hit || g.points != [2]int{0, 2} {
		t.Fatalf("a successful bluff scores for the bluffer: %v", g.points)
	}
}

func TestMix(t *testing.T) {
	g := started(t, "mix")
	if g.rounds != 8 {
		t.Fatalf("a short mix is 8 rounds, got %d", g.rounds)
	}
	modes, spotlight := map[string]int{}, [2]int{}
	for i, c := range g.deck {
		modes[c.mode]++
		if i%2 == 1 && c.mode != g.deck[i-1].mode {
			t.Fatalf("rounds come in pairs of the same mode: %d", i)
		}
	}
	if len(modes) != 4 {
		t.Fatalf("8 rounds are 4 modes, 2 rounds each: %v", modes)
	}
	// play it through with whatever each mode needs
	for g.phase != "end" {
		if s := g.subject(); s >= 0 {
			spotlight[s]++
		}
		ids := [2]string{"a", "b"}
		switch g.mode {
		case "wyzwania":
			g.Act(ids[g.subject()], "answer", "", "1")
			g.Act(ids[1-g.subject()], "answer", "", "0")
			continue
		case "blef":
			g.Act(ids[g.subject()], "answer", "0", "raz\ndwa\ntrzy")
			g.Act(ids[1-g.subject()], "answer", "", "0")
		case "telepatia":
			g.Act("a", "answer", "", "x")
			g.Act("b", "answer", "", "x")
		case "rysowanie":
			g.Act(ids[1-g.subject()], "answer", "", g.cur.Q)
		default:
			g.Act("a", "answer", "", "0")
			g.Act("b", "answer", "", "0")
		}
		if g.phase == "reveal" {
			g.Act("a", "next", "", "")
		}
	}
	if spotlight[0] != spotlight[1] {
		t.Fatalf("spotlight must be shared equally: %v", spotlight)
	}
}

func TestRysowanie(t *testing.T) {
	drawTime = 60 * time.Second
	g := started(t, "rysowanie")
	word := g.cur.Q
	if strings.Contains(string(g.view("b")), word) || strings.Contains(string(g.view("")), word) || !strings.Contains(string(g.view("a")), word) {
		t.Fatal("only the one drawing sees the word")
	}
	if g.Act("b", "draw", "new", "1,2") == nil || g.Act("a", "draw", "new", "1,2,3") == nil || g.Act("a", "draw", "new", "1,1001") == nil || g.Act("a", "answer", "", word) == nil {
		t.Fatal("only the drawer draws, in pairs within 0-1000; the drawer cannot guess")
	}
	g.Act("a", "draw", "new", "10,20,30,40")
	g.Act("a", "draw", "", "50,60")
	g.Act("a", "draw", "new", "0,0")
	if len(g.strokes) != 2 || len(g.strokes[0]) != 6 {
		t.Fatalf("strokes: %v", g.strokes)
	}
	if !strings.Contains(string(g.view("")), `"strokes":[[10,20,30,40,50,60],[0,0]]`) || strings.Contains(string(g.view("b")), "strokes") {
		t.Fatal("the drawing goes to the projector only")
	}
	g.Act("a", "clear", "", "")
	if len(g.strokes) != 0 {
		t.Fatal("clear must wipe the drawing")
	}
	g.Act("b", "answer", "", "na pewno nie to")
	if g.phase != "question" || len(g.guesses) != 1 {
		t.Fatal("a wrong guess keeps the round going")
	}
	g.Act("a", "draw", "new", "1,1")
	g.Act("a", "draw", "new", "2,2,3,3")
	g.Act("a", "undo", "", "")
	if len(g.strokes) != 1 || g.ink != 2 || g.Act("b", "undo", "", "") == nil {
		t.Fatalf("undo removes the drawer's last stroke: %v ink=%d", g.strokes, g.ink)
	}
	g.cur.Alt = []string{"Synonim"}
	g.Act("b", "answer", "", "synonim")
	if g.phase != "reveal" || !g.hit || g.points != [2]int{1, 2} {
		t.Fatalf("an alternative word counts as a right guess: phase=%s points=%v", g.phase, g.points)
	}
	g.Act("a", "next", "", "") // round 2: b draws, a gives up
	if g.Act("b", "giveup", "", "") == nil || g.Act("a", "giveup", "", "") != nil || g.phase != "reveal" || g.hit {
		t.Fatalf("only the guesser can give up, which reveals the word: phase=%s", g.phase)
	}
	g.Act("a", "next", "", "") // round 3: a draws again
	word = g.cur.Q
	g.Act("b", "answer", "", " "+strings.ToUpper(word))
	if g.phase != "reveal" || !g.hit || g.points != [2]int{2, 4} {
		t.Fatalf("a right guess ends the round: phase=%s points=%v", g.phase, g.points)
	}

	// the clock: an unguessed drawing ends by itself, and an old timer never touches a later round
	drawTime = 30 * time.Millisecond
	g.Act("a", "next", "", "")
	time.Sleep(120 * time.Millisecond)
	g.mu.Lock()
	phase, hit, points := g.phase, g.hit, g.points
	g.mu.Unlock()
	if phase != "reveal" || hit || points != [2]int{2, 4} {
		t.Fatalf("time out must reveal without points: phase=%s hit=%v points=%v", phase, hit, points)
	}
	drawTime = 60 * time.Second
}

func TestSkipAndRules(t *testing.T) {
	g := started(t, "znasz")
	rule := func(id string) string {
		var v struct{ Rule string }
		json.Unmarshal(g.view(id), &v)
		return v.Rule
	}
	if rule("") != rules["znasz"] {
		t.Fatal("the first round of a mode explains the rules")
	}
	first := g.cur
	g.Act("a", "answer", "", "0")
	if err := g.Act("b", "skip", "", ""); err != nil {
		t.Fatal(err)
	}
	if g.round != 2 || g.phase != "question" || g.points != [2]int{} || g.matches != 0 || !g.store.asked[first.id("znasz")] {
		t.Fatalf("skip moves on without points and uses the question up: round=%d phase=%s points=%v", g.round, g.phase, g.points)
	}
	if rule("") != "" {
		t.Fatal("the rules are explained once per mode")
	}
	for g.phase == "question" {
		g.Act("a", "skip", "", "")
	}
	if g.phase != "end" || g.Act("a", "skip", "", "") == nil {
		t.Fatalf("skipping the last round ends the game: %s", g.phase)
	}

	for _, m := range menu["mode"] {
		if rules[m.ID] == "" {
			t.Errorf("no rules text for %s", m.ID)
		}
	}

	g = started(t, "mix")
	explained := 0
	for g.phase == "question" {
		if rule("") != "" {
			explained++
		}
		g.Act("a", "skip", "", "")
	}
	if explained != 4 {
		t.Fatalf("a short mix has 4 modes, each explained once; got %d", explained)
	}
	g.Act("a", "quit", "", "")
	if rule("") != rules["mix"] {
		t.Fatal("the menu explains the selected game")
	}
}

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

func TestRestartKeepsTheGame(t *testing.T) {
	g := started(t, "skala")
	dir := g.store.dir
	g.Act("a", "answer", "", "50")
	g.Act("b", "answer", "", "52")
	g.Act("a", "next", "", "")
	g.Act("b", "answer", "", "30") // b already answered round 2 when the server goes down

	r := New(dir)
	if r.phase != "question" || r.round != 2 || r.cur.Q != g.cur.Q || r.points != [2]int{0, 3} || r.answers != [2]int{-1, 30} ||
		r.subj != 1 || r.key != g.key || r.stakes != g.stakes || r.seat("a") != 0 || r.seat("b") != 1 {
		t.Fatalf("restored: phase=%s round=%d points=%v answers=%v subj=%d", r.phase, r.round, r.points, r.answers, r.subj)
	}
	r.Subscribe("a")
	if err := r.Act("a", "answer", "", "31"); err != nil || r.phase != "reveal" || r.points != [2]int{3, 3} {
		t.Fatalf("the restored game must simply continue: %v phase=%s points=%v", err, r.phase, r.points)
	}

	// the join code: needed for a new phone, not for one that already has a seat
	if r.JoinAs("c", "Obcy", "#123456", "", "zły kod", 0) != ErrKey || r.JoinAs("a", "Ola", "#ff00aa", "f", "", 0) != nil || r.players[0].Form != "f" {
		t.Fatal("join code must guard free seats only")
	}
	if r.JoinAs("a", "Ola", "#ff00aa", "x", r.key, 0) != ErrBad {
		t.Fatal("unknown form must be rejected")
	}
	if strings.Contains(string(r.view("a")), r.key) || !strings.Contains(string(r.view("")), r.key) {
		t.Fatal("only the projector shows the join code")
	}

	// a drawing round interrupted by a restart counts as out of time; a broken file gives a fresh lobby
	d := started(t, "rysowanie")
	if r := New(d.store.dir); r.phase != "reveal" || r.hit {
		t.Fatalf("interrupted drawing: phase=%s", r.phase)
	}
	os.WriteFile(filepath.Join(dir, "game.json"), []byte(`{"Phase":"question","Round":7,"Deck":[]`), 0o644)
	if r := New(dir); r.phase != "lobby" || r.key == "" {
		t.Fatalf("broken state file: phase=%s", r.phase)
	}
}

func TestSeveralCouples(t *testing.T) {
	g := New(t.TempDir())
	for _, id := range []string{"a", "b", "c", "d"} {
		g.Subscribe(id)
	}
	seat := func(id, name, color string, n int) error { return g.JoinAs(id, name, color, "", g.key, n) }
	if seat("a", "Ola", "#ff0000", 0) != nil || seat("b", "Jan", "#00ff00", 0) != nil || seat("c", "Kasia", "#0000ff", 1) != nil {
		t.Fatal("joining two couples")
	}
	if seat("x", "Piąty", "#123456", 0) != ErrFull || seat("x", "Piąty", "#123456", 4) != ErrBad {
		t.Fatal("a complete couple takes no third player; there are four couples at most")
	}
	if g.Act("a", "start", "", "") != errAlone {
		t.Fatal("a player without a partner must block the start")
	}
	seat("d", "Tomek", "#ffff00", 1)
	g.Act("a", "config", "mode", "rysowanie")
	if g.Act("a", "start", "", "") != errSolo {
		t.Fatal("drawing is for one couple only")
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
	if strings.Contains(screen, `"answers"`) || !strings.Contains(screen, `"detail":"Ola: 50 · Jan: 52"`) || !strings.Contains(screen, `"detail":"Kasia: 10 · Tomek: 90"`) {
		t.Fatalf("the projector shows every couple in the ranking, not one couple's answers: %s", screen)
	}

	// someone who walks in during the game only watches
	seat("e", "Gość", "#00ffff", 2)
	if g.Act("e", "skip", "", "") == nil {
		t.Fatal("a couple that is not playing cannot act")
	}
	for g.phase != "end" {
		if g.phase == "reveal" {
			g.Act("a", "next", "", "")
		} else {
			g.Act("d", "skip", "", "")
		}
	}
	if strings.Contains(string(g.view("d")), "masaż") {
		t.Fatal("rewards are for a single couple only")
	}
	h := g.store.history
	if len(h) != 1 || len(h[0].Pairs) != 2 || h[0].winner() != "Ola i Jan" || h[0].Pairs[0].Score != 3 {
		t.Fatalf("history: %+v", h)
	}

	// guests who left keep their seats, but neither play nor block the couple that stayed
	g.Act("a", "quit", "", "")
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
	g.Act("a", "quit", "", "")
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

func TestHistoryAndStats(t *testing.T) {
	g := started(t, "skala")
	dir := g.store.dir
	g.Act("a", "quit", "", "")
	if len(g.store.history) != 0 {
		t.Fatal("an abandoned game is not recorded")
	}
	for range 2 { // two finished games: b guesses seat 0's number exactly in round 1, the rest is skipped
		g.Act("a", "quit", "", "") // back to the menu after the previous game
		g.Act("a", "start", "", "")
		g.Act("a", "answer", "", "40")
		g.Act("b", "answer", "", "40")
		g.Act("a", "next", "", "")
		for g.phase == "question" {
			g.Act("a", "skip", "", "")
		}
	}
	st := openStore(dir).stats() // from disk: what the next evening will see
	if st.Games != 2 || st.Evenings != 1 || len(st.Series) != 2 || st.Series[0] != 12 || st.Compat != 12 ||
		len(st.Wins) != 1 || st.Wins[0] != (statLine{Name: "Jan", Value: 2}) || st.Modes[0].Name != "Skala" || len(st.Recent) != 2 {
		t.Fatalf("stats: %+v", st)
	}
	g.Act("a", "quit", "", "")
	if strings.Contains(string(g.view("")), `"stats"`) || g.Act("b", "stats", "", "") != nil || !strings.Contains(string(g.view("")), `"stats"`) {
		t.Fatal("stats are sent only while a phone asked to show them")
	}
	g.Act("a", "list", "", "")
	if g.showStats {
		t.Fatal("the list and the stats replace each other on the projector")
	}
}

func TestPlayingAloneWithBots(t *testing.T) {
	botDelay = time.Millisecond
	defer func() { botDelay = 1500 * time.Millisecond }()
	g := New(t.TempDir())
	g.Subscribe("a")
	g.join("a", "Ola", "#ff00aa")
	// wait until the bots' timers brought the game to the wanted state
	until := func(what string, ok func() bool) {
		t.Helper()
		for range 400 {
			g.mu.Lock()
			done := ok()
			g.mu.Unlock()
			if done {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("bots never got to: %s (mode=%s phase=%s round=%d)", what, g.mode, g.phase, g.round)
	}
	if g.Act("a", "start", "", "") == nil || g.Act("a", "bot", "", "") != nil || !g.players[1].Bot {
		t.Fatal("alone you cannot start; a bot partner fills the seat")
	}
	if !strings.Contains(string(g.view("")), `"online":true,"bot":true`) {
		t.Fatal("a bot counts as present")
	}

	// every mode can be played through alone
	for _, mode := range menu["mode"] {
		if err := g.Act("a", "config", "mode", mode.ID); err != nil {
			t.Fatal(err)
		}
		if err := g.Act("a", "start", "", ""); err != nil {
			t.Fatal(mode.ID, err)
		}
		for round := 1; g.phase != "end"; {
			until(mode.ID+": my turn or a result", func() bool {
				switch {
				case g.phase != "question" || g.round != round:
					return true // revealed, or the round ended without a reveal
				case g.mode == "rozmowy":
					return true
				case g.mode == "wyzwania":
					return g.pending != nil && g.subj == 0 || g.pending == nil && g.subj == 1
				case g.mode == "blef":
					return g.subj == 0 || g.stmts != nil
				case g.mode == "rysowanie":
					return g.subj == 0 || len(g.strokes) > 0
				}
				return true
			})
			g.mu.Lock()
			phase, m, subj, pending, now := g.phase, g.mode, g.subj, g.pending != nil, g.round
			g.mu.Unlock()
			switch {
			case now != round:
				round = now
				continue
			case phase == "reveal":
				g.Act("a", "next", "", "")
			case phase == "end":
			case m == "rozmowy":
				g.Act("a", "next", "", "")
			case m == "wyzwania" && pending:
				g.Act("a", "answer", "", "1")
			case m == "wyzwania" && subj == 1:
				g.Act("a", "answer", "", "0")
			case m == "wyzwania": // my dare: the bot judges
				until("bot judges", func() bool { return g.round != round || g.phase != "question" })
			case m == "blef" && subj == 0 && g.stmts == nil:
				g.Act("a", "answer", "0", "raz\ndwa\ntrzy")
				until("bot guesses", func() bool { return g.phase != "question" })
			case m == "rysowanie" && subj == 1:
				g.Act("a", "giveup", "", "")
			case m == "rysowanie": // I draw: the bot guesses after a while
				until("bot guesses the drawing", func() bool { return g.phase != "question" })
			case m == "telepatia":
				g.Act("a", "answer", "", "morze")
				until("bot answers", func() bool { return g.phase != "question" })
			default:
				g.Act("a", "answer", "", "0")
				until("bot answers", func() bool { return g.phase != "question" || g.round != round })
			}
			g.mu.Lock()
			round = g.round
			g.mu.Unlock()
		}
		g.Act("a", "quit", "", "")
	}
	if len(g.store.history) != 0 {
		t.Fatal("try-outs with a bot must not count in the statistics")
	}

	// a couple of bots to try the ranking alone; a real partner replaces the bot
	if g.Act("a", "botpair", "", "") != nil || len(g.seated()) != 2 || !g.multi() {
		t.Fatal("bot couple")
	}
	g.Act("a", "config", "mode", "wolisz")
	g.Act("a", "start", "", "")
	g.Act("a", "answer", "", "0")
	until("everyone answered", func() bool { return g.phase == "reveal" })
	g.Act("a", "quit", "", "")
	if g.Act("a", "botpair", "", "") != nil || len(g.seated()) != 1 {
		t.Fatal("removing the bot couple")
	}
	g.Subscribe("b")
	if g.join("b", "Jan", "#00ffaa") != nil || g.players[1].Bot || g.players[1].Name != "Jan" {
		t.Fatal("a real partner takes the bot's seat")
	}
}
