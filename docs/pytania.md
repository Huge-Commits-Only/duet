# ✏️ Własne pytania

[← README](../.github/README.md)

Wrzuć plik JSON do `/data/questions/` w wolumenie — gra wczyta go przy następnym starcie rundy, bez restartu:

```json
{
  "mode": "wolisz",
  "items": [
    { "tone": "lekko", "a": "Kawa", "b": "Herbata" }
  ]
}
```

```bash
docker cp moje-pytania.json duet:/data/questions/moje-pytania.json
```

`tone` to `lekko`, `romantycznie` albo `pikantnie`. Formy zależne od płci zapisuj tak jak w gotowych paczkach („zrobiłeś/aś", „chciał(a)byś", „pewny/a") — gra sama wybierze właściwą. Pola zależą od trybu (`mode`):

| Tryb | `mode` | Pola pytania |
|---|---|---|
| Co wolisz | `wolisz` | `a`, `b` — dwie możliwości |
| Kto prędzej… | `kto` | `q` — pytanie; odpowiedziami są Wasze imiona |
| Jak dobrze mnie znasz | `znasz` | `q` + `options` (2–4 odpowiedzi) |
| Skala | `skala` | `q`; opcjonalnie `a` i `b` jako opisy końców skali |
| Telepatia | `telepatia` | `q` — słowo, do którego wpisujecie skojarzenie |
| Wspólne TAK | `tak` | `q` — pomysł, na który każde mówi tak albo nie |
| Głębokie rozmowy | `rozmowy` | `q` + `level` (1–3, głębia pytania) |
| Wyzwania | `wyzwania` | `q` + `kind`: `prawda` (pytanie) albo `wyzwanie` (zadanie) |
| Blef | `blef` | `q` — temat, na który piszecie dwie prawdy i kłamstwo |
| Rysowanie | `rysowanie` | `q` — hasło do narysowania (jedno słowo, bo partner wpisuje je z klawiatury); opcjonalnie `alt` — lista słów też uznawanych za trafienie |

„Wieczór mix" nie ma własnych pytań — losuje rundy z pozostałych trybów.
