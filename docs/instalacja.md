# 🚀 Instalacja i utrzymanie

[← README](../.github/README.md)

## Start
Wystarczy plik [`docker-compose.yml`](../docker-compose.yml) z tego repozytorium:

```bash
docker compose pull
docker compose up -d
```

Gra słucha na porcie `8080`. Otwórz `http://<adres-ip>:8080/screen`.

> [!WARNING]
> Gra nie ma logowania. Trzymaj ją wyłącznie w sieci domowej i nie wystawiaj do internetu.
> Kod w adresie z QR (`/play?k=…`) chroni tylko przed przypadkowym zajęciem miejsca: każdy, kto otworzy `/screen` w twojej sieci, go zobaczy. To nie jest hasło.
> Gdyby gra kiedyś musiała być dostępna z zewnątrz, ustaw w Nginx Proxy Manager **Access List** (login i hasło) dla tego hosta.

## Bez Dockera: plik z wydania
Każde wydanie ma gotowe binarki dla Linuksa (`amd64` i `arm64` dla Raspberry Pi), sumy `SHA256SUMS` i podpisaną atestację pochodzenia. Sprawdź podpis przed uruchomieniem:

```bash
gh release download -R Huge-Commits-Only/duet -p 'duet_linux_arm64'
gh attestation verify duet_linux_arm64 -R Huge-Commits-Only/duet
chmod +x duet_linux_arm64 && DATA_DIR=$HOME/duet-data ./duet_linux_arm64   # http://<adres>:8080/screen
```

## Przeglądarki
Projektor: Chrome, Edge albo Safari (Firefox od wersji 126, bo starszy nie pomniejsza widoku, gdy się nie mieści). Jeśli komputer przy projektorze ma Linuksa, doinstaluj czcionkę emoji (`sudo apt install fonts-noto-color-emoji`), inaczej 🔇, 📱 i 👀 wyjdą jako puste prostokąty. Telefon: iOS 15 lub nowszy, Android z aktualnym Chrome.

Na iPhonie stronę pilota warto dodać do ekranu głównego (Udostępnij → Do ekranu początkowego): ma własną ikonę, otwiera się bez paska przeglądarki, a przy rysowaniu przeciągnięcie palcem od krawędzi nie cofa strony (w zwykłym Safari cofa i nie da się tego wyłączyć).

## Wersje: aktualizacja i powrót
Każdy merge do `main` publikuje obraz z trzema tagami: `latest`, numerem wersji (np. `1.4.2`) i skrótem commita.

```bash
docker compose pull && docker compose up -d          # najnowsza wersja
DUET_VERSION=1.4.2 docker compose up -d              # konkretna wersja, także powrót do poprzedniej
```

Dostępne wersje: `https://github.com/orgs/Huge-Commits-Only/packages/container/package/duet`. Dane w wolumenie zostają przy zmianie wersji.

**Zalecane: przypnij wersję na stałe.** Wpisz ją do pliku `.env` obok `docker-compose.yml` (compose czyta go sam), żeby nowy obraz nie wszedł niespodziewanie w środku wieczoru. Aktualizujesz świadomie, zmieniając numer:

```bash
echo "DUET_VERSION=1.4.2" > .env
docker compose pull && docker compose up -d
```

## Weryfikacja obrazu
Każdy obraz z `main` ma podpisaną atestację pochodzenia: z jakiego commita i workflowu powstał. Sprawdzisz ją przed wdrożeniem:
```bash
gh attestation verify oci://ghcr.io/huge-commits-only/duet:latest -R Huge-Commits-Only/duet
```

## Kopia zapasowa
W menu telefonu, pod przyciskiem „Więcej", są dwa przyciski: **Pobierz kopię zapasową** (jeden plik JSON z historią gier, listą wspólnych TAK, listą zadanych pytań, pytaniami wyłączonymi przyciskiem „Nie pokazuj więcej” rundami profilu pary i listami z kapsuły czasu) oraz **Wczytaj kopię zapasową** (zastępuje zapisane dane zawartością pliku). Przydaje się przed przenosinami na inny serwer.

**Kopia sama, raz w tygodniu.** Jedyne niezastępowalne dane to lista wspólnych TAK, historia gier, profil pary i listy z kapsuły czasu w wolumenie. Na Pi dopisz do `crontab -e` (nazwę wolumenu sprawdzisz przez `docker volume ls | grep duet`):

```cron
# niedziela 4:00: spakuj wolumen Duet, trzymaj kopie z ostatnich 3 miesięcy
0 4 * * 0 docker run --rm -v duet_duet-data:/data:ro -v $HOME/duet-kopie:/kopie busybox tar czf /kopie/duet-$(date +\%F).tgz -C /data . && find $HOME/duet-kopie -name 'duet-*.tgz' -mtime +90 -delete
```

Przywrócenie: `docker compose stop duet`, potem `docker run --rm -v duet_duet-data:/data -v $HOME/duet-kopie:/kopie busybox tar xzf /kopie/duet-RRRR-MM-DD.tgz -C /data` i `docker compose start duet`.

## Parametry i dane
| Zmienna | Domyślnie | Opis |
|---|---|---|
| `PORT` | `8080` | Port serwera w kontenerze |
| `PUBLIC_URL` | *(brak)* | Adres wpisywany w kod QR. Ustaw tylko wtedy, gdy QR pokazuje zły adres; normalnie gra bierze go z adresu, pod którym otwarto ekran |

Dane (lista zadanych pytań, Wasza lista wspólnych TAK, historia gier, własne paczki pytań, stan trwającej gry) leżą w wolumenie `duet-data` pod `/data`. Restart kontenera nie przerywa gry, bo telefony same do niej wracają.

## Nginx Proxy Manager
1. **Hosts → Proxy Hosts → Add Proxy Host**.
2. *Domain Names*: np. `duet.twoja-domena.duckdns.org`, *Scheme*: `http`, *Forward Hostname / IP*: adres Raspberry Pi, *Forward Port*: `8080`.
3. Zakładka **SSL**: wybierz certyfikat i włącz *Force SSL*.

Nic więcej nie trzeba ustawiać: gra sama wyłącza buforowanie dla połączenia na żywo. HTTPS jest potrzebny, żeby telefon nie wygaszał ekranu w trakcie gry. To samo dotyczy projektora: otwieraj go przez domenę (`https://…/screen`), nie przez `http://ip:8080`, bo inaczej komputer przy projektorze może wygasić ekran np. w trakcie Głębokich rozmów. Gdyby i tak gasł, wyłącz wygaszanie ekranu w systemie tego komputera.
