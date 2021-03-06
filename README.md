# Devoir à distance 3A Web du 05/04/20

## Rendu

Réalisé en environ 16 heures reparties sur 2 jours.

```
$ cat $(pwd)/fixtures/0/1/3/guide.txt

Ideally, you should be able to just type:

./configure
make
make install


For Kali Linux, you might also need to do:

apt install libssl1.0-dev

It needs an older SSL version due to OpenSSH code for the SSH module.


For Ubuntu, you might also need to add the -lrt flag to the LDFLAGS variable in the Makefile,
if you get the undefined reference to `clock_gettime' error:

LDFLAGS = -Lnbase -Lnsock/src -Lopensshlib $(DBGFLAGS) $(STATIC) -lrt

$ search index $(pwd)/fixtures
Walking /Users/user907865/www/go-search/fixtures...
Successfully indexed file /Users/user907865/www/go-search/fixtures/0/1/3/guide.txt
Successfully indexed file /Users/user907865/www/go-search/fixtures/0/1/3/guide.2.txt

$ search query make
Querying index for "make":

                           File|   Count|   First match|
       fixtures/0/1/3/guide.txt|       4|          make|
     fixtures/0/1/3/guide.2.txt|       2|          make|

$ search dump
add:
  fixtures/0/1/3/guide.2.txt: 2
  fixtures/0/1/3/guide.txt: 1
also:
  fixtures/0/1/3/guide.2.txt: 4
  fixtures/0/1/3/guide.txt: 2
...
```

## Présentation

Le but du devoir est de construire un moteur de recherche en texte intégral
_très_ basique pour rechercher du contenu sur disque, en stockant ses données
dans Redis.

Les tâches qui le composent sont les suivantes:

- [x] créer un programme en ligne de commande acceptant une commande d'indexation,
   une de déboguage, et une de recherche
- [x] créer une fonction pouvant enregistrer une valeur dans un "sorted set" Redis
- [x] créer une fonction pouvant extraire un résultat d'un "sorted set" Redis
- [x] créer un test d'intégration combinant les deux précédentes
- [x] créer une fonction lisant un fichier texte mot par mot
- [x] créer une fonction enregistrant les statistiques de mots d'un fichier au moyen des fonctions de 2. et 5.
- [x] créer une fonction parcourant une sous-arborescence disque pour en examiner tous les fichiers
- [x] créer une fonction identifiant si un fichier est reconnu par Go comme un fichier texte
- [x] intégrer les deux précédentes pour que le parcours ne traite que les fichiers reconnus comme texte
- [x] intégrer la fonction avec l'écriture des comptes de mots dans Redis
- [x] écrire une fonction implémentant la commande de déboguage, qui liste tout le contenu de la base Redis après une indexation
- [x] écrire une fonction implémentant la commande de recherche, qui interroge Redis pour ramener les meilleurs résultats pour une recherche
- [x] contextualiser l'affichage des résultats avec le texte avoisinant

## Ressources

- Redis:
  - pour Linux, vous pouvez l'installer depuis les paquets de votre distribution,
    par exemple `apt install redis`
  - pour macOS, vous pouvez l'installer depuis Homebrew: `brew install redis`
  - sur tous les systèmes, vous pouvez l'installer par Docker, en suivant les
    instructions sur https://hub.docker.com/_/redis/
- Pilote Go pour Redis:
  - utiliser Redigo https://pkg.go.dev/github.com/gomodule/redigo/redis?tab=overview
  - API: https://godoc.org/github.com/gomodule/redigo/redis#hdr-Executing_Commands
  - Vous pouvez chercher des exemples en ligne, l'un des miens est visible sur
    https://github.com/fgm/drupal_redis_stats
  - Vous utiliserez l'adresse et le port Redis par défaut: `redis://localhost:6379/0`.
- Pour vider la base Redis entre deux essais vous pouvez utiliser en ligne de commande:
  `redis-cli flushall`
- Organisation du code:
  - à la racine du projet: `go.mod`, `go.sum`, `main.go`, `README.md`
  - dans `cmd/`: vos fichiers de commandes (root+3) et de test (1).
  - dans `engine/`: vos fichiers réalisant les traitements décrits à chaque étape:
    - `engine/disk.go`: fonctions liées à l'accès disque
    - `engine/redis.go`: fonctions liées à Redis
    - `engine/results.go`: fonctions liées à l'affichage des résultats
    - `engine/text.go`: fonctions liées à la manipulation de texte
    - `text_test.go`: tests des fonctions de manipulation de texte
- Codage : votre code doit être formaté au standard Go et passer 
  `golint -min_confidence 0.3 ./...` sans remonter de messages.
- Pensez à indiquer le temps total passé sur le devoir. Il devrait représenter
  de 3 à 6 heures de travail, tests unitaires et d'intégration compris.


## 1. Commandes

Votre programme s'appelera `search`.

Utiliser Cobra comme vu en cours (`cobra init search`) pour créer 3 commandes:

- `search index <répertoire de départ>`
  - Le but de cette commande sera de déclencher l'indexation à l'étape 10.
  - un seul argument : le chemin d'un répertoire. La fonction de commande `IndexDir`
    devra vérifier que le répertoire existe et est lisible, et sortir proprement
    sinon.
- `search dump`
  - Le but de cette commande, implémentée par une fonction `Dump`, est de lister
    l'intégralité de la base de données Redis sur la sortie standard, au format YAML :

```yaml
    <mot>:
      <fichier>: <nombre d'occurrences>
      ...fichier suivant...
    ...mot suivant...
```
- `search query <mot>`
  - Le but de cette commande sera de lister à l'étape 13 les résultats de
    l'indexation obtenue à l'étape 12, dans une fonction `Query`

A l'étape 1, vos commande ont juste besoin d'exister et d'afficher leurs arguments:

- `index <répertoire>`:
  - si le répertoire existe et est lisible, afficher `walking <répertoire>`
  - sinon afficher une erreur et terminer
- `dump`:
  - afficher `dumping index`
- `query <mot>`
  - afficher `querying index for <mot>`
  
Attention, sur macOS par exemple `/etc` est un lien symbolique vers `/private/etc`,
qui est traité comme un répertoire par `Stat`, mais dont le parcours ne ramène que
le lien lui-même. Si vous voulez lire son contenu, spécifiez-le comme `/etc/`
(ou bien traitez spécialement le cas avec `Lstat()` mais ce n'est pas utile ici).

Pensez, une fois la structure de commande Cobra créée, à activer le système de
modules par la commande `go mod init search`.


## 2. Écriture dans Redis

L'indexation est réalisée dans un "sorted set" Redis, avec `ZADD`:

    https://redis.io/commands/zadd

- Ajoutez à votre commande d'indexation le fait d'ouvrir la connexion à Redis,
  en appelant une fonction `func Dial() (redis.Conn, error)` placée dans le
  fichier de votre commande racine, et de sortir avec un message d'erreur en cas 
  d'échec de connexion.
- Créez une fonction ajoutant une entrée dans Redis:

```go
    func AddFile(c redis.Conn, key, file string, score int) error {
        // Utiliser Redigo pour écrire une entrée avec la commande Redis ZADD
    }
```
- Votre commande `index` doit invoquer cette fonction et vérifier le succès de
  la commande
```go
    err = AddFile(c, "test", "demo", 1)
```
  En cas d'échec, afficher l'erreur et sortir.


## 3. Lecture dans Redis

La lecture s'opère depuis un "sorted set" Redis, avec `ZREVRANGE`.

    https://redis.io/commands/zrevrange

- Ajoutez à votre commande de dump le fait d'ouvrir la connexion à Redis avec
  votre fonction `Dial`
- Créez une fonction lisant les valeurs d'une entrée dans Redis. Pour convertir
  le résultat de l'appel Redigo, pensez à utiliser un "helper":
  https://godoc.org/github.com/gomodule/redigo/redis#hdr-Reply_Helpers

```go
    func Get(c redis.Conn, key string) ([]string, error) {
        // Utilisez Redigo pour lire toutes les valeurs de la clef, et les
        // placer dans une tranche de chaînes. Renvoyez une erreur si nécessaire.
    }
```
- La fonction `Dump` de votre commande `dump` doit invoquer cette fonction et
  vérifier le succès de la commande, puis afficher le résultat:
```go
    files, err = Get(c, "test")
    // ...contrôlez le succès
    for _, file := range files {
        fmt.Println(file)
    }
```


## 4. Test d'intégration

_Cette étape est recommandée et notée, mais pas indispensable pour la suite_

Créez une fonction de test `TestReadWrite(t *testing.T)` qui réalise les opérations
suivantes :

- ouverture de la connexion Redis, échec en cas d'erreur
- effacement total de Redis avec la commande `FLUSHALL` via Redigo
  https://redis.io/commands/flushall
- écriture d'une valeur avec la fonction `AddFile` du 2.
- lecture d'une valeur avec la fonction `Get` du 3.
- vérification : le résultat doit contenir une seule chaîne, qui correspond à celle
  passée à `addHit`.


## 5. Lecture de texte

- Créez une fonction de lecture par mots :
```go
    func Scan(r io.Reader) ScanHits // ScanHits est un type défini pour map[string]int
```
  - La fonction doit parcourir le lecteur reçu en argument pour découper le texte
    en mots avec un `bufio.Scanner` avec le callback `bufio.ScanWords`.
  - Au fil de la lecture, elle remplit une carte comptant pour chaque mot trouvé le
    nombre de ses occurrences dans le fichier.
  - En l'absence de résultats, la fonction renvoie une carte vide non-nil.
- Créez une fonction de test unitaire `func TestScan( t *testing.T)` qui vérifie
  sur au moins trois textes d'exemple, dont un au moins sera la chaîne vide,
  que la fonction est correcte.
  - Pour créer un `Reader` à passer à `Scan`, utilisez `strings.NewReader` sur
    vos chaînes de test.


## 6. Indexation d'un fichier

- Créez une fonction `IndexFile(c redis.Conn, file string) error`
  - La fonction ouvre le fichier spécifié et le parcourt avec `Scan` du 5.
  - En cas d'échec, elle renvoie une erreur.
  - En cas de succès, elle enregistre les résultats dans Redis avec une boucle
    sur la fonction `AddHit` du 2.


## 7. Parcours d'arborescence disque

- Modifiez la fonction de commande `IndexDir` pour qu'elle parcoure le répertoire
  indiqué et ses sous-répertoires.
  - Pour cela, utilisez la fonction `filepath.Walk`
  - Pour créer votre fonction de rappel, créez une fonction `ScanFile` avec comme signature:
```go
    func ScanFile(c redis.Conn, path string, info os.FileInfo, err error) error
```

  - Puisque cette fonction n'est pas une `WalkFunction` du fait de l'argument
    supplémentaire `redis.Conn`, utilisez une fonction anonyme comme fonction de
    rappel, qui aura accès à la connexion Redis pour pouvoir invoquer `ScanFile`
  - Pour cette étape,
    - votre fonction vérifie si les fichiers sur lesquels elle
      est invoquée sont bien des fichiers ordinaires (et pas des symlinks, des
      répertoires, des fichiers périphériques, etc) avec le paramètre de type 
      `os.FileInfo` (regardez les constantes `os.ModeXXX`)
    - elle saute silencieusement tout ce qui n'est pas un simple fichier
    - elle affiche le chemin absolu des simples fichiers (un par ligne)
  - Ignorez les erreurs renvoyées par `filepath.Walk`.


## 8. Vérification de contenu

Pour cet exemple, vous allez utiliser la fonction standard `http.DetectContentType`
pour déterminer quel est le type d'un fichier.

- Créez une fonction `IsText(r io.Reader) bool` qui renvoie "vrai" si le contenu
  du lecteur qui lui est passé est identifié par la fonction standard comme
  étant un type MIME de la forme `text/<quelquechose>`, par exemple `text/plain`
  ou `text/html`.
  - Vous n'avez besoin que des 512 premiers octets du contenu du lecteur, donc
    utilisez un `LimitedReader`
  - _Attention_ : cette fonction standard identifie les sources JSON, Go, et PHP comme
    `text/plain`, ce n'est pas une erreur de votre part si vous obtenez ces résultats,
    le résultat dans ces cas doit bien être "vrai".
- Créez une fonction de test unitaire `TestIsText(t *testing.T)` à laquelle vous
  passerez des fichiers de différents types. Afin de pouvoir disposer de fichiers
  binaires à tester, vous utiliserez le mécanisme `testdata`
  - tel que décrit sur https://dave.cheney.net/2016/05/10/test-fixtures-in-go
  - utilisé par exemple dans `http_test.TestServeFile`
    https://golang.org/src/net/http/fs_test.go#L70
  - pour ne pas avoir de problème de chemins, lancez vos tests depuis la racine
    du projet, par `go test -v ./...`

## 9. Parcours d'arborescence filtré

- Modifiez votre fonction `ScanFile` pour qu'elle utilise `IsText` afin de ne
  plus lister que les fichiers "texte" au sens de `IsText`. 
  - Ignorez les fichiers illisibles, par exemple par manque de permissions.


## 10. Analyse de fichiers texte

- Complétez la fonction `ScanFile` pour qu'elle invoque la fonction `IndexFile`
  au lieu de simplement afficher le nom des fichiers.

À ce stade, votre indexeur est normalement complet.


## 11. Fonction de dump.

- Modifier la fonction `Dump` pour qu'elle ne liste plus simplement une clef
  avec `ZREVRANGE`, mais qu'elle parcoure l'ensemble des clefs avec `SCAN` pour
  parcourir l'ensemble de la base Redis.
- Afficher l'ensemble de la base au même format que précédemment, mais en triant
  par ordre alphabétique des clefs. Attention, cela nécessite trois étapes:
  - dans un premier temps, construisez en mémoire une tranche de chaînes contenant
    toutes les clefs ramenées par `SCAN`
  - dans un second temps, triez la tranche avec `sort.Strings`
  - dans un troisième temps, bouclez sur la tranche triée pour invoquer votre
    fonction `Get` sur chaque clef, et affichez chaque résultat sérialisé, donnant
    ainsi la sérialisation de l'ensemble.

Le code de https://github.com/fgm/drupal_redis_stats/blob/master/stats/stats.go#L95
peut vous être utile pour l'utilisation de `SCAN`, qui n'est pas complètement
intuitive, mais ce n'est qu'un exemple.


## 12. Requêtage

- Ajoutez à votre commande `query` le fait d'ouvrir la connexion à Redis,
  et de sortir avec un message d'erreur en cas d'échec de connexion. Réutilisez
  la fonction que vous aurez créée pour cela à l'étape 2.
- Créez une fonction `ShowResults(w io.Writer, key string, hits []string)`
  qui affiche le mot recherché et les noms de fichiers qui lui sont passés, un
  par ligne.
- Modifier la fonction `Query` pour que
  - elle extraie de Redis les 3 meilleurs résultats pour le mot recherché,
    sous la forme d'une tranche de chaînes triée du score le plus élevé au score
    le plus bas. Utilisez pour cela les paramètres additionels de `ZREVRANGE`.
    Certains mots auront moins de 3 résultats. Incluez leurs scores dans l'affichage.
  - elle invoque `ShowResults` pour l'affichage des résultats en lui passant
    `os.Stdout` comme `io.Writer`.

Exemple de résultat après avoir indexé `/etc/`:

```bash
$ search query your
Querying index for "your":
- /etc/authorization.deprecated (8 hits)
- /etc/apache2/original/httpd.conf (6 hits)
- /etc/apache2/httpd.conf~previous (6 hits)
$ 
```

## 13. Contextualisation

- Modifier la fonction `ShowResults` pour qu'elle contextualise les résultats en
  affichant pour chaque fichier la première ligne contenant le mot recherché.
  Pour cela, pour chaque fichier, elle doit
  - relire le fichier ligne par ligne avec un `bufio.Scanner` en mode `ScanLines`
  - à la première ligne correspondant au mot recherché, afficher
    - le nom du fichier
    - un caractère ":" et un espace
    - la ligne en cours
  - refermer le fichier et passer au suivant.
- Pour aligner les colonnes, il est conseillé d'utiliser le paquet `tabwriter`

Exemple de résultat après avoir indexé `/etc/`:

```
$ Querying index for "file":
File                             |Count |First match
/etc/apache2/magic               |15    |# Magic data for mod_mime_magic Apache module (originally for file(1) command)
/etc/apache2/original/httpd.conf |8     |# This is the main Apache HTTP server configuration file.  It contains the
/etc/apache2/httpd.conf~previous |8     |# This is the main Apache HTTP server configuration file.  It contains the
$ 
```
