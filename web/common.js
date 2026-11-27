// Shared by the projector and the phones.
const $ = id => document.getElementById(id);

// h('div', {className: 'x'}, child, ...) -> element; null/false children are skipped.
function h(tag, props, ...kids) {
  const el = Object.assign(document.createElement(tag), props);
  el.append(...kids.flat(Infinity).filter(k => k != null && k !== false));
  return el;
}

// Updates the children of parent to look like kids, reusing the elements already there: a tap on a
// setting changes only what changed, so nothing flickers, scrolled rows keep their place and a field
// being typed in keeps its text and focus. A new screen still replaces everything (with replaceChildren).
const HANDLERS = ['onclick', 'oninput', 'onchange', 'onsubmit', 'onscroll', 'ontoggle'];
function morph(parent, kids) {
  kids = kids.flat(Infinity).filter(k => k != null && k !== false).map(k => k instanceof Node ? k : document.createTextNode(String(k)));
  kids.forEach((next, i) => {
    const old = parent.childNodes[i];
    if (old === next) return;
    // a canvas carries its painting, not attributes: always take the new one
    if (!old || old.nodeName !== next.nodeName || next.nodeName === 'CANVAS') {
      return old ? parent.replaceChild(next, old) : parent.append(next);
    }
    if (next.nodeType === 3) { if (old.data !== next.data) old.data = next.data; return; }
    for (const a of [...old.attributes]) if (!next.hasAttribute(a.name)) old.removeAttribute(a.name);
    for (const a of next.attributes) if (old.getAttribute(a.name) !== a.value) old.setAttribute(a.name, a.value);
    for (const ev of HANDLERS) old[ev] = next[ev];
    if (next.type === 'range' && old.value !== next.value) old.value = next.value; // the slider follows its draft; typed text is never overwritten
    morph(old, [...next.childNodes]);
  });
  while (parent.childNodes.length > kids.length) parent.lastChild.remove();
}

// The menu shows each game as a card: an icon and what it is about, in a few words.
const MODE_INFO = {
  wolisz: ['🤔', 'Dwie opcje, liczy się zgodność'], kto: ['👉', 'Wskazujecie, kogo dotyczy pytanie'],
  nigdy: ['🙊', 'Zdarzyło się czy nigdy?'], znasz: ['🧩', 'Jedno odpowiada, drugie zgaduje'],
  skala: ['🎚️', 'Suwak od 0 do 100 i zgadywanie'], szacowanie: ['🔢', 'Liczba o sobie, partner zgaduje'],
  telepatia: ['🔮', 'Czy wpiszecie to samo słowo?'], tak: ['✅', 'Tak albo nie, liczą się wspólne TAK'],
  rozmowy: ['🕯️', 'Karty do rozmowy, bez punktów'], wyzwania: ['🎯', 'Prawda albo zadanie'],
  blef: ['🃏', 'Dwie prawdy i jedno kłamstwo'], rysowanie: ['🎨', 'Rysujesz palcem, partner zgaduje'],
  rocznica: ['🎂', 'Pytania z Waszej historii'], mix: ['🎲', 'Gra sama miesza tryby'],
};

const optName = (s, key) => (s.menu[key].find(o => o.id === s.cfg[key]) || {}).name || '';
const percent = s => s.rounds ? Math.round(100 * s.matches / s.rounds) : 0;

// In "znasz" and "skala" the question is about one player (the subject); the other one guesses.
const subj = s => s.subject >= 0 ? s.players[s.subject] : null;
const guesser = s => s.subject >= 0 ? s.players[1 - s.subject] : null;

// s.mode is the mode of the current round; s.cfg.mode is the chosen game (they differ in "mix").
const cardMode = s => s.cfg.mode === 'tak' || s.cfg.mode === 'rozmowy'; // cards instead of rounds
// a question asked again: when it was before and, in the reveal, what each of the two said then
const againLine = s => s.before && h('p', { className: 'again' }, s.before.said.some(Boolean)
  ? [`🔁 ${s.before.when[0].toUpperCase()}${s.before.when.slice(1)}: `, s.players.map((p, i) => s.before.said[i] && [i > 0 && ' · ', h('b', { style: 'color:' + p.color }, p.name), ': ' + s.before.said[i]])]
  : `🔁 To pytanie już było ${s.before.when}. Ciekawe, czy coś się zmieniło.`);

// letters from the time capsule: the sentence, who wrote it and when
const letterList = ls => h('div', { className: 'letters' }, ls.map(l => h('figure', {}, h('blockquote', {}, `„${l.text}”`), h('figcaption', {}, `${l.name}, ${l.when}`))));

// "Pytanie o Natalię": the accusative of a first name, by its ending (a guess for unusual names)
// (Natalia → Natalię, Marek → Marka, Paweł → Pawła, Jerzy → Jerzego, Piotr → Piotra; each part of Aleksandra-Maria)
const accusative = name => name.split(/([ -])/).map(n =>
  /a$/i.test(n) ? n.slice(0, -1) + 'ę' : /ek$/i.test(n) ? n.slice(0, -2) + 'ka' : /eł$/i.test(n) ? n.slice(0, -2) + 'ła'
  : /y$/i.test(n) && n.length > 2 ? n.slice(0, -1) + 'ego' : /i$/i.test(n) && n.length > 2 ? n + 'ego' : /[bcćdfghjklłmnńprsśtwzźż]$/i.test(n) ? n + 'a' : n).join('');

const modeName = s => (s.menu.mode.find(o => o.id === s.mode) || {}).name || '';

// Headline of a reveal.
function verdict(s) {
  const m = s.mode, who = guesser(s);
  const text =
    m === 'skala' || m === 'szacowanie' ? `Różnica ${Math.abs(s.answers[0] - s.answers[1])} · ` + (s.gain ? `+${s.gain} pkt: ${who.name}` : 'bez punktów')
    : m === 'znasz' ? (s.hit ? `Trafione! +1 pkt: ${who.name}` : 'Pudło!')
    : m === 'blef' ? (s.hit ? `Kłamstwo wykryte! +1 pkt: ${who.name}` : `Blef się udał! +1 pkt: ${subj(s).name}`)
    : m === 'rysowanie' ? (s.hit ? `Zgadnięte! +2 pkt: ${who.name}, +1 pkt: ${subj(s).name}` : 'Hasło nieodgadnięte')
    : m === 'telepatia' ? (s.hit ? 'Telepatia! To samo słowo' : 'Tym razem inne słowa')
    : s.hit ? 'Zgodni!' : 'Tym razem inaczej';
  return { hit: m === 'skala' || m === 'szacowanie' ? s.gain > 0 : s.hit, text };
}

// a long name shortened where room is scarce (the top bar): "Aleksandra-Maria" is "Aleksandra"
const short = n => { const f = n.split(/[ -]/)[0]; return f.length > 11 ? f.slice(0, 10) + '…' : f; };
const score = s => `${s.players[0].name} ${s.points[0]} : ${s.points[1]} ${s.players[1].name}`;
const chips = items => h('div', { className: 'list' }, items.map(t => h('span', {}, t)));

// With several couples, dares, bluffs and drawings go round the table: s.active is the couple playing now.
const activePair = s => s.pairs.find(p => p.n === s.active);
const ranked = s => s.multi && !s.active; // everyone answers at once: the projector shows the ranking

// Several couples at one projector are ranked by score.
const pairName = p => p.players.map(x => x.name).join(' i ');
function ranking(s, round) {
  const rows = [...s.pairs].sort((a, b) => b.score - a.score);
  return h('table', { className: 'ranking' }, rows.map((p, i) => h('tr', { className: round && p.hit ? 'hit' : '' },
    h('td', {}, `${i + 1}.`),
    h('td', {}, pairName(p), round && p.detail && h('small', {}, p.detail)),
    round && h('td', {}, p.gain ? `+${p.gain}` : p.hit ? '✓' : '✗'),
    h('td', {}, `${p.score} pkt`))));
}

// End of game: compatibility (or the mutual yeses), points and the revealed rewards.
// short: the phone's version, without what is read together on the projector (talk, letters)
function summary(s, short = false) {
  if (s.multi) {
    const top = Math.max(...s.pairs.map(p => p.score)), best = s.pairs.filter(p => p.score === top);
    return [
      h('p', { className: 'dim' }, 'Najbardziej zgrana para'),
      h('h2', { className: 'hit' }, best.length === 1 ? pairName(best[0]) : 'Remis na szczycie!'),
      ranking(s),
      s.found && h('p', { className: 'dim' }, 'Wasze wspólne TAK z tego wieczoru'),
      s.found && chips(s.found),
    ];
  }
  if (s.cfg.mode === 'rozmowy') return [h('h2', {}, 'Koniec rozmowy'), h('p', { className: 'dim' }, 'Dobrze było pogadać.')];
  const tak = s.cfg.mode === 'tak';
  // what the headline number counts depends on the game
  const count = { tak: 'Wspólne TAK', wyzwania: 'Zaliczone zadania', blef: 'Wykryte kłamstwa', rysowanie: 'Odgadnięte hasła' }[s.cfg.mode];
  const together = s.cfg.mode === 'mix' || s.cfg.mode === 'rocznica';
  return [
    h('p', { className: 'dim' }, count || (together ? 'Wspólny wynik' : 'Wasza zgodność')),
    h('div', { className: 'big' + (count ? ' half' : '') }, tak ? String(s.matches) : count ? `${s.matches} z ${s.rounds}` : percent(s) + '%'),
    !tak && s.found && h('p', { className: 'dim' }, 'Wasze wspólne TAK z tego wieczoru'),
    s.found && chips(s.found),
    s.scored && h('h2', {}, s.winner < 0 ? 'Remis!' : `Wygrywa ${s.players[s.winner].name}!`),
    s.scored && h('p', {}, score(s)),
    (s.stakes || []).map((t, i) => t && h('p', { className: 'stake', style: '--c:' + s.players[i].color }, `Nagroda dla: ${s.players[i].name} · ${t}`)),
    // where the two answers were furthest apart: something to talk about after the game
    !short && s.talk && [h('p', { className: 'dim' }, 'Do pogadania'), h('div', { className: 'talk' }, s.talk.map(q => h('p', {}, q)))],
    !short && s.letters && [h('p', { className: 'dim' }, '📬 Wasze listy z kapsuły czasu'), letterList(s.letters)],
  ];
}

// Countdown for the timed mode. The server sends the time left with each state;
// between states the page counts down by itself.
let endAt = 0;
const secs = () => Math.max(0, Math.ceil((endAt - Date.now()) / 1000)) + ' s';
function clock(s) {
  endAt = Date.now() + (s.left || 0);
  return h('p', { id: 'timer', className: 'timer' }, secs());
}
setInterval(() => { if ($('timer')) $('timer').textContent = secs(); }, 250);

// The ends of a scale: 0-100 with words ("skala"), or 0 to the question's own max ("szacowanie").
const ends = q => q.max ? [`0${q.options[0] ? ' · ' + q.options[0] : ''}`, `${q.options[1] ? q.options[1] + ' · ' : ''}${q.max}`]
  : ['0 · ' + q.options[0], q.options[1] + ' · 100'];

// Drawings are strokes of x,y,x,y… on a 1000x1000 canvas, on the phone and on the projector alike.
// Pen colors; a stroke carries its index. As many as drawColors in modes.go.
const PALETTE = [['#111', 'czarny'], ['#e53935', 'czerwony'], ['#1e66f5', 'niebieski'], ['#2e9e44', 'zielony'], ['#f59e0b', 'pomarańczowy'], ['#8b5a2b', 'brązowy']];
function pen(canvas) {
  const c = canvas.getContext('2d');
  c.lineWidth = 9;
  c.lineCap = c.lineJoin = 'round';
  c.strokeStyle = '#111';
  return c;
}
function line(c, x0, y0, x1, y1) {
  c.beginPath();
  c.moveTo(x0, y0);
  c.lineTo(x1, y1);
  c.stroke();
}
function paint(c, strokes, colors = []) {
  c.clearRect(0, 0, 1000, 1000);
  for (const [n, st] of strokes.entries()) { // a single point is drawn as a dot
    c.strokeStyle = PALETTE[colors[n] || 0][0];
    for (let i = 0; i < st.length; i += 2) line(c, st[Math.max(0, i - 2)], st[Math.max(1, i - 1)], st[i] + (i ? 0 : .1), st[i + 1]);
  }
}

// Live state stream. EventSource reconnects by itself, but iOS Safari can leave it
// dead after the phone was locked, so reopen it when the page becomes visible again.
function live(url, onState) {
  let es;
  const open = () => {
    es?.close();
    es = new EventSource(url);
    es.onmessage = e => { document.body.classList.remove('offline'); onState(JSON.parse(e.data)); };
    es.onerror = () => {
      document.body.classList.add('offline');
      if (es.readyState === EventSource.CLOSED) setTimeout(() => es.readyState === EventSource.CLOSED && open(), 3000); // the browser gave up (e.g. a 502 while the server restarts)
    };
  };
  open();
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState !== 'visible') return;
    open(); // Safari may keep a dead stream OPEN after unlocking, so always start a fresh one
    keepAwake();
  });
}

// Keep the display on during the game (needs https; silently unavailable otherwise).
function keepAwake() {
  navigator.wakeLock?.request('screen').catch(() => {});
}
keepAwake();

// Sound effects are synthesised (no audio files). Browsers only allow sound after a tap,
// a click or a key press on the page; each page unlocks it its own way.
const SFX = { // [frequency Hz, start s, length s, waveform]
  join: [[523, 0, .15], [784, .12, .25]],
  tick: [[880, 0, .06]],
  next: [[440, 0, .08], [660, .07, .1]],
  hit: [[523, 0, .12], [659, .1, .12], [784, .2, .35]],
  miss: [[196, 0, .4, 'sawtooth']],
  clock: [[1200, 0, .04]],
  end: [[523, 0, .15], [659, .15, .15], [784, .3, .15], [1047, .45, .6]],
};
let audio, muted = false;
try { audio = new AudioContext(); } catch {}
const audible = () => audio && audio.state === 'running' && !muted;
function play(name, delay = 0, volume = 1) {
  if (!audible()) return;
  for (const [freq, at, len, type = 'sine'] of SFX[name]) {
    const osc = audio.createOscillator(), gain = audio.createGain(), t = audio.currentTime + delay + at;
    osc.type = type;
    osc.frequency.value = freq;
    gain.gain.setValueAtTime((type === 'sine' ? .2 : .08) * volume, t);
    gain.gain.exponentialRampToValueAtTime(.001, t + len);
    osc.connect(gain).connect(audio.destination);
    osc.start(t);
    osc.stop(t + len);
  }
}

// Answers have a shape and a color, the same on the projector and on the phones, so a phone
// works like a game pad: find the color, tap. In "Kto prędzej" the answers are the players: their colors.
const SHAPES = ['▲', '◆', '●', '■'];
const SHAPE_COLORS = ['#ff3d9a', '#3db8ff', '#b98cff', '#2ee6d6']; // the neon of the theme
const optColor = (s, i) => s.mode === 'kto' && s.players[i] ? s.players[i].color : SHAPE_COLORS[i % SHAPE_COLORS.length];

// A short burst of confetti for a match; purely decorative, and skipped when motion is reduced.
function confetti() {
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
  const box = h('div', { className: 'confetti', ariaHidden: true }, Array.from({ length: 36 }, (_, i) => h('i', {
    style: `--x:${Math.random() * 100}vw;--d:${(Math.random() * .5).toFixed(2)}s;--r:${Math.round(Math.random() * 720 - 360)}deg;--c:${SHAPE_COLORS[i % 4]}`,
  })));
  document.body.append(box);
  setTimeout(() => box.remove(), 2600);
}
