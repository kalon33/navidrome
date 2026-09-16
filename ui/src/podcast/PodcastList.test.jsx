import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
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
  }
})

const getPodcastsMock = vi.fn()
const refreshPodcastsMock = vi.fn()
const createPodcastChannelMock = vi.fn()
const deletePodcastChannelMock = vi.fn()

vi.mock('../subsonic', () => ({
  default: {
    getPodcasts: (...a) => getPodcastsMock(...a),
    refreshPodcasts: (...a) => refreshPodcastsMock(...a),
    createPodcastChannel: (...a) => createPodcastChannelMock(...a),
    deletePodcastChannel: (...a) => deletePodcastChannelMock(...a),
  },
}))

import PodcastList from './PodcastList'

const renderList = () => {
  const history = createMemoryHistory()
  history.push = pushMock
  return render(
    <ThemeProvider theme={createTheme()}>
      <Router history={history}>
        <PodcastList />
      </Router>
    </ThemeProvider>,
  )
}

describe('<PodcastList />', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getPodcastsMock.mockResolvedValue([])
  })

  it('shows the loading spinner then the empty state', async () => {
    getPodcastsMock.mockResolvedValue([])
    renderList()
    await waitFor(() => expect(getPodcastsMock).toHaveBeenCalled())
    expect(screen.getByText('resources.podcast.empty')).toBeInTheDocument()
  })

  it('renders channels from the response and navigates on click', async () => {
    getPodcastsMock.mockResolvedValue([
      {
        id: 'ch-1',
        title: 'My Show',
        description: 'A show',
        originalImageUrl: 'https://img/cover.jpg',
      },
    ])
    renderList()
    await waitFor(() => expect(screen.getByText('My Show')).toBeInTheDocument())
    expect(screen.getByText('A show')).toBeInTheDocument()
    const card = screen
      .getByText('My Show')
      .closest('[class*="MuiCardActionArea"]')
    fireEvent.click(card)
    expect(pushMock).toHaveBeenCalledWith('/podcast/ch-1')
  })

  it('refreshes podcasts when the refresh button is clicked', async () => {
    getPodcastsMock.mockResolvedValue([])
    refreshPodcastsMock.mockResolvedValue({})
    renderList()
    await waitFor(() => expect(getPodcastsMock).toHaveBeenCalled())
    const refreshBtn = screen.getByRole('button', { name: 'ra.action.refresh' })
    fireEvent.click(refreshBtn)
    await waitFor(() => expect(refreshPodcastsMock).toHaveBeenCalled())
  })

  it('opens the create dialog and adds a channel', async () => {
    getPodcastsMock.mockResolvedValue([])
    createPodcastChannelMock.mockResolvedValue({})
    renderList()
    await waitFor(() => expect(getPodcastsMock).toHaveBeenCalled())
    fireEvent.click(
      screen.getByRole('button', { name: 'resources.podcast.actions.add' }),
    )
    const input = await screen.findByRole('textbox')
    fireEvent.change(input, { target: { value: 'https://feed.xml' } })
    fireEvent.click(screen.getByRole('button', { name: 'ra.action.save' }))
    await waitFor(() =>
      expect(createPodcastChannelMock).toHaveBeenCalledWith('https://feed.xml'),
    )
  })

  it('deletes a channel when the delete button is clicked', async () => {
    getPodcastsMock.mockResolvedValue([
      { id: 'ch-1', title: 'My Show', description: 'desc' },
    ])
    deletePodcastChannelMock.mockResolvedValue({})
    renderList()
    await waitFor(() => expect(screen.getByText('My Show')).toBeInTheDocument())
    const delBtn = screen.getByRole('button', { name: 'ra.action.delete' })
    fireEvent.click(delBtn)
    await waitFor(() =>
      expect(deletePodcastChannelMock).toHaveBeenCalledWith('ch-1'),
    )
  })
})
