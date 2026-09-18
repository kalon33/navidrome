# Podcast Navidrome Plugin

A functional **RSS podcast backend** plugin that implements the Navidrome
**Podcast** capability and serves the full Subsonic podcast API surface
(`getPodcasts`, `getNewestPodcasts`, `getPodcastEpisode`, `createPodcastChannel`,
`refreshPodcasts`, `downloadPodcastEpisode`, `deletePodcastChannel`,
`deletePodcastEpisode`).

Unlike a stub, this plugin is a real backend: it subscribes to RSS feeds,
fetches and parses them over the HTTP host service, persists channels and
episodes in the KVStore (surviving Navidrome restarts), and schedules periodic
feed refreshes via the Scheduler host service.

## How it works

- **Subscribe** (`createPodcastChannel`): fetches the feed URL, parses the RSS
  2.0 feed (`encoding/xml`, with iTunes namespace extensions), and stores the
  channel + episodes in the KVStore.
- **Persistence**: channels are stored under `channel:<id>`, with a
  `channel-url:<url> -> <id>` index for de-duplication and an `index:channels`
  list. IDs are stable, derived from `sha256(url)` for channels and
  `sha256(channelID:guid)` for episodes, so clients can bookmark and stream
  them across restarts.
- **List/Get** (`getPodcasts`, `getNewestPodcasts`, `getPodcastEpisode`): read
  from the KVStore. `getNewestPodcasts` returns the most recently published
  episodes across all channels, sorted by `publishDate`.
- **Refresh** (`refreshPodcasts`, plus the scheduled callback): re-fetches each
  feed, merges new episodes, and preserves the download status of episodes that
  were already known.
- **Episode status**: episodes with a reachable enclosure URL are marked
  `completed` by default, since they are immediately streamable from the
  publisher's URL. This makes episodes visible and playable in Subsonic
  clients that only surface `completed` episodes (e.g. Tempus). Episodes
  without a usable enclosure are `new`. The full OpenSubsonic PodcastStatus
  surface is supported: `new`, `downloading`, `completed`, `error`, `deleted`,
  `skipped`.
- **Status preservation on refresh**: only `completed`, `error`, and `deleted`
  are preserved across feed refreshes. `new` and `skipped` are re-evaluated
  against the enclosure URL so episodes pick up the correct default. This
  also migrates legacy data: older versions defaulted non-downloaded episodes
  to `skipped`, which would otherwise stay stuck on `skipped` and never become
  visible to clients filtering on `completed` (e.g. Tempus). After rebuilding and
  reinstalling this plugin, a `refreshPodcasts` call re-evaluates all episodes
  and marks streamable ones `completed`.
- **Download** (`downloadPodcastEpisode`): transitions the episode to
  `downloading`, verifies the enclosure is reachable (HEAD request), then sets
  `completed`/`error` accordingly. Episodes are not written to disk: their
  `streamUrl` points at the enclosure so Subsonic clients stream directly from
  the publisher. A backend that needs offline copies would download the file
  via the Storage host service.
- **Delete** (`deletePodcastChannel`, `deletePodcastEpisode`): removes the
  channel/episode from the KVStore.
- **Auto-refresh**: on load (`nd_on_init`) the plugin registers a recurring
  schedule (`refreshSchedule` config, default `0 * * * *` = hourly) that
  triggers a refresh of all subscribed feeds via the `nd_scheduler_callback`.

## Configuration

The manifest declares a `refreshSchedule` config option (cron expression,
default every hour). Configure it from the Navidrome web UI or leave empty to
disable automatic refreshes. Subscriptions themselves are added at runtime via
the Subsonic `createPodcastChannel` endpoint (e.g. from a Subsonic client).

## Permissions

| Service   | Reason                                              |
|-----------|-----------------------------------------------------|
| `http`    | Fetch and parse podcast RSS feeds from the public web |
| `kvstore` | Persist channel subscriptions and episodes across restarts |
| `scheduler` | Schedule periodic podcast feed refreshes           |

HTTP `requiredHosts` is intentionally left open: podcast feeds live on
arbitrary public hosts. Navidrome's SSRF protection still blocks private,
loopback, and link-local addresses unless an explicit IP/CIDR is added.

## Building

1. Install [TinyGo](https://tinygo.org/getting-started/install/) (produces
   smaller binaries). Standard `GOOS=wasip1 GOARCH=wasm go build` also works.
2. Build the plugin:

   ```bash
   cd plugins/examples/podcast
   go mod tidy
   tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .
   zip -j podcast.ndp manifest.json plugin.wasm
   ```

   Or using the examples Makefile:

   ```bash
   cd plugins/examples
   make podcast.ndp
   ```

## Installing

Copy `podcast.ndp` to your Navidrome plugins folder (default:
`<data-folder>/plugins/`) and enable plugins in your `navidrome.toml`:

```toml
[Plugins]
Enabled = true
```

Once enabled, the Subsonic podcast endpoints are served by this plugin instead
of returning "no podcast provider configured".

## Extending

This backend streams episodes directly from the publisher's enclosure URL. To
support offline playback, extend `DownloadEpisode` to download the enclosure
to the `/storage` directory via the Storage host service and set the episode's
`path`/`suffix`/`contentType` to the local file, leaving `streamUrl` empty so
Navidrome streams the downloaded file instead.

See `plugins/README.md` for the full host service reference.
