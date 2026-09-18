import React from 'react'
import { Route } from 'react-router-dom'
import config from './config'
import Personal from './personal/Personal'
import PodcastShow from './podcast/PodcastShow'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  config.podcastEnabled && (
    <Route
      exact
      path="/podcast/:id"
      render={() => <PodcastShow />}
      key={'podcast-show'}
    />
  ),
].filter(Boolean)

export default routes
