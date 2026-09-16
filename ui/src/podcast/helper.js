import subsonic from '../subsonic'
import { PODCAST_PLACEHOLDER_IMAGE } from '../consts'

export const episodeStatus = (ep) => ep?.status || 'new'

export const isDownloaded = (ep) => episodeStatus(ep) === 'completed'

export const formatEpisodeDate = (iso) => {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleDateString()
}

export const podcastCoverUrl = (record, size) => {
  if (!record) return PODCAST_PLACEHOLDER_IMAGE
  // Prefer the server-resolved cover (pc-<id> via getCoverArt) so the image is
  // fetched/proxied through the artwork service and resized server-side. The
  // channel id (or the record id for a channel) is the artwork id's ID part.
  const id = record.channelId || record.id
  if (id && /^ch-/.test(String(id))) {
    return subsonic.getCoverArtUrl({ isPodcast: true, id }, size)
  }
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
    duration: ep.duration,
    isPodcast: true,
  }
}
