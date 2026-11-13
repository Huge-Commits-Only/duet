# 💞 Duet
Gry i quizy dla par. Projektor (albo telewizor) pokazuje grę, a telefony służą za piloty — dołączacie, skanując kod QR. Gracie we dwoje albo z gośćmi: do czterech par przy jednym ekranie. Całość działa lokalnie w homelabie, bez kont i bez chmury.


<br>


## 🌟 Jak to działa
1. Na projektorze otwierasz `https://twoja-domena/screen` — pojawia się kod QR.
2. Oboje skanujecie kod telefonami, wpisujecie imię i wybieracie, jak się do Was zwracać („byłeś" czy „byłaś") — telefon to zapamięta. Dołączyć może tylko ktoś, kto ma adres z kodu QR.
3. Na telefonie wybieracie grę, ton (lekko / romantycznie / pikantnie 18+) i długość, a potem **Start**.
4. Odpowiadacie w tajemnicy na telefonach; projektor odsłania odpowiedzi i liczy zgodność.

Menu na telefonie ma na wierzchu tylko wybór gry, tonu i długości oraz przycisk Start; lista wspólnych TAK, statystyki, boty, zmiana imienia, gracze i kopia zapasowa są pod „Więcej". Ekran projektora dopasowuje się do każdej rozdzielczości i proporcji (16:9, 16:10, 4:3) — gdy treść się nie mieści, sam ją pomniejsza.

Rozmiar tekstu na projektorze zmienisz klawiszami `+` i `-` (`0` przywraca domyślny), a `m` wycisza dźwięk.

Dźwięk gra z urządzenia podłączonego do projektora. Dźwięk jest zawsze włączony na starcie, ale przeglądarka puści go dopiero po pierwszym kliknięciu lub klawiszu na ekranie projektora. Wyciszasz ikoną głośnika w prawym górnym rogu albo klawiszem `m`.

Projektor: Chrome, Edge albo Safari (Firefox od wersji 126 — starszy nie pomniejsza widoku, gdy się nie mieści). Jeśli komputer przy projektorze ma Linuksa, doinstaluj czcionkę emoji (`sudo apt install fonts-noto-color-emoji`) — inaczej 🔇, 📱 i 👀 wyjdą jako puste prostokąty. Telefon: iOS 15 lub nowszy, Android z aktualnym Chrome.

### Tryby
| Tryb | O co chodzi |
|---|---|
| Co wolisz | Oboje wybieracie jedną z dwóch opcji — liczy się zgodność |
| Kto prędzej… | Oboje wskazujecie, kogo z Was dotyczy pytanie |
| Nigdy przenigdy | Oboje w tajemnicy: zdarzyło mi się czy nigdy — ekran odsłania obie odpowiedzi |
| Jak dobrze mnie znasz | Jedno odpowiada o sobie, drugie zgaduje |
| Skala | Jedno ustawia suwak 0–100, drugie zgaduje jak najbliżej |
| Szacowanie | Jedno podaje liczbę o sobie (np. ile razy w tygodniu…), drugie zgaduje jak najbliżej |
| Telepatia | Oboje wpisujecie skojarzenie — punkt za to samo słowo |
| Wspólne TAK | Tak/nie na pomysły; ekran pokazuje tylko to, na co oboje się zgodziliście, i zapisuje listę |
| Głębokie rozmowy | Karty do rozmowy na trzech poziomach głębi, bez punktów |
| Wyzwania | Prawda albo zadanie dla jednej osoby, druga ocenia wykonanie |
| Blef | Dwie prawdy i jedno kłamstwo — druga osoba szuka kłamstwa |
| Rysowanie | Jedno rysuje palcem na telefonie, drugie zgaduje hasło w 60 sekund |
| Wieczór mix | Gra sama przeplata tryby i liczy wspólny wynik |

### Kilka par
Przy dołączaniu każdy wybiera numer pary (partnerzy ten sam). Gdy siedzą co najmniej dwie pary, ekran pokazuje ranking — wygrywa najbardziej zgrana para.

- **Wszyscy naraz:** Co wolisz, Kto prędzej…, Nigdy przenigdy, Jak dobrze mnie znasz, Skala, Szacowanie, Telepatia, Wspólne TAK. Każda para odpowiada między sobą na to samo pytanie.
- **Po kolei:** Wyzwania, Blef, Rysowanie. Jedna para gra, reszta patrzy na ekran; kolejka idzie wokół stołu, a gra jest odpowiednio dłuższa, żeby każdy miał tyle samo rund.
- **Wieczór mix** miesza jedne i drugie. **Głębokie rozmowy** są tylko dla jednej pary.

Pod przyciskiem „Więcej" na dole menu telefonu widać, kto siedzi w której parze, i można usunąć gracza, który zajął miejsce przez pomyłkę.

**Para 1 to domownicy.** Tylko ona widzi zapisaną listę wspólnych TAK, statystyki i kopię zapasową, i tylko wtedy, gdy nie siedzi nikt inny. Goście wybierają pary 2–4.

### Test w pojedynkę
Bez drugiej osoby też da się wszystko sprawdzić. Gdy siedzisz w parze sam, telefon pokazuje przycisk „Zagraj z botem" — bot zajmuje miejsce partnera i odpowiada losowo po krótkiej chwili. Z menu można jeszcze dosadzić parę botów, żeby zobaczyć ranking kilku par. Gry z botem nie liczą się do statystyk, a prawdziwy partner, który dołączy do Twojej pary, po prostu zastępuje bota.

### Statystyki
Każda dokończona gra trafia do historii. Przycisk „Pokaż statystyki na ekranie" w menu telefonu wyświetla na projektorze liczbę wieczorów i gier, średnią zgodność, wykres zgodności w ostatnich grach, zgodność w poszczególnych trybach, bilans zwycięstw i ostatnie wyniki.

### Stawka
Gdy gra jedna para, w grach z punktami każde z Was może przed startem wpisać tajną stawkę — co przegrany robi dla wygranego. Ekran odsłania stawkę zwycięzcy na końcu (przy remisie obie).


<br>


## 💻 Technologie
| Technologia | Rola | Szczegóły |
|---|---|---|
| 🎯 **Go** | Serwer | Jeden plik binarny, frontend wbudowany, ok. 25 MB RAM |
| 🌐 **HTML / CSS / JS** | Ekran i pilot | Bez frameworka i bez kroku budowania, zero zasobów z internetu |
| 🐳 **Docker** | Kontener | Obraz `scratch`, multi-arch (amd64 / arm64 dla Raspberry Pi) |
| 🔄 **GitHub Actions** | CI/CD | Testy, lint, skany bezpieczeństwa, publikacja obrazu |
| 📦 **GHCR** | Rejestr | `ghcr.io/huge-commits-only/duet` |


<br>


## 📁 Struktura projektu
```
duet/
├── cmd/duet/            # punkt startowy programu
├── internal/
│   ├── game/            # gra: stan, tryby, punktacja, zapis (bez HTTP)
│   └── server/          # HTTP: strony, strumień zdarzeń, API dla telefonów
├── questions/           # paczki pytań, jeden plik JSON na tryb
├── web/                 # ekran projektora i pilot na telefon
├── e2e/                 # test w przeglądarce (Playwright)
├── .github/             # workflowy CI/CD i to README
├── Dockerfile
└── docker-compose.yml
```


<br>


## 🚀 Uruchomienie w homelabie
Wystarczy plik [`docker-compose.yml`](../docker-compose.yml) z tego repozytorium:

```bash
docker compose pull
docker compose up -d
```

Gra słucha na porcie `8080`. Otwórz `http://<adres-ip>:8080/screen`.

### Wersje: aktualizacja i powrót
Każdy merge do `main` publikuje obraz z trzema tagami: `latest`, numerem wersji (np. `1.4.2`) i skrótem commita.

```bash
docker compose pull && docker compose up -d          # najnowsza wersja
DUET_VERSION=1.4.2 docker compose up -d              # konkretna wersja — także powrót do poprzedniej
```

Dostępne wersje: `https://github.com/orgs/Huge-Commits-Only/packages/container/package/duet`. Dane w wolumenie zostają przy zmianie wersji.

**Zalecane: przypnij wersję na stałe.** Wpisz ją do pliku `.env` obok `docker-compose.yml` (compose czyta go sam), żeby nowy obraz nie wszedł niespodziewanie w środku wieczoru. Aktualizujesz świadomie, zmieniając numer:

```bash
echo "DUET_VERSION=1.4.2" > .env
docker compose pull && docker compose up -d
```

### Pytania, których nie chcecie
Pod każdym pytaniem na telefonie jest mały link **Nie pokazuj więcej**: pytanie znika ze wszystkich następnych gier (zapisuje się w `banned.json` w wolumenie). Działa tylko dla pary 1, czyli domowników.

### Kopia zapasowa
W menu telefonu, pod przyciskiem „Więcej", są dwa przyciski: **Pobierz kopię zapasową** (jeden plik JSON z historią gier, listą wspólnych TAK, listą zadanych pytań i pytaniami wyłączonymi przyciskiem „Nie pokazuj więcej”) oraz **Wczytaj kopię zapasową** (zastępuje zapisane dane zawartością pliku). Przydaje się przed przenosinami na inny serwer.

**Kopia sama, raz w tygodniu.** Jedyne niezastępowalne dane to lista wspólnych TAK i historia gier w wolumenie. Na Pi dopisz do `crontab -e` (nazwę wolumenu sprawdzisz przez `docker volume ls | grep duet`):

```cron
# niedziela 4:00: spakuj wolumen Duet, trzymaj kopie z ostatnich 3 miesięcy
0 4 * * 0 docker run --rm -v duet_duet-data:/data:ro -v $HOME/duet-kopie:/kopie busybox tar czf /kopie/duet-$(date +\%F).tgz -C /data . && find $HOME/duet-kopie -name 'duet-*.tgz' -mtime +90 -delete
```

Przywrócenie: `docker compose stop duet`, potem `docker run --rm -v duet_duet-data:/data -v $HOME/duet-kopie:/kopie busybox tar xzf /kopie/duet-RRRR-MM-DD.tgz -C /data` i `docker compose start duet`.

### Parametry
| Zmienna | Domyślnie | Opis |
|---|---|---|
| `PORT` | `8080` | Port serwera w kontenerze |
| `PUBLIC_URL` | *(brak)* | Adres wpisywany w kod QR. Ustaw tylko wtedy, gdy QR pokazuje zły adres — normalnie gra bierze go z adresu, pod którym otwarto ekran |

Dane (lista zadanych pytań, Wasza lista wspólnych TAK, historia gier, własne paczki pytań, stan trwającej gry) leżą w wolumenie `duet-data` pod `/data`. Restart kontenera nie przerywa gry — telefony same do niej wracają.

Na iPhonie stronę pilota warto dodać do ekranu głównego (Udostępnij → Do ekranu początkowego): ma własną ikonę, otwiera się bez paska przeglądarki, a przy rysowaniu przeciągnięcie palcem od krawędzi nie cofa strony (w zwykłym Safari cofa i nie da się tego wyłączyć).

### Nginx Proxy Manager
1. **Hosts → Proxy Hosts → Add Proxy Host**.
2. *Domain Names*: np. `duet.twoja-domena.duckdns.org`, *Scheme*: `http`, *Forward Hostname / IP*: adres Raspberry Pi, *Forward Port*: `8080`.
3. Zakładka **SSL**: wybierz certyfikat i włącz *Force SSL*.

Nic więcej nie trzeba ustawiać — gra sama wyłącza buforowanie dla połączenia na żywo. HTTPS jest potrzebny, żeby telefon nie wygaszał ekranu w trakcie gry. To samo dotyczy projektora: otwieraj go przez domenę (`https://…/screen`), nie przez `http://ip:8080` — inaczej komputer przy projektorze może wygasić ekran np. w trakcie Głębokich rozmów. Gdyby i tak gasł, wyłącz wygaszanie ekranu w systemie tego komputera.

> [!WARNING]
> Gra nie ma logowania. Trzymaj ją wyłącznie w sieci domowej i nie wystawiaj do internetu.
> Kod w adresie z QR (`/play?k=…`) chroni tylko przed przypadkowym zajęciem miejsca — każdy, kto otworzy `/screen` w twojej sieci, go zobaczy. To nie jest hasło.
> Gdyby gra kiedyś musiała być dostępna z zewnątrz, ustaw w Nginx Proxy Manager **Access List** (login i hasło) dla tego hosta.


<br>


## ✏️ Własne pytania
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


<br>


## 🔄 CI/CD
| Workflow | Plik | Kiedy | Co robi |
|---|---|---|---|
| Go Verification | `go-test.yml` | PR, `main` | `go mod verify`, `go mod tidy -diff`, `gofmt`, kompilacja |
| Unit Test Verification | `unit-test.yml` | PR, `main` | `go test -race`, podsumowanie pokrycia i próg minimalny (85%) |
| Lint Verification | `lint-test.yml` | PR, `main` | `golangci-lint` |
| Security Verification | `security-test.yml` | PR, `main` | `govulncheck` + Trivy (podatności, sekrety, konfiguracja) |
| Container Verification | `container-test.yml` | PR | Buduje obraz, sprawdza czy kontener startuje, skanuje go Trivy |
| E2E Verification | `e2e-test.yml` | PR, `main` | Playwright: projektor i dwa telefony rozgrywają każdy tryb w prawdziwej przeglądarce |
| Workflow Verification | `workflow-test.yml` | PR, `main` | `actionlint` dla workflowów |
| PR Automation | `pr-automation.yml` | nowy PR | Przypisuje autora i reviewera, nadaje etykiety z nazwy brancha, ustawia milestone |
| Release Draft | `release-draft.yml` | `main` | Aktualizuje szkic notatek wydania |
| Docker Image Publish | `docker-build.yml` | `main` | Buduje obraz amd64 + arm64 i publikuje do GHCR (`latest` + wersja) |

### Po pierwszym merge'u (jednorazowo)
- **Paczka publiczna** — patrz sekcja wyżej.
- **Pierwszy release** — Release Drafter zbiera PR-y od ostatniego *opublikowanego* wydania. Opublikuj pierwsze ręcznie (**Releases → Draft a new release → Generate release notes**) i go nie usuwaj; inaczej szkic będzie pusty.
- **Ruleset na `main`** — wymagane checki to nazwy jobów (`Build & Modules`, `Go Tests`, `golangci-lint`, `govulncheck & Trivy`, `Trivy Image Scan`, `Playwright Game Run`, `Actionlint`). Pojawiają się do wyboru dopiero po pierwszym uruchomieniu.

### Etykiety z nazwy brancha
`feat/*`, `feature/*` → `enhancement` · `fix/*`, `bug/*` → `bug` · `docs/*` → `documentation` · `chore/*` → `chore` · `deps/*` → `dependencies`


<br>


## 🛠️ Praca lokalna
```bash
go vet ./... && go test ./...
DATA_DIR=./data PORT=18080 go run ./cmd/duet   # http://localhost:18080/screen

# test w przeglądarce: wszystkie tryby, projektor + dwa telefony
(cd e2e && npm ci && npx playwright install chromium)
go build -o bin/duet ./cmd/duet && ./e2e/run.sh
```
