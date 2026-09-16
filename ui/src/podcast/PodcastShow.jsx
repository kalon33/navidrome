import React, { useEffect, useState, useCallback } from 'react'
import {
  Button,
  Card,
  CardMedia,
  CircularProgress,
  IconButton,
  List,
  ListItem,
  ListItemAvatar,
  ListItemSecondaryAction,
  ListItemText,
  makeStyles,
  Tooltip,
  Typography,
} from '@material-ui/core'
import { useTranslate, useNotify, Title } from 'react-admin'
import { useParams, useHistory } from 'react-router-dom'
import { useDispatch } from 'react-redux'
import ArrowBackIcon from '@material-ui/icons/ArrowBack'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import PlaylistAddIcon from '@material-ui/icons/PlaylistAdd'
import RefreshIcon from '@material-ui/icons/Refresh'
import CloudDownloadIcon from '@material-ui/icons/CloudDownload'
import DeleteIcon from '@material-ui/icons/Delete'
import subsonic from '../subsonic'
import {
  podcastCoverUrl,
  songFromPodcastEpisode,
  episodeStatus,
  isDownloaded,
  formatEpisodeDate,
} from './helper'
import { SafeHTML } from '../common/SafeHTML'
import { playTracks, addTracks, setTrack } from '../actions'
import { formatDuration } from '../utils'

const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: theme.spacing(1),
  },
  header: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: theme.spacing(2),
    [theme.breakpoints.up('sm')]: {
      flexDirection: 'row',
      alignItems: 'flex-start',
    },
  },
  cover: {
    width: 200,
    height: 200,
    objectFit: 'cover',
    borderRadius: theme.spacing(1),
  },
  headerInfo: {
    marginLeft: 0,
    [theme.breakpoints.up('sm')]: {
      marginLeft: theme.spacing(2),
    },
    marginTop: theme.spacing(2),
    [theme.breakpoints.up('sm')]: {
      marginTop: 0,
    },
  },
  toolbar: {
    display: 'flex',
    gap: theme.spacing(1),
    marginTop: theme.spacing(1),
  },
  description: {
    marginTop: theme.spacing(1),
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    display: '-webkit-box',
    WebkitLineClamp: 4,
    WebkitBoxOrient: 'vertical',
  },
  episodeList: {
    width: '100%',
  },
  episodeAvatar: {
    borderRadius: 4,
  },
  episodeContent: {
    display: 'flex',
    flexDirection: 'column',
    width: '100%',
    overflow: 'hidden',
  },
  episodeTitle: {
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
  },
  episodeTitleText: {
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  statusChip: {
    fontSize: '0.7rem',
    flexShrink: 0,
  },
  episodeMeta: {
    display: 'flex',
    alignItems: 'center',
    color: theme.palette.text.secondary,
    fontSize: '0.875rem',
  },
  empty: {
    padding: theme.spacing(4),
    textAlign: 'center',
  },
}))

const statusLabel = (status, translate) => {
  switch (status) {
    case 'completed':
      return translate('resources.podcast.status.downloaded')
    case 'downloading':
      return translate('resources.podcast.status.downloading')
    case 'error':
      return translate('resources.podcast.status.error')
    case 'skipped':
    default:
      return translate('resources.podcast.status.skipped')
  }
}

const PodcastShow = () => {
  const { id } = useParams()
  const translate = useTranslate()
  const notify = useNotify()
  const history = useHistory()
  const dispatch = useDispatch()
  const classes = useStyles()
  const [channel, setChannel] = useState(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await subsonic.getPodcasts(id)
      if (data.length > 0) {
        const ch = data[0]
        if (ch.episode) {
          ch.episode = ch.episode.map((ep) => ({
            ...ep,
            channelTitle: ch.title,
          }))
        }
        setChannel(ch)
      } else {
        setChannel(null)
      }
    } catch (e) {
      notify(e.message || 'message.podcastLoadError', { type: 'warning' })
    }
    setLoading(false)
  }, [id, notify])

  useEffect(() => {
    load()
  }, [load])

  const playEpisode = (ep) => {
    const song = songFromPodcastEpisode(ep)
    if (!song) return
    dispatch(setTrack(song))
  }

  const playAll = () => {
    if (!channel?.episode?.length) return
    const data = {}
    const ids = []
    channel.episode.forEach((ep) => {
      const song = songFromPodcastEpisode(ep)
      if (song) {
        data[song.id] = song
        ids.push(song.id)
      }
    })
    dispatch(playTracks(data, ids))
  }

  const addToQueue = (ep) => {
    const song = songFromPodcastEpisode(ep)
    if (!song) return
    const data = { [song.id]: song }
    dispatch(addTracks(data, [song.id]))
    notify('message.queueAdded', 'success')
  }

  const handleRefresh = async () => {
    setRefreshing(true)
    try {
      await subsonic.refreshPodcasts()
      await load()
    } catch (e) {
      notify(e.message || 'message.podcastRefreshError', { type: 'warning' })
    }
    setRefreshing(false)
  }

  const handleDownload = async (ep) => {
    try {
      await subsonic.downloadPodcastEpisode(ep.id)
      notify('message.podcastDownloadStarted', 'success')
      await load()
    } catch (e) {
      notify(e.message || 'message.podcastDownloadError', { type: 'warning' })
    }
  }

  const handleDeleteEpisode = async (ep) => {
    try {
      await subsonic.deletePodcastEpisode(ep.id)
      notify('message.podcastEpisodeDeleted', 'success')
      await load()
    } catch (e) {
      notify(e.message || 'message.podcastDeleteError', { type: 'warning' })
    }
  }

  if (loading) {
    return (
      <Card className={classes.root}>
        <div className={classes.empty}>
          <CircularProgress />
        </div>
      </Card>
    )
  }

  if (!channel) {
    return (
      <Card className={classes.root}>
        <Title title={'Navidrome - ' + translate('resources.podcast.name')} />
        <div className={classes.empty}>
          <Typography>{translate('resources.podcast.notFound')}</Typography>
          <Button onClick={() => history.push('/podcast')}>
            {translate('ra.action.back')}
          </Button>
        </div>
      </Card>
    )
  }

  const episodes = channel.episode || []

  return (
    <Card className={classes.root}>
      <Title
        title={
          'Navidrome - ' +
          (channel.title || translate('resources.podcast.name'))
        }
      />
      <div className={classes.toolbar}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={() => history.push('/podcast')}
        >
          {translate('ra.action.back')}
        </Button>
      </div>
      <div className={classes.header}>
        <CardMedia
          component="img"
          className={classes.cover}
          image={podcastCoverUrl(channel)}
          title={channel.title}
        />
        <div className={classes.headerInfo}>
          <Typography variant="h5">{channel.title || channel.url}</Typography>
          <Typography
            variant="body2"
            className={classes.description}
            component="div"
          >
            <SafeHTML>{channel.description}</SafeHTML>
          </Typography>
          <div className={classes.toolbar}>
            <Button
              variant="contained"
              color="primary"
              startIcon={<PlayArrowIcon />}
              onClick={playAll}
              disabled={episodes.length === 0}
            >
              {translate('resources.podcast.actions.playAll')}
            </Button>
            <Button
              startIcon={<RefreshIcon />}
              onClick={handleRefresh}
              disabled={refreshing}
            >
              {refreshing ? (
                <CircularProgress size={20} />
              ) : (
                translate('ra.action.refresh')
              )}
            </Button>
          </div>
        </div>
      </div>
      {episodes.length === 0 ? (
        <div className={classes.empty}>
          <Typography>{translate('resources.podcast.noEpisodes')}</Typography>
        </div>
      ) : (
        <List className={classes.episodeList}>
          {episodes.map((ep) => (
            <ListItem key={ep.id} alignItems="flex-start" button>
              <ListItemAvatar>
                <img
                  src={podcastCoverUrl(ep) || podcastCoverUrl(channel)}
                  alt={ep.title}
                  className={classes.episodeAvatar}
                  width={56}
                  height={56}
                />
              </ListItemAvatar>
              <ListItemText
                disableTypography
                primary={
                  <span className={classes.episodeContent}>
                    <span className={classes.episodeTitle}>
                      <span className={classes.episodeTitleText}>
                        {ep.title}
                      </span>
                      <Typography
                        component="span"
                        className={classes.statusChip}
                        color="textSecondary"
                      >
                        {statusLabel(episodeStatus(ep), translate)}
                      </Typography>
                    </span>
                    <span className={classes.episodeMeta}>
                      {formatEpisodeDate(ep.publishDate)}
                      {ep.duration > 0 && ` · ${formatDuration(ep.duration)}`}
                    </span>
                  </span>
                }
              />
              <ListItemSecondaryAction>
                <Tooltip title={translate('resources.podcast.actions.play')}>
                  <IconButton edge="end" onClick={() => playEpisode(ep)}>
                    <PlayArrowIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip
                  title={translate('resources.podcast.actions.addToQueue')}
                >
                  <IconButton edge="end" onClick={() => addToQueue(ep)}>
                    <PlaylistAddIcon />
                  </IconButton>
                </Tooltip>
                {!isDownloaded(ep) && (
                  <Tooltip
                    title={translate('resources.podcast.actions.download')}
                  >
                    <IconButton edge="end" onClick={() => handleDownload(ep)}>
                      <CloudDownloadIcon />
                    </IconButton>
                  </Tooltip>
                )}
                <Tooltip title={translate('ra.action.delete')}>
                  <IconButton
                    edge="end"
                    onClick={() => handleDeleteEpisode(ep)}
                  >
                    <DeleteIcon />
                  </IconButton>
                </Tooltip>
              </ListItemSecondaryAction>
            </ListItem>
          ))}
        </List>
      )}
    </Card>
  )
}

export default PodcastShow
