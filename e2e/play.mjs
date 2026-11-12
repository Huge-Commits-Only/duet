// Browser test: a projector and two phones play every mode from the menu to the final score.
// It answers whatever the phones ask for, so new modes built from the existing inputs need no changes here.
import { chromium, devices } from 'playwright';

const base = process.argv[2];
const browser = await chromium.launch();
const errors = [];
async function open(options, path, refused = false) {
  const page = await (await browser.newContext(options)).newPage();
  page.on('pageerror', e => errors.push(`${path}: ${e.message}`));
  // a refused request is logged by the browser as an error; expected only where the test provokes one
  page.on('console', m => m.type() === 'error' && !refused && errors.push(`${path}: ${m.text()}`));
  page.on('dialog', d => d.accept());
  await page.goto(base + path);
  return page;
}
const check = (ok, what) => { if (!ok) throw new Error(what); };
// SHOTS=/some/dir saves a few screenshots for a look at the layout
const shot = (name, page) => process.env.SHOTS && page.screenshot({ path: `${process.env.SHOTS}/${name}.png` });
const has = async (page, selector) => (await page.locator(selector).count()) > 0;
const text = page => page.locator('#app').innerText();
// Resolves once the phone shows something else than `before`, i.e. the state update arrived.
const changed = (page, before) => page.waitForFunction(t => document.getElementById('app').innerText !== t, before);

const screen = await open({ viewport: { width: 1280, height: 720 } }, '/screen');

// ---- joining: only with the code from the QR link ----
const stranger = await open(devices['iPhone 14'], '/play', true);
await stranger.fill('#name', 'Obcy');
await stranger.click('#join .btn');
await stranger.getByText('zeskanuj kod QR').waitFor();
await stranger.close();

await screen.locator('#url').filter({ hasText: '/play?k=' }).waitFor();
const link = '/play?k=' + (await screen.locator('#url').textContent()).split('k=')[1];
const a = await open(devices['iPhone 14'], link);
const b = await open(devices['iPhone 14'], link);
const phones = [a, b];
await a.fill('#name', 'Ola');
await a.getByRole('button', { name: 'jak do niej' }).click();
await a.click('#join .btn');
await b.fill('#name', 'Michał');
await b.getByRole('button', { name: 'jak do niego' }).click();
await b.click('#join .btn');
await a.getByRole('button', { name: 'Start' }).waitFor();
check((await screen.locator('.slot.on').count()) === 2, 'both players must show as joined on the projector');

// ---- one step of a game: do whatever a phone is waiting for ----
// Returns false once the game is over. Odd rounds agree or hit, even rounds differ or miss.
async function step() {
  for (const p of phones) {
    // both players picked a form, so a phone never shows a double spelling like "zrobiłeś/aś"
    const double = (await text(p)).match(/\p{L}+\/aś|\p{L}+\(a\)\p{L}*|\p{L}+y\/a\b/u);
    check(!double, `unresolved form on a phone: ${double}`);
    const other = p === a ? b : a;
    if (await has(p, 'button:text-is("Menu")')) return false;
    const next = p.locator('button:text-is("Dalej"), button:text-is("Zobacz wynik"), button:text-is("Zakończ rozmowę")');
    if (await next.count()) {
      const before = await text(p);
      await next.first().click();
      await changed(p, before);
      return true;
    }
    if (await has(p, '.stmt')) { // blef: write three statements, the partner guesses
      const st = p.locator('input[name=st]');
      for (const [i, s] of ['Byłem w Japonii', 'Mam trzy koty', 'Boję się gęsi'].entries()) await st.nth(i).fill(s);
      await p.locator('input[name=lie]').nth(1).check();
      await p.getByRole('button', { name: 'Gotowe' }).click();
      await Promise.all([p.locator('.stmt').first().waitFor({ state: 'detached' }), other.locator('.pick').first().waitFor()]);
      check(!(await text(other)).includes('kłamstwo jest'), 'the guesser must not be told which statement is the lie');
      await other.locator('.pick').nth(1).click();
      return true;
    }
    if (await has(p, 'canvas.pad')) { // rysowanie: draw, undo, wrong guess, right guess
      const word = (await p.locator('#app h2').textContent()).trim();
      check(!(await text(screen)).includes(word) && !(await text(other)).includes(word), 'only the one drawing may see the word');
      const box = await p.locator('canvas.pad').boundingBox();
      const at = (x, y) => [box.x + box.width * x, box.y + box.height * y];
      for (const pts of [[[.2, .8], [.2, .4], [.8, .4], [.8, .8]], [[.1, .1], [.9, .9]]]) {
        await p.mouse.move(...at(...pts[0]));
        await p.mouse.down();
        for (const pt of pts.slice(1)) await p.mouse.move(...at(...pt), { steps: 6 });
        await p.mouse.up();
      }
      await p.getByRole('button', { name: 'Cofnij kreskę' }).click();
      await other.fill('#word', 'na pewno nie to');
      await other.getByRole('button', { name: 'Wyślij' }).click();
      await screen.getByText('na pewno nie to').waitFor();
      await other.fill('#word', word.toLowerCase());
      await other.getByRole('button', { name: 'Wyślij' }).click();
      await p.locator('canvas.pad').waitFor({ state: 'detached' });
      return true;
    }
    if (await has(p, 'button:text-is("Prawda")')) { // wyzwania: pick truth or dare...
      const before = await text(p);
      await p.getByRole('button', { name: 'Wyzwanie' }).click();
      await Promise.all([changed(p, before), other.getByRole('button', { name: 'Zaliczone' }).waitFor()]);
      return true;
    }
    if (await has(p, 'button:text("Zaliczone")')) { // ...then the partner judges
      const before = await text(p);
      await p.getByRole('button', { name: 'Zaliczone' }).click();
      await changed(p, before);
      return true;
    }
  }
  const before = await text(a);
  const round = (await a.locator('#app p').first().textContent().catch(() => '')) || '';
  const odd = Number((round.match(/\d+/) || [1])[0]) % 2 === 1;
  if (await has(a, '#word') && await has(b, '#word')) { // telepatia
    await a.fill('#word', 'Słońce');
    await a.getByRole('button', { name: 'Wyślij' }).click();
    await b.fill('#word', odd ? 'slonce' : 'deszcz');
    await b.getByRole('button', { name: 'Wyślij' }).click();
  } else if (await has(a, 'input[type=range]') && await has(b, 'input[type=range]')) { // skala
    await a.locator('input[type=range]').fill('60');
    await a.getByRole('button', { name: 'Zatwierdź' }).click();
    await b.locator('input[type=range]').fill(odd ? '62' : '10');
    await b.getByRole('button', { name: 'Zatwierdź' }).click();
  } else if (await has(a, '.pick') && await has(b, '.pick') && !(await has(a, '.pick[aria-pressed="true"]'))) { // any choice
    await a.locator('.pick').first().click();
    await b.locator('.pick').nth(odd ? 0 : 1).click();
  } else {
    await a.waitForTimeout(50);
    return true;
  }
  await changed(a, before);
  return true;
}

async function play(label, expectAtEnd) {
  const pick = a.locator('.picker button', { hasText: label });
  await pick.click();
  await a.locator('.picker button[aria-pressed="true"]', { hasText: label }).waitFor();
  check((await screen.locator('.rule').count()) === 1, `${label}: the menu must explain the rules`);
  await a.getByRole('button', { name: 'Start' }).click();
  await a.getByRole('button', { name: 'Start' }).waitFor({ state: 'detached' });
  check((await screen.locator('.rule').count()) === 1, `${label}: the first round must explain the rules`);
  for (let steps = 0; await step(); steps++) check(steps < 600, `${label}: stuck at\n${await text(a)}\n---\n${await text(b)}`);
  const end = (await text(screen)).replace(/\n+/g, ' | ');
  check(expectAtEnd.test(end), `${label}: unexpected final screen: ${end}`);
  console.log(`✔ ${label}: ${end}`);
  await a.getByRole('button', { name: 'Menu' }).click();
  await b.getByRole('button', { name: 'Start' }).waitFor();
}

// ---- a secret stake, kept while the partner changes settings ----
await a.locator('.picker button', { hasText: 'Skala' }).click();
await a.locator('#stake').waitFor();
await a.locator('#stake').pressSequentially('masaż');
await b.locator('.picker button', { hasText: 'Romantycznie' }).click();
await a.locator('.picker button[aria-pressed="true"]', { hasText: 'Romantycznie' }).waitFor();
await a.locator('#stake').pressSequentially(' stóp');
check((await a.locator('#stake').inputValue()) === 'masaż stóp', 'typing a stake must survive a state update');
await a.locator('#stake').blur();
await screen.getByText('✓ stawka wpisana').waitFor();
await b.fill('#stake', 'tydzień zmywania');
await b.locator('#stake').blur();
await screen.locator('.slot small', { hasText: '✓ stawka wpisana' }).nth(1).waitFor();
await b.locator('.picker button', { hasText: 'Lekko' }).click();

// ---- every mode ----
await play('Co wolisz', /Wasza zgodność \| 50%$/); // no points, so no reward either
await play('Kto prędzej', /Wasza zgodność \| 50%/);
await play('Jak dobrze mnie znasz', /Wygrywa|Remis/);
// Michał guesses in the odd rounds, which are the close ones: only the winner's reward is revealed
await play('Skala', /Wygrywa Michał! \| Ola 0 : 12 Michał \| Nagroda dla: Michał — tydzień zmywania$/);
await play('Telepatia', /Wasza zgodność \| 50%/);
await play('Wspólne TAK', /Wspólne TAK \| 4/);
await play('Głębokie rozmowy', /Koniec rozmowy/);
await play('Wyzwania', /Zaliczone zadania \| 6 z 6/);
await play('Blef', /Wykryte kłamstwa \| 4 z 4/);
await play('Rysowanie', /Odgadnięte hasła \| 4 z 4/);
await play('Wieczór mix', /Wspólny wynik/);

// ---- the saved list of mutual yeses: show, tick off, remove ----
await a.getByRole('button', { name: /Pokaż na ekranie naszą listę/ }).click();
await screen.getByText('Wasze wspólne TAK').waitFor();
const saved = await screen.locator('.list span').count();
check(saved >= 4, 'mutual yeses must be saved between games');
await b.locator('.yes button[aria-label="Oznacz jako zrobione"]').first().click();
await screen.locator('.list .done').waitFor();
await b.locator('.yes button[aria-label="Usuń z listy"]').first().click();
await screen.locator('.list span').nth(saved - 1).waitFor({ state: 'detached' });
await a.getByRole('button', { name: 'Ukryj naszą listę' }).click();

// ---- skipping a round and coming back after a reload ----
await a.locator('.picker button', { hasText: 'Co wolisz' }).click();
await a.locator('.picker button[aria-pressed="true"]', { hasText: 'Co wolisz' }).waitFor();
await a.getByRole('button', { name: 'Start' }).click();
await a.getByText('Runda 1 z 8').waitFor();
await b.getByRole('button', { name: 'Pomiń rundę' }).click();
await a.getByText('Runda 2 z 8').waitFor();
await a.reload();
await a.getByText('Runda 2 z 8').waitFor(); // straight back in, no name form
await a.getByRole('button', { name: 'Zakończ grę' }).click();
await b.getByRole('button', { name: 'Start' }).waitFor();

// ---- statistics across the games just played ----
await a.getByRole('button', { name: 'Pokaż statystyki na ekranie' }).click();
await screen.getByText('Wasze statystyki').waitFor();
check(Number(await screen.locator('.tile b').nth(1).textContent()) === 11, 'every finished game must be counted, the abandoned one not');
check((await screen.locator('.chart polyline').count()) === 1, 'compatibility over time must be drawn');
await shot('stats', screen);
await a.getByRole('button', { name: 'Ukryj statystyki' }).click();
await screen.getByText('Ustawcie grę').waitFor();

// ---- a second couple sits down: everyone answers at once, couples are ranked ----
const c = await open(devices['iPhone 14'], link);
const d = await open(devices['iPhone 14'], link);
for (const [page, name] of [[c, 'Kasia'], [d, 'Tomek']]) {
  await page.fill('#name', name);
  await page.locator('#pairs button', { hasText: '2' }).click();
  await page.click('#join .btn');
}
await d.getByRole('button', { name: 'Start' }).waitFor();
await screen.getByText('Para 2').waitFor();
await shot('lobby-two-couples', screen);
check(await a.locator('.picker button', { hasText: 'Rysowanie' }).isDisabled(), 'one-couple modes must be unavailable with two couples');
check((await a.locator('#stakeBox:visible').count()) === 0, 'rewards are for a single couple');
await c.locator('.picker button', { hasText: 'Co wolisz' }).click();
await c.getByRole('button', { name: 'Start' }).click();
for (let i = 1; i <= 8; i++) {
  for (const page of [a, b, c, d]) await page.getByText(`Runda ${i} z 8`).waitFor();
  await a.locator('.pick').first().click(); // Ola and Michał always agree,
  await b.locator('.pick').first().click();
  await c.locator('.pick').first().click(); // Kasia and Tomek never do
  if (i === 1) {
    check((await text(screen)).includes('… Kasia i Tomek'), 'the projector must show which couple is still answering');
    check((await text(a)).includes('Czekamy na pozostałe pary'), 'a couple that is done waits for the others');
  }
  await d.locator('.pick').nth(1).click();
  await screen.locator('.ranking').waitFor();
  if (i === 1) await shot('round-ranking', screen);
  await d.getByRole('button', { name: i === 8 ? 'Zobacz wynik' : 'Dalej' }).click();
}
await screen.getByText('Najbardziej zgrana para').waitFor();
const final = (await text(screen)).replace(/\s+/g, ' ');
check(/Ola i Michał 1\. Ola i Michał 8 pkt 2\. Kasia i Tomek 0 pkt/.test(final), `unexpected ranking: ${final}`);
await shot('final-ranking', screen);
await shot('final-phone', c);
console.log(`✔ dwie pary: ${final}`);

await browser.close();
if (errors.length) {
  console.error('Browser errors:\n' + errors.join('\n'));
  process.exit(1);
}
console.log('✔ no browser errors');
