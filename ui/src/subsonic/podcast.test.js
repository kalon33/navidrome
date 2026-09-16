import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../dataProvider', () => ({
  httpClient: vi.fn(),
  clientUniqueId: 'test-id',
  clientUniqueIdHeader: 'X-ND-Client-Unique-Id',
}))

import subsonic from './index'
import { httpClient } from '../dataProvider'

const localStorageMock = () => {
  const store = {
    username: 'testuser',
    'subsonic-token': 'testtoken',
    'subsonic-salt': 'testsalt',
  }
  Object.defineProperty(window, 'localStorage', {
    value: { getItem: vi.fn((k) => store[k] || null) },
    configurable: true,
  })
}

const mockResp = (body) => ({
  json: { 'subsonic-response': body },
})

describe('subsonic podcast client', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorageMock()
  })

  it('getPodcasts returns channels', async () => {
    httpClient.mockResolvedValue(
      mockResp({
        status: 'ok',
        podcasts: { channel: [{ id: 'ch-1', title: 'A' }] },
      }),
    )
    const channels = await subsonic.getPodcasts()
    expect(channels).toHaveLength(1)
    expect(channels[0].id).toBe('ch-1')
    expect(httpClient).toHaveBeenCalled()
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('getPodcasts')
    expect(calledUrl).toContain('includeEpisodes=true')
  })

  it('getPodcasts passes id param', async () => {
    httpClient.mockResolvedValue(
      mockResp({ status: 'ok', podcasts: { channel: [] } }),
    )
    await subsonic.getPodcasts('ch-1')
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('id=ch-1')
  })

  it('getPodcasts returns empty array when channels missing', async () => {
    httpClient.mockResolvedValue(mockResp({ status: 'ok', podcasts: {} }))
    const channels = await subsonic.getPodcasts()
    expect(channels).toEqual([])
  })

  it('getNewestPodcasts returns episodes', async () => {
    httpClient.mockResolvedValue(
      mockResp({
        status: 'ok',
        newestPodcasts: { episode: [{ id: 'ep-1', title: 'E1' }] },
      }),
    )
    const episodes = await subsonic.getNewestPodcasts(5)
    expect(episodes).toHaveLength(1)
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('getNewestPodcasts')
    expect(calledUrl).toContain('count=5')
  })

  it('getNewestPodcasts returns empty when missing', async () => {
    httpClient.mockResolvedValue(mockResp({ status: 'ok', newestPodcasts: {} }))
    expect(await subsonic.getNewestPodcasts()).toEqual([])
  })

  it('getPodcastEpisode returns the episode', async () => {
    httpClient.mockResolvedValue(
      mockResp({ status: 'ok', podcastEpisode: { id: 'ep-1', title: 'E1' } }),
    )
    const ep = await subsonic.getPodcastEpisode('ep-1')
    expect(ep.id).toBe('ep-1')
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('getPodcastEpisode')
    expect(calledUrl).toContain('id=ep-1')
  })

  it('createPodcastChannel calls the endpoint with url param', async () => {
    httpClient.mockResolvedValue(mockResp({ status: 'ok' }))
    await subsonic.createPodcastChannel('https://feed.xml')
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('createPodcastChannel')
    expect(calledUrl).toContain('url=https%3A%2F%2Ffeed.xml')
  })

  it('refreshPodcasts calls the endpoint', async () => {
    httpClient.mockResolvedValue(mockResp({ status: 'ok' }))
    await subsonic.refreshPodcasts()
    expect(httpClient.mock.calls[0][0]).toContain('refreshPodcasts')
  })

  it('downloadPodcastEpisode calls the endpoint with id', async () => {
    httpClient.mockResolvedValue(mockResp({ status: 'ok' }))
    await subsonic.downloadPodcastEpisode('ep-1')
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('downloadPodcastEpisode')
    expect(calledUrl).toContain('id=ep-1')
  })

  it('deletePodcastChannel calls the endpoint with id', async () => {
    httpClient.mockResolvedValue(mockResp({ status: 'ok' }))
    await subsonic.deletePodcastChannel('ch-1')
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('deletePodcastChannel')
    expect(calledUrl).toContain('id=ch-1')
  })

  it('deletePodcastEpisode calls the endpoint with id', async () => {
    httpClient.mockResolvedValue(mockResp({ status: 'ok' }))
    await subsonic.deletePodcastEpisode('ep-1')
    const calledUrl = httpClient.mock.calls[0][0]
    expect(calledUrl).toContain('deletePodcastEpisode')
    expect(calledUrl).toContain('id=ep-1')
  })
})
