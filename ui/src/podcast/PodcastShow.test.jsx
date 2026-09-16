import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { Provider } from 'react-redux'
import { createStore, combineReducers } from 'redux'
import { ThemeProvider, createTheme } from '@material-ui/core/styles'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createMemoryHistory } from 'history'
import { Router } from 'react-router-dom'

const mockNotify = vi.fn()
const pushMock = vi.fn()
vi.mock('react-admin', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    useNotify: () => mockNotify,
    useTranslate: () => (x) => x,
    Title: ({ title }) => <div>{title}</div>,
  }
})

const getPodcastsMock = vi.fn()
const refreshPodcastsMock = vi.fn()
const playTracksMock = vi.fn()
const addTracksMock = vi.fn()
const setTrackMock = vi.fn()
vi.mock('../subsonic', () => ({
  default: {
    getPodcasts: (...a) => getPodcastsMock(...a),
    refreshPodcasts: (...a) => refreshPodcastsMock(...a),
    getCoverArtUrl: () => '/rest/getCoverArt?id=pc-test',
  },
}))
vi.mock('../actions', () => ({
  playTracks: (...a) => playTracksMock(...a),
  addTracks: (...a) => addTracksMock(...a),
  setTrack: (...a) => setTrackMock(...a),
}))

import PodcastShow from './PodcastShow'

const noopReducer = (state = {}, _action) => state

const renderShow = () => {
  const history = createMemoryHistory()
  history.push = pushMock
  const store = createStore(combineReducers({ player: noopReducer }), {})
  return render(
    <Provider store={store}>
      <ThemeProvider theme={createTheme()}>
        <Router history={history}>
          <PodcastShow />
        </Router>
      </ThemeProvider>
    </Provider>,
  )
}

describe('<PodcastShow />', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getPodcastsMock.mockResolvedValue([])
  })

  it('renders the channel HTML description as sanitized links', async () => {
    getPodcastsMock.mockResolvedValue([
      {
        id: 'ch-1',
        title: 'My Channel',
        description:
          'Rendez-vous sur <a href="https://example.com/rf">Radio France</a>',
        originalImageUrl: 'https://img/cover.jpg',
      },
    ])
    renderShow()
    await waitFor(() =>
      expect(screen.getByText('My Channel')).toBeInTheDocument(),
    )
    const link = await screen.findByRole('link', { name: 'Radio France' })
    expect(link).toHaveAttribute('href', 'https://example.com/rf')
    expect(
      screen.queryByText((content) => content.includes('<a href=')),
    ).not.toBeInTheDocument()
  })

  it('renders episode title and date in a single aligned content column', async () => {
    getPodcastsMock.mockResolvedValue([
      {
        id: 'ch-1',
        title: 'My Channel',
        description: 'desc',
        originalImageUrl: 'https://img/cover.jpg',
        episode: [
          {
            id: 'ep-1',
            streamId: 'ep-1',
            title: 'Episode One',
            publishDate: '2024-01-15T00:00:00Z',
            duration: 0,
            status: 'skipped',
            streamUrl: 'https://enclosure/a.mp3',
          },
        ],
      },
    ])
    renderShow()
    await waitFor(() =>
      expect(screen.getByText('Episode One')).toBeInTheDocument(),
    )
    expect(
      screen.getByText('resources.podcast.status.skipped'),
    ).toBeInTheDocument()
    const date = new Date('2024-01-15T00:00:00Z').toLocaleDateString()
    expect(screen.getByText(date)).toBeInTheDocument()
  })

  it('shows the not found state when no channel is returned', async () => {
    getPodcastsMock.mockResolvedValue([])
    renderShow()
    await waitFor(() => expect(getPodcastsMock).toHaveBeenCalled())
    expect(screen.getByText('resources.podcast.notFound')).toBeInTheDocument()
  })

  it('surfaces the server error message when loading fails', async () => {
    getPodcastsMock.mockRejectedValue(new Error('plugin not configured'))
    renderShow()
    await waitFor(() => expect(getPodcastsMock).toHaveBeenCalled())
    await waitFor(() =>
      expect(mockNotify).toHaveBeenCalledWith('plugin not configured', {
        type: 'warning',
      }),
    )
  })
})
