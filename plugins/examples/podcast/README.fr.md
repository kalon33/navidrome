# Plugin Podcast Navidrome

Un plugin **backend RSS pour podcasts** fonctionnel qui implémente la
capacité **Podcast** de Navidrome et sert l'intégralité de la surface d'API
Subsonic pour les podcasts (`getPodcasts`, `getNewestPodcasts`,
`getPodcastEpisode`, `createPodcastChannel`, `refreshPodcasts`,
`downloadPodcastEpisode`, `deletePodcastChannel`, `deletePodcastEpisode`).

Contrairement à un bouchon, ce plugin est un vrai backend : il s'abonne aux flux
RSS, les récupère et les analyse via le service hôte HTTP, persiste les chaînes
et les épisodes dans le KVStore (survivant aux redémarrages de Navidrome) et
planifie des actualisations périodiques des flux via le service hôte Scheduler.

## Fonctionnement

- **Abonnement** (`createPodcastChannel`) : récupère l'URL du flux, analyse le
  flux RSS 2.0 (`encoding/xml`, avec les extensions de l'espace de noms iTunes)
  et stocke la chaîne + les épisodes dans le KVStore.
- **Persistance** : les chaînes sont stockées sous `channel:<id>`, avec un index
  `channel-url:<url> -> <id>` pour la déduplication et une liste
  `index:channels`. Les identifiants sont stables, dérivés de `sha256(url)` pour
  les chaînes et `sha256(channelID:guid)` pour les épisodes, afin que les clients
  puissent les mémoriser et les lire entre les redémarrages.
- **Liste/Lecture** (`getPodcasts`, `getNewestPodcasts`, `getPodcastEpisode`) :
  lit depuis le KVStore. `getNewestPodcasts` renvoie les épisodes les plus
  récemment publiés parmi toutes les chaînes, triés par `publishDate`.
- **Actualisation** (`refreshPodcasts`, plus le callback planifié) : récupère
  à nouveau chaque flux, fusionne les nouveaux épisodes et préserve le statut de
  téléchargement des épisodes déjà connus.
- **Téléchargement** (`downloadPodcastEpisode`) : vérifie que l'enclosure de
  l'épisode est accessible (requête HEAD) et la marque `completed`/`error`. Les
  épisodes ne sont pas écrits sur disque : leur `streamUrl` pointe vers
  l'enclosure afin que les clients Subsonic lisent directement depuis
  l'éditeur. Un backend nécessitant des copies hors ligne téléchargerait le
  fichier via le service hôte Storage.
- **Suppression** (`deletePodcastChannel`, `deletePodcastEpisode`) : supprime
  la chaîne/l'épisode du KVStore.
- **Actualisation automatique** : au chargement (`nd_on_init`), le plugin
  enregistre une planification récurrente (configuration `refreshSchedule`, par
  défaut `0 * * * *` = toutes les heures) qui déclenche une actualisation de
  tous les flux abonnés via `nd_scheduler_callback`.

## Configuration

Le manifeste déclare une option de configuration `refreshSchedule` (expression
cron, par défaut toutes les heures). Configurez-la depuis l'interface web de
Navidrome ou laissez vide pour désactiver l'actualisation automatique. Les
abonnements eux-mêmes sont ajoutés à l'exécution via l'endpoint Subsonic
`createPodcastChannel` (par exemple depuis un client Subsonic).

## Permissions

| Service     | Raison                                                    |
|-------------|-----------------------------------------------------------|
| `http`      | Récupérer et analyser les flux RSS de podcasts depuis le web public |
| `kvstore`   | Persister les abonnements aux chaînes et les épisodes entre les redémarrages |
| `scheduler` | Planifier les actualisations périodiques des flux de podcasts |

`requiredHosts` HTTP est intentionnellement laissé ouvert : les flux de
podcasts se trouvent sur des hôtes publics arbitraires. La protection SSRF de
Navidrome continue de bloquer les adresses privées, de rebouclage et locales
de lien, sauf si une IP/CIDR explicite est ajoutée.

## Compilation

1. Installez [TinyGo](https://tinygo.org/getting-started/install/) (produit des
   binaires plus petits). La commande standard `GOOS=wasip1 GOARCH=wasm go
   build` fonctionne également.
2. Compilez le plugin :
   ```bash
   cd plugins/examples/podcast
   go mod tidy
   tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .
   zip -j podcast.ndp manifest.json plugin.wasm
   ```
   Ou en utilisant le Makefile des exemples :
   ```bash
   cd plugins/examples
   make podcast.ndp
   ```

## Installation

Copiez `podcast.ndp` dans votre dossier de plugins Navidrome (par défaut
`<data-folder>/plugins/`) et activez les plugins dans votre `navidrome.toml` :

```toml
[Plugins]
Enabled = true
```

Une fois activé, les endpoints de podcast Subsonic sont servis par ce plugin au
lieu de renvoyer « no podcast provider configured ».

## Extension

Ce backend lit les épisodes directement depuis l'URL d'enclosure de l'éditeur.
Pour prendre en charge la lecture hors ligne, étendez `DownloadEpisode` pour
télécharger l'enclosure vers le répertoire `/storage` via le service hôte
Storage et définissez les champs `path`/`suffix`/`contentType` de l'épisode vers
le fichier local, en laissant `streamUrl` vide afin que Navidrome lise le
fichier téléchargé à la place.

Voir `plugins/README.md` pour la référence complète des services hôtes.
