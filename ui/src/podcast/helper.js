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
  return record.originalImageUrl || record.coverArt || PODCAST_PLACEHOLDER_IMAGE
}

export const songFromPodcastEpisode = (ep) => {
  if (!ep) return undefined
  return {
    ...ep,
    id: ep.id,
    trackId: ep.streamId || ep.id,
    title: ep.title,
    album: ep.channelTitle || ep.title,
    artist: ep.channelTitle || 'Podcast',
    cover: podcastCoverUrl(ep),
    streamUrl: ep.streamUrl,
    isRadio: true,
    isPodcast: true,
  }
}
