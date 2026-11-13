// Shared by the projector and the phones.
const $ = id => document.getElementById(id);

// h('div', {className: 'x'}, child, ...) -> element; null/false children are skipped.
function h(tag, props, ...kids) {
  const el = Object.assign(document.createElement(tag), props);
  el.append(...kids.flat(Infinity).filter(k => k != null && k !== false));
  return el;
}

const optName = (s, key) => (s.menu[key].find(o => o.id === s.cfg[key]) || {}).name || '';
const percent = s => s.rounds ? Math.round(100 * s.matches / s.rounds) : 0;

// In "znasz" and "skala" the question is about one player (the subject); the other one guesses.
const subj = s => s.subject >= 0 ? s.players[s.subject] : null;
const guesser = s => s.subject >= 0 ? s.players[1 - s.subject] : null;

// s.mode is the mode of the current round; s.cfg.mode is the chosen game (they differ in "mix").
const cardMode = s => s.cfg.mode === 'tak' || s.cfg.mode === 'rozmowy'; // cards instead of rounds
const modeName = s => (s.menu.mode.find(o => o.id === s.mode) || {}).name || '';

// Headline of a reveal.
function verdict(s) {
  const m = s.mode, who = guesser(s);
  const text =
    m === 'skala' ? `Różnica ${Math.abs(s.answers[0] - s.answers[1])} · ` + (s.gain ? `+${s.gain} pkt: ${who.name}` : 'bez punktów')
    : m === 'znasz' ? (s.hit ? `Trafione! +1 pkt: ${who.name}` : 'Pudło!')
    : m === 'blef' ? (s.hit ? `Kłamstwo wykryte! +1 pkt: ${who.name}` : `Blef się udał! +1 pkt: ${subj(s).name}`)
    : m === 'rysowanie' ? (s.hit ? `Zgadnięte! +2 pkt: ${who.name}, +1 pkt: ${subj(s).name}` : 'Czas minął!')
    : m === 'telepatia' ? (s.hit ? 'Telepatia! To samo słowo' : 'Tym razem inne słowa')
    : s.hit ? 'Zgodni!' : 'Tym razem inaczej';
  return { hit: m === 'skala' ? s.gain > 0 : s.hit, text };
}

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
function summary(s) {
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
  return [
    h('p', { className: 'dim' }, count || (s.cfg.mode === 'mix' ? 'Wspólny wynik' : 'Wasza zgodność')),
    h('div', { className: 'big' + (count ? ' half' : '') }, tak ? String(s.matches) : count ? `${s.matches} z ${s.rounds}` : percent(s) + '%'),
    !tak && s.found && h('p', { className: 'dim' }, 'Wasze wspólne TAK z tego wieczoru'),
    s.found && chips(s.found),
    s.scored && h('h2', {}, s.winner < 0 ? 'Remis!' : `Wygrywa ${s.players[s.winner].name}!`),
    s.scored && h('p', {}, score(s)),
    (s.stakes || []).map((t, i) => t && h('p', { className: 'stake', style: '--c:' + s.players[i].color }, `Nagroda dla: ${s.players[i].name} — ${t}`)),
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

// Drawings are strokes of x,y,x,y… on a 1000x1000 canvas, on the phone and on the projector alike.
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
function paint(c, strokes) {
  c.clearRect(0, 0, 1000, 1000);
  for (const st of strokes) { // a single point is drawn as a dot
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
