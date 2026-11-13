package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

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
		all(g, "next")
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
	all(g, "next")
	g.Act("a", "answer", "", "0")
	g.Act("b", "answer", "", "1")
	if g.points != [2]int{0, 1} {
		t.Fatalf("a miss gives nothing: %v", g.points)
	}

	g = started(t, "kto")
	var v struct{ Q viewQ }
	json.Unmarshal(g.view(""), &v)
	if v.Q.Options[0] != "Natalia" || v.Q.Options[1] != "Jan" || g.subject() != -1 {
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
	all(g, "next")
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
	all(g, "quit")
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
	all(g, "skip") // skipped before picking: nothing was shown
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
	all(g, "next")
	g.Act("b", "answer", "0", three) // now b bluffs
	g.Act("a", "answer", "", "2")
	if g.hit || g.points != [2]int{0, 2} {
		t.Fatalf("a successful bluff scores for the bluffer: %v", g.points)
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
	g.Act("a", "draw", "new:3", "1,1")
	g.Act("a", "draw", "new:5", "2,2,3,3")
	g.Act("a", "undo", "", "")
	if len(g.strokes) != 1 || g.ink != 2 || g.Act("b", "undo", "", "") == nil {
		t.Fatalf("undo removes the drawer's last stroke: %v ink=%d", g.strokes, g.ink)
	}
	if !slices.Equal(g.colors, []int{3}) || !strings.Contains(string(g.view("")), `"colors":[3]`) || g.Act("a", "draw", "new:6", "1,1") == nil {
		t.Fatalf("each stroke keeps its color, and only the palette's: %v", g.colors)
	}
	g.cur.Alt = []string{"Synonim"}
	g.Act("b", "answer", "", "synonim")
	if g.phase != "reveal" || !g.hit || g.points != [2]int{1, 2} {
		t.Fatalf("an alternative word counts as a right guess: phase=%s points=%v", g.phase, g.points)
	}
	all(g, "next") // round 2: b draws, a gives up
	if g.Act("b", "giveup", "", "") == nil || g.Act("a", "giveup", "", "") != nil || g.phase != "reveal" || g.hit {
		t.Fatalf("only the guesser can give up, which reveals the word: phase=%s", g.phase)
	}
	all(g, "next") // round 3: a draws again
	word = g.cur.Q
	g.Act("b", "answer", "", " "+strings.ToUpper(word))
	if g.phase != "reveal" || !g.hit || g.points != [2]int{2, 4} {
		t.Fatalf("a right guess ends the round: phase=%s points=%v", g.phase, g.points)
	}

	// the clock: an unguessed drawing ends by itself, and an old timer never touches a later round
	drawTime = 30 * time.Millisecond
	all(g, "next")
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
	if err := all(g, "skip"); err != nil {
		t.Fatal(err)
	}
	if g.round != 2 || g.phase != "question" || g.points != [2]int{} || g.matches != 0 || !g.store.asked[first.id("znasz")] {
		t.Fatalf("skip moves on without points and uses the question up: round=%d phase=%s points=%v", g.round, g.phase, g.points)
	}
	if rule("") != "" {
		t.Fatal("the rules are explained once per mode")
	}
	for g.phase == "question" {
		all(g, "skip")
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
		all(g, "skip")
	}
	if explained != 4 {
		t.Fatalf("a short mix has 4 modes, each explained once; got %d", explained)
	}
	all(g, "quit")
	if rule("") != rules["mix"] {
		t.Fatal("the menu explains the selected game")
	}
}

// "Nigdy przenigdy" is "Co wolisz" with fixed answers; "Szacowanie" is "Skala" on each question's own range.
func TestNigdyAndSzacowanie(t *testing.T) {
	g := started(t, "nigdy")
	if q := g.viewQ(g.pair, 0); q.Kind != "choice" || !slices.Equal(q.Options, nigdyOptions) {
		t.Fatalf("nigdy: %+v", q)
	}
	g.Act("a", "answer", "", "1")
	g.Act("b", "answer", "", "1")
	if g.phase != "reveal" || !g.hit || g.points != [2]int{} {
		t.Fatalf("nigdy: the same answer is a match, without points: hit=%v points=%v", g.hit, g.points)
	}

	g = started(t, "szacowanie")
	g.cur.Max = 20
	if q := g.viewQ(g.pair, 0); q.Kind != "scale" || q.Max != 20 || g.Act("a", "answer", "", "21") == nil {
		t.Fatalf("szacowanie: the scale ends at the question's max: %+v", q)
	}
	g.Act("a", "answer", "", "10")
	g.Act("b", "answer", "", "13") // 3 off out of 20 is 15%: two points, still a match
	if g.gain != 2 || !g.hit || g.points != [2]int{0, 2} {
		t.Fatalf("szacowanie: gain=%d hit=%v points=%v", g.gain, g.hit, g.points)
	}
}
