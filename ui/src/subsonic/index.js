import { baseUrl } from '../utils'
import {
  httpClient,
  clientUniqueId,
  clientUniqueIdHeader,
} from '../dataProvider'

const url = (command, id, options) => {
  const username = localStorage.getItem('username')
  const token = localStorage.getItem('subsonic-token')
  const salt = localStorage.getItem('subsonic-salt')
  if (!username || !token || !salt) {
    return ''
  }

  const params = new URLSearchParams()
  params.append('u', username)
  params.append('t', token)
  params.append('s', salt)
  params.append('f', 'json')
  params.append('v', '1.8.0')
  params.append('c', 'NavidromeUI')
  id && params.append('id', id)
  if (options) {
    if (options.ts) {
      options['_'] = new Date().getTime()
      delete options.ts
    }
    Object.keys(options).forEach((k) => {
      const value = options[k]
      // Handle array parameters by appending each value separately
      if (Array.isArray(value)) {
        value.forEach((v) => params.append(k, v))
      } else {
        params.append(k, value)
      }
    })
  }
  return `/rest/${command}?${params.toString()}`
}

const ping = () => httpClient(url('ping'))

const reportPlaybackUrl = (mediaId, positionMs, state) =>
  url('reportPlayback', null, { mediaId, mediaType: 'song', positionMs, state })

const reportPlayback = (mediaId, positionMs, state) =>
  httpClient(reportPlaybackUrl(mediaId, positionMs, state))

const reportPlaybackKeepalive = (mediaId, positionMs, state) => {
  const u = reportPlaybackUrl(mediaId, positionMs, state)
  if (u) {
    fetch(baseUrl(u), {
      keepalive: true,
      headers: { [clientUniqueIdHeader]: clientUniqueId },
    })
  }
}

const star = (id) => httpClient(url('star', id))

const unstar = (id) => httpClient(url('unstar', id))

const setRating = (id, rating) => httpClient(url('setRating', id, { rating }))

const download = (id, format = 'raw', bitrate = '0') =>
  (window.location.href = baseUrl(url('download', id, { format, bitrate })))

const startScan = (options) => httpClient(url('startScan', null, options))

const getScanStatus = () => httpClient(url('getScanStatus'))

const getNowPlaying = () => httpClient(url('getNowPlaying'))

const getAvatarUrl = (username, size) =>
  baseUrl(
    url('getAvatar', null, {
      username,
      ...(size && { size }),
    }),
  )

const getCoverArtUrl = (record, size, square) => {
  const suffix = record.imageHash ? '_' + record.imageHash : ''
  const options = {
    // A hash-suffixed url is already pixel-versioned; the buster would defeat immutable caching.
    ...(!record.imageHash && record.updatedAt && { _: record.updatedAt }),
    ...(size && { size }),
    ...(square && { square }),
  }

  // TODO Move this logic to server
  if (record.album) {
    return baseUrl(url('getCoverArt', 'mf-' + record.id + suffix, options))
  } else if (record.albumArtist) {
    return baseUrl(url('getCoverArt', 'al-' + record.id + suffix, options))
  } else if (record.sync !== undefined) {
    // This is a playlist
    return baseUrl(url('getCoverArt', 'pl-' + record.id + suffix, options))
  } else if (record.isPodcast) {
    // Podcast channel/episode cover, resolved server-side via the channel
    return baseUrl(
      url(
        'getCoverArt',
        'pc-' + (record.channelId || record.id) + suffix,
        options,
      ),
    )
  } else if (record.streamUrl !== undefined) {
    // This is a radio station
    return baseUrl(url('getCoverArt', 'ra-' + record.id + suffix, options))
  } else {
    return baseUrl(url('getCoverArt', 'ar-' + record.id + suffix, options))
  }
}

const getDiscCoverArtUrl = (albumId, discNumber, updatedAt, size) => {
  const options = {
    ...(updatedAt && { _: updatedAt }),
    ...(size && { size }),
  }
  return baseUrl(
    url('getCoverArt', 'dc-' + albumId + ':' + discNumber, options),
  )
}

const getArtistInfo = (id) => {
  return httpClient(url('getArtistInfo', id))
}

const getAlbumInfo = (id) => {
  return httpClient(url('getAlbumInfo', id))
}

const getSimilarSongs2 = (id, count = 100) => {
  return httpClient(url('getSimilarSongs2', id, { count }))
}

const getTopSongs = (artist, count = 50) => {
  return httpClient(url('getTopSongs', null, { artist, count }))
}

const streamUrl = (id, options) => {
  return baseUrl(
    url('stream', id, {
      ts: true,
      ...options,
    }),
  )
}

const subsonicResponse = (resp) => resp.json['subsonic-response']

const requireOk = (resp) => {
  const data = subsonicResponse(resp)
  if (!data || data.status !== 'ok') {
    const message = data?.error?.message || data?.status || 'subsonic error'
    throw new Error(message)
  }
  return data
}

const getPodcasts = (id) => {
  const options = { includeEpisodes: true }
  if (id) {
    options.id = id
  }
  return httpClient(url('getPodcasts', null, options)).then((resp) => {
    const data = requireOk(resp)
    return data.podcasts?.channel || []
  })
}

const getNewestPodcasts = (count = 20) => {
  return httpClient(url('getNewestPodcasts', null, { count })).then((resp) => {
    const data = requireOk(resp)
    return data.newestPodcasts?.episode || []
  })
}

const getPodcastEpisode = (id) => {
  return httpClient(url('getPodcastEpisode', null, { id })).then((resp) => {
    const data = requireOk(resp)
    return data.podcastEpisode
  })
}

const createPodcastChannel = (feedUrl) =>
  httpClient(url('createPodcastChannel', null, { url: feedUrl })).then(
    requireOk,
  )

const refreshPodcasts = () => httpClient(url('refreshPodcasts')).then(requireOk)

const downloadPodcastEpisode = (id) =>
  httpClient(url('downloadPodcastEpisode', null, { id })).then(requireOk)

const deletePodcastChannel = (id) =>
  httpClient(url('deletePodcastChannel', null, { id })).then(requireOk)

const deletePodcastEpisode = (id) =>
  httpClient(url('deletePodcastEpisode', null, { id })).then(requireOk)

export default {
  url,
  ping,
  reportPlayback,
  reportPlaybackKeepalive,
  download,
  star,
  unstar,
  setRating,
  startScan,
  getScanStatus,
  getNowPlaying,
  getCoverArtUrl,
  getDiscCoverArtUrl,
  getAvatarUrl,
  streamUrl,
  getPodcasts,
  getNewestPodcasts,
  getPodcastEpisode,
  createPodcastChannel,
  refreshPodcasts,
  downloadPodcastEpisode,
  deletePodcastChannel,
  deletePodcastEpisode,
  getAlbumInfo,
  getArtistInfo,
  getTopSongs,
  getSimilarSongs2,
}
