package game

import (
	"regexp"
	"strings"
)

// The packs are written for either gender ("zrobiłeś/aś", "chciał(a)byś", "pewny/a").
// gender picks one spelling: you is the form ("m" or "f") of the player the text is
// addressed to, other the form of their partner; "" leaves that person's words as written.
//
// Almost everything addresses "you". The partner only shows up as the one asking,
// in first person: "żebym robił(a)", "powinienem/powinnam".
var (
	reAskerClause = regexp.MustCompile(`(żebym [^,.?!]*?ł)\(a\)`)
	reParen       = regexp.MustCompile(`(ł|sam)\(a\)`)
	reAdjective   = regexp.MustCompile(`y/a($|[^\p{L}])`)
)

const askerVerb, askerShould = "\x00", "\x01" // placeholders, so the rules for "you" leave the asker's words alone

// "partner" in the packs is the other one of the couple. When she is addressed as "f",
// it becomes "partnerka" in the right case. "partnera" is either accusative ("pocałuj partnera")
// or genitive ("szyję partnera"); the word before it tells which.
var (
	rePartner = regexp.MustCompile(`(?:(\p{L}+) )?(nagiego )?partner(owi|em|ze|a)?($|[^\p{L}])`)
	// words followed by the genitive: nouns the partner "owns", and a few prepositions
	genitiveBefore = map[string]bool{
		"dla": true, "do": true, "od": true, "u": true, "bez": true,
		"dłoni": true, "majtki": true, "palec": true, "piosenki": true, "plecach": true, "sutki": true,
		"szyję": true, "ud": true, "finał": true, "ucha": true, "kolanach": true, "ramiona": true,
		"ciało": true, "ciele": true, "bieliznę": true, "włosy": true, "usta": true, "ręce": true,
	}
)

func partnerka(s string) string {
	return rePartner.ReplaceAllStringFunc(s, func(m string) string {
		g := rePartner.FindStringSubmatch(m)
		before, naked, ending, after := g[1], g[2], g[3], g[4]
		word := map[string]string{"": "partnerka", "owi": "partnerce", "em": "partnerką", "ze": "partnerce"}[ending]
		if ending == "a" {
			word = "partnerkę"
			if genitiveBefore[strings.ToLower(before)] {
				word = "partnerki"
			}
		}
		if naked != "" { // "nagiego partnera" is always the accusative here
			naked = "nagą "
		}
		if before != "" {
			before += " "
		}
		return before + naked + word + after
	})
}

func gender(s, you, other string) string {
	if other == "f" && strings.Contains(s, "partner") {
		s = partnerka(s)
	}
	if !strings.ContainsAny(s, "/(") {
		return s
	}
	for prev := ""; prev != s; { // one clause can hold several verbs
		prev, s = s, reAskerClause.ReplaceAllString(s, "${1}"+askerVerb)
	}
	s = strings.ReplaceAll(s, "powinienem/powinnam", askerShould)

	if you != "" {
		f := you == "f"
		pick := func(m, fem string) string {
			if f {
				return fem
			}
			return m
		}
		s = strings.NewReplacer(
			"mógłbyś/mogłabyś", pick("mógłbyś", "mogłabyś"),
			"powinieneś/powinnaś", pick("powinieneś", "powinnaś"),
			"nastolatek/ka", pick("nastolatek", "nastolatka"),
			"łeś/aś", pick("łeś", "łaś"),
			"ym/ą", pick("ym", "ą"),
		).Replace(s)
		s = reParen.ReplaceAllString(s, "${1}"+pick("", "a"))
		s = reAdjective.ReplaceAllString(s, pick("y", "a")+"${1}")
	}

	verb, should := "(a)", "powinienem/powinnam"
	if other != "" {
		verb, should = "", "powinienem"
		if other == "f" {
			verb, should = "a", "powinnam"
		}
	}
	return strings.NewReplacer(askerVerb, verb, askerShould, should).Replace(s)
}

// gendered returns the question with its texts spelled for the given forms.
func (q question) gendered(you, other string) question {
	q.Q, q.A, q.B = gender(q.Q, you, other), gender(q.A, you, other), gender(q.B, you, other)
	opts := make([]string, len(q.Options)) // the deck shares the original slice
	for i, o := range q.Options {
		opts[i] = gender(o, you, other)
	}
	q.Options = opts
	return q
}

// forms says how to spell the current question for one viewer:
// p is their couple and me their seat there (-1 for the projector).
func (g *Game) forms(p *pair, me int) (you, other string) {
	form := func(seat int) string {
		if seat >= 0 && seat < len(p.players) {
			return p.players[seat].Form
		}
		return ""
	}
	switch {
	case me < 0 && g.multi(): // one text on the projector for several couples
		return "", ""
	case g.subj >= 0: // the question is about one player, for everyone watching
		return form(g.subj), form(1 - g.subj)
	case me >= 0: // each phone reads it addressed to its owner
		return form(me), form(1 - me)
	case form(0) == form(1): // the projector can only pick a spelling that fits both
		return form(0), form(0)
	}
	return "", ""
}
