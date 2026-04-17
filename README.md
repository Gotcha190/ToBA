<div align="center">
  <pre>
░▒▓████████▓▒░▒▓██████▓▒░░▒▓███████▓▒░ ░▒▓██████▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓███████▓▒░░▒▓████████▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░  ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░   ░▒▓██████▓▒░░▒▓███████▓▒░░▒▓█▓▒░░▒▓█▓▒░
  </pre>
  <h1>Tamago Build Application</h1>
  <p>CLI do tworzenia lokalnych środowisk WordPress na bazie Lando.</p>
</div>

## ToBA

ToBA to narzędzie CLI napisane w Go, które automatyzuje przygotowanie lokalnego projektu WordPress z użyciem Lando, WP-CLI oraz danych startowych pobieranych z lokalnego backupu lub przez SSH.

## Co robi ToBA

ToBA udostępnia cztery komendy:

- `toba config` tworzy lub odświeża globalny plik konfiguracyjny w `~/.config/toba/.env`.
- `toba create` tworzy lokalny projekt WordPress, przywraca dane startowe i finalizuje środowisko.
- `toba doctor` sprawdza, czy wszystkie wymagane narzędzia są dostępne w systemie.
- `toba version` wypisuje aktualny numer wersji CLI.

Podczas `toba create` narzędzie działa w jednym z dwóch trybów:

- `local backup mode`: używa istniejącego folderu `./<project-name>` z kompletem plików Updraft.
- `SSH fallback mode`: pobiera starter database, plugins i uploads przez SSH, jeśli lokalny folder projektu nie istnieje.

## Wymagania

Do pełnego działania potrzebne są:

- `git`
- `node`
- `npm`
- `lando`
- `docker`
- `ssh`
- `scp`
- `zip`
- `Go 1.26.1+` do budowania z aktualnego repozytorium

## Instalacja

Instalacja przez Go:

```bash
go install github.com/gotcha190/toba@latest
```

Lokalny build z repozytorium:

```bash
git clone git@github.com:Gotcha190/ToBA.git
cd ToBA
go install .
```

## Szybki start

1. Zainicjalizuj globalny config:

```bash
toba config
```

2. Uzupełnij wartości w:

```bash
~/.config/toba/.env
```

3. Zweryfikuj zależności:

```bash
toba doctor
```

4. Utwórz projekt:

```bash
toba create demo
```

## Konfiguracja

ToBA korzysta ze wspólnego pliku:

```bash
~/.config/toba/.env
```

Obsługiwane pola:

```bash
TOBA_PHP_VERSION=
TOBA_STARTER_REPO=
TOBA_SSH_TARGET=
```

Kolejność rozwiązywania konfiguracji:

1. flagi CLI,
2. `~/.config/toba/.env`,
3. wartości domyślne.

Ważne domyślne zachowania:

- PHP domyślnie: `8.4`
- baza danych: `wordpress`
- lokalna domena: `<project-name>.lndo.site`

Format `TOBA_SSH_TARGET`:

```bash
user@host -p port
```

## Komendy

```bash
toba config
toba create [project-name] [--php=8.4] [--starter-repo=git@github.com:org/repo.git] [--ssh-target='user@host -p port'] [--dry-run]
toba doctor
toba version
```

## Jak działa create

`toba create` wykonuje aktualnie taki przepływ:

1. normalizuje konfigurację projektu,
2. wybiera źródło danych startowych,
3. tworzy albo przygotowuje katalog projektu,
4. generuje `.lando.yml`, `config/php.ini` i `wp-cli.yml`,
5. uruchamia `lando start`,
6. instaluje WordPress,
7. instaluje lub przywraca theme,
8. przywraca plugins, uploads, opcjonalne others i database,
9. wykonuje `search-replace` na lokalną domenę,
10. resetuje hasło administratora,
11. aktywuje theme,
12. czyści cache i odświeża rewrite rules.

## Tryby danych startowych

### Local backup mode

Jeśli istnieje folder `./<project-name>`, ToBA skanuje go w poszukiwaniu rozpoznawalnych plików Updraft. Wymagany jest kompletny zestaw:

- dokładnie jeden backup bazy danych,
- co najmniej jedno archiwum `plugins`,
- co najmniej jedno archiwum `uploads`,
- co najmniej jedno archiwum `themes`,
- opcjonalne archiwa `others`.

Jeśli folder istnieje, ale jest pusty albo niekompletny, `toba create` zakończy się błędem.

### SSH mode

Jeśli `./<project-name>` nie istnieje, ToBA używa `TOBA_SSH_TARGET` albo flagi `--ssh-target` i pobiera starter data z hosta zdalnego.

Zasady:

- SSH nie pobiera `themes`,
- SSH nie pobiera `others`,

## Przykłady

Utworzenie projektu z lokalnego backupu plików Updrafta:

```bash
mkdir demo
cp backup-demo-db.sql demo/
cp backup-demo-plugins.zip demo/
cp backup-demo-uploads.zip demo/
cp backup-demo-themes.zip demo/
toba create demo
```

Utworzenie projektu przez SSH:

```bash
toba config
toba create demo --starter-repo=git@github.com:org/repo.git --ssh-target='user@host -p 22'
```

Suchy przebieg bez zapisu plików:

```bash
toba create demo --dry-run --starter-repo=git@github.com:org/repo.git --ssh-target='user@host -p 22'
```

## Struktura repozytorium

```text
.
├── cmd/
├── internal/
│   ├── cli/
│   ├── create/
│   │   └── steps/
│   ├── doctor/
│   ├── lando/
│   ├── project/
│   ├── templates/
│   │   └── files/
│   ├── theme/
│   ├── updraft/
│   └── wordpress/
├── .env.example
├── LICENSE
├── README.md
├── go.mod
└── main.go
```

## Licencja

Projekt jest udostępniony na licencji MIT. Możesz go dowolnie używać, kopiować, modyfikować i rozpowszechniać, ale musisz zachować treść licencji oraz informację o autorze.

Pełna treść znajduje się w pliku [LICENSE](LICENSE).

## Autor

Repozytorium zostało stworzone przez **Konrada Formella (gotcha190)**.
