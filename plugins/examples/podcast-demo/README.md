# Podcast Demo Navidrome Plugin

A minimal, in-memory podcast backend plugin that implements the Navidrome
**Podcast** capability and serves the Subsonic podcast API endpoints
(`getPodcasts`, `getNewestPodcasts`, `getPodcastEpisode`, `createPodcastChannel`, `refreshPodcasts`,
`downloadPodcastEpisode`, `deletePodcastChannel`, `deletePodcastEpisode`).

It serves a single hard-coded channel with one episode, and treats the
management operations as no-ops against the in-memory store. It is intended as a
reference implementation to demonstrate the capability surface.

## Building

1. Install [TinyGo](https://tinygo.org/getting-started/install/)
2. Build the plugin:

   ```bash
   go mod tidy
   tinygo build -o plugin.wasm -target wasip1 -buildmode=c-shared .
   zip -j podcast-demo.ndp manifest.json plugin.wasm
   ```

   Or using the examples Makefile:

   ```bash
   cd plugins/examples
   make podcast-demo.ndp
   ```

## Installing

Copy `podcast-demo.ndp` to your Navidrome plugins folder (default:
`<data-folder>/plugins/`).

Enable plugins in your `navidrome.toml`:

```toml
[Plugins]
Enabled = true
```

Once enabled, the Subsonic podcast endpoints will return the demo data instead
of the "not implemented" response.

## Building a real backend

This demo is in-memory and resets on every call. A real podcast backend should:

- Use the **HTTP** host service to fetch and parse RSS feeds.
- Use the **KVStore** host service to persist channels and episodes across
  restarts.
- Use the **Scheduler** host service to refresh feeds periodically (export the
  `nd_scheduler_callback` capability function to receive scheduled events).
- Use the **Task** host service to download episodes in the background (export
  the `nd_task_execute` capability function to process download tasks).
- Return stable channel/episode IDs so clients can bookmark and stream them.

See `plugins/README.md` for the full host service reference.
