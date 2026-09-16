import { describe, it, expect } from 'vitest'
import {
  episodeStatus,
  isDownloaded,
  formatEpisodeDate,
  podcastCoverUrl,
  songFromPodcastEpisode,
} from './helper'

describe('podcast helper', () => {
  describe('episodeStatus', () => {
    it('returns the episode status', () => {
      expect(episodeStatus({ status: 'completed' })).toBe('completed')
    })
    it('defaults to skipped when status missing', () => {
      expect(episodeStatus({})).toBe('skipped')
      expect(episodeStatus(undefined)).toBe('skipped')
    })
  })

  describe('isDownloaded', () => {
    it('is true when status is completed', () => {
      expect(isDownloaded({ status: 'completed' })).toBe(true)
    })
    it('is false otherwise', () => {
      expect(isDownloaded({ status: 'skipped' })).toBe(false)
    })
  })

  describe('formatEpisodeDate', () => {
    it('returns empty string for no date', () => {
      expect(formatEpisodeDate('')).toBe('')
      expect(formatEpisodeDate(null)).toBe('')
    })
    it('returns a locale date string for valid ISO', () => {
      const out = formatEpisodeDate('2024-01-02T00:00:00Z')
      expect(out).not.toBe('2024-01-02T00:00:00Z')
      expect(out.length).toBeGreaterThan(0)
    })
    it('returns the raw input for invalid dates', () => {
      expect(formatEpisodeDate('not-a-date')).toBe('not-a-date')
    })
  })

  describe('podcastCoverUrl', () => {
    it('prefers originalImageUrl', () => {
      expect(
        podcastCoverUrl({
          originalImageUrl: 'https://a/img.jpg',
          coverArt: 'c',
        }),
      ).toBe('https://a/img.jpg')
    })
    it('falls back to placeholder for a non-URL coverArt id', () => {
      // coverArt is an opaque plugin id (e.g. "ch-..."), not a usable image
      // URL, so it must not be returned as-is (the browser would treat it as
      // a relative path and 404).
      expect(podcastCoverUrl({ coverArt: 'ch-abc' })).toBe('podcast-icon.svg')
      expect(podcastCoverUrl({ coverArt: 'c' })).toBe('podcast-icon.svg')
    })
    it('uses an absolute http coverArt URL when present', () => {
      expect(
        podcastCoverUrl({ coverArt: 'https://covers.example/x.jpg' }),
      ).toBe('https://covers.example/x.jpg')
    })
    it('falls back to placeholder when nothing available', () => {
      expect(podcastCoverUrl({})).toBe('podcast-icon.svg')
      expect(podcastCoverUrl(null)).toBe('podcast-icon.svg')
    })
  })

  describe('songFromPodcastEpisode', () => {
    it('returns undefined for no episode', () => {
      expect(songFromPodcastEpisode(null)).toBeUndefined()
    })
    it('maps episode fields to a playable song', () => {
      const ep = {
        id: 'ep-1',
        streamId: 'ep-1',
        title: 'Episode 1',
        channelTitle: 'Show',
        streamUrl: 'https://enclosure/audio.mp3',
        originalImageUrl: 'https://img/cover.jpg',
        status: 'skipped',
      }
      const song = songFromPodcastEpisode(ep)
      expect(song.id).toBe('ep-1')
      expect(song.trackId).toBe('ep-1')
      expect(song.title).toBe('Episode 1')
      expect(song.name).toBe('Episode 1')
      expect(song.album).toBe('Show')
      expect(song.artist).toBe('Show')
      expect(song.streamUrl).toBe('https://enclosure/audio.mp3')
      expect(song.cover).toBe('https://img/cover.jpg')
      expect(song.isRadio).toBe(true)
      expect(song.isPodcast).toBe(true)
    })
    it('uses streamId when different from id', () => {
      const song = songFromPodcastEpisode({ id: 'ep-1', streamId: 'mf-9' })
      expect(song.trackId).toBe('mf-9')
    })
    it('uses placeholder cover when only an opaque coverArt id is present', () => {
      const song = songFromPodcastEpisode({
        id: 'ep-1',
        streamId: 'ep-1',
        title: 'Episode 1',
        coverArt: 'ch-abc',
        streamUrl: 'https://enclosure/audio.mp3',
      })
      expect(song.cover).toBe('podcast-icon.svg')
    })
  })
})
