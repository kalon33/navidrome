import React, { useEffect, useState, useCallback } from 'react'
import {
  Button,
  Card,
  CardContent,
  CardActionArea,
  CardMedia,
  CircularProgress,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  makeStyles,
  TextField,
  Tooltip,
  Typography,
} from '@material-ui/core'
import { useTranslate, useNotify, Title } from 'react-admin'
import { useHistory } from 'react-router-dom'
import AddIcon from '@material-ui/icons/Add'
import RefreshIcon from '@material-ui/icons/Refresh'
import DeleteIcon from '@material-ui/icons/Delete'
import subsonic from '../subsonic'
import { podcastCoverUrl } from './helper'

const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: theme.spacing(1),
  },
  toolbar: {
    display: 'flex',
    justifyContent: 'flex-end',
    padding: theme.spacing(1),
    gap: theme.spacing(1),
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
    gap: theme.spacing(2),
    padding: theme.spacing(2),
  },
  card: {
    position: 'relative',
    width: '100%',
  },
  media: {
    height: 180,
    objectFit: 'cover',
  },
  deleteBtn: {
    position: 'absolute',
    top: 4,
    right: 4,
    backgroundColor: 'rgba(0, 0, 0, 0.4)',
    color: '#fff',
    '&:hover': {
      backgroundColor: 'rgba(0, 0, 0, 0.6)',
    },
  },
  channelTitle: {
    fontWeight: 'bold',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  channelDesc: {
    color: theme.palette.text.secondary,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    display: '-webkit-box',
    WebkitLineClamp: 2,
    WebkitBoxOrient: 'vertical',
  },
  empty: {
    padding: theme.spacing(4),
    textAlign: 'center',
  },
}))

const PodcastList = () => {
  const translate = useTranslate()
  const notify = useNotify()
  const history = useHistory()
  const classes = useStyles()
  const [channels, setChannels] = useState([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [newUrl, setNewUrl] = useState('')
  const [creating, setCreating] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await subsonic.getPodcasts()
      setChannels(data)
    } catch (e) {
      notify(e.message || 'message.podcastLoadError', { type: 'warning' })
    }
    setLoading(false)
  }, [notify])

  useEffect(() => {
    load()
  }, [load])

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

  const handleCreate = async () => {
    if (!newUrl) return
    setCreating(true)
    try {
      await subsonic.createPodcastChannel(newUrl)
      setNewUrl('')
      setCreateOpen(false)
      notify('message.podcastChannelAdded', 'success')
      await load()
    } catch (e) {
      notify(e.message || 'message.podcastAddError', { type: 'warning' })
    }
    setCreating(false)
  }

  const handleDelete = async (id) => {
    try {
      await subsonic.deletePodcastChannel(id)
      notify('message.podcastChannelDeleted', 'success')
      await load()
    } catch (e) {
      notify(e.message || 'message.podcastDeleteError', { type: 'warning' })
    }
  }

  return (
    <Card className={classes.root}>
      <Title title={'Navidrome - ' + translate('resources.podcast.name')} />
      <div className={classes.toolbar}>
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
        <Button startIcon={<AddIcon />} onClick={() => setCreateOpen(true)}>
          {translate('resources.podcast.actions.add')}
        </Button>
      </div>
      {loading ? (
        <div className={classes.empty}>
          <CircularProgress />
        </div>
      ) : channels.length === 0 ? (
        <div className={classes.empty}>
          <Typography>{translate('resources.podcast.empty')}</Typography>
        </div>
      ) : (
        <div className={classes.grid}>
          {channels.map((ch) => (
            <Card key={ch.id} className={classes.card}>
              <CardActionArea onClick={() => history.push(`/podcast/${ch.id}`)}>
                <CardMedia
                  component="img"
                  className={classes.media}
                  image={podcastCoverUrl(ch)}
                  title={ch.title}
                />
                <CardContent>
                  <Typography className={classes.channelTitle} title={ch.title}>
                    {ch.title || ch.url}
                  </Typography>
                  <Typography
                    variant="body2"
                    className={classes.channelDesc}
                    title={ch.description}
                  >
                    {ch.description}
                  </Typography>
                </CardContent>
              </CardActionArea>
              <Tooltip title={translate('ra.action.delete')}>
                <IconButton
                  className={classes.deleteBtn}
                  size="small"
                  onClick={() => handleDelete(ch.id)}
                >
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            </Card>
          ))}
        </div>
      )}
      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} fullWidth>
        <DialogTitle>{translate('resources.podcast.actions.add')}</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            fullWidth
            type="url"
            label={translate('resources.podcast.fields.feedUrl')}
            value={newUrl}
            onChange={(e) => setNewUrl(e.target.value)}
            disabled={creating}
          />
          <div className={classes.toolbar}>
            <Button onClick={() => setCreateOpen(false)} disabled={creating}>
              {translate('ra.action.cancel')}
            </Button>
            <Button
              variant="contained"
              color="primary"
              onClick={handleCreate}
              disabled={creating || !newUrl}
            >
              {creating ? (
                <CircularProgress size={20} />
              ) : (
                translate('ra.action.save')
              )}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  )
}

export default PodcastList
