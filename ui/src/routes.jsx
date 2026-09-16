import React from 'react'
import { Route } from 'react-router-dom'
import Personal from './personal/Personal'
import PodcastShow from './podcast/PodcastShow'

const routes = [
  <Route exact path="/personal" render={() => <Personal />} key={'personal'} />,
  <Route
    exact
    path="/podcast/:id"
    render={() => <PodcastShow />}
    key={'podcast-show'}
  />,
]

export default routes
