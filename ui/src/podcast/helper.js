import { PODCAST_PLACEHOLDER_IMAGE } from '../consts'

export const episodeStatus = (ep) => ep?.status || 'skipped'

export const isDownloaded = (ep) => episodeStatus(ep) === 'completed'

export const formatEpisodeDate = (iso) => {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleDateString()
}

export const podcastCoverUrl = (record) => {
  if (!record) return PODCAST_PLACEHOLDER_IMAGE
  // coverArt is an opaque plugin id (e.g. "ch-..."), not a resolvable URL:
  // only a real remote image URL is usable directly by the UI; otherwise
  // fall back to the placeholder so the browser never treats the id as a
  // relative path (which produces 404s like /app/ch-...).
  if (record.originalImageUrl) return record.originalImageUrl
  if (record.coverArt && /^https?:\/\//.test(record.coverArt)) {
    return record.coverArt
  }
  return PODCAST_PLACEHOLDER_IMAGE
}

export const songFromPodcastEpisode = (ep) => {
  if (!ep) return undefined
  const cover = podcastCoverUrl(ep)
  return {
    ...ep,
    id: ep.id,
    trackId: ep.streamId || ep.id,
    name: ep.title,
    title: ep.title,
    album: ep.channelTitle || ep.title,
    artist: ep.channelTitle || 'Podcast',
    cover,
    streamUrl: ep.streamUrl,
    isRadio: true,
    isPodcast: true,
  }
}
