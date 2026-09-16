import React from 'react'
import MicIcon from '@material-ui/icons/Mic'
import MicNoneOutlinedIcon from '@material-ui/icons/MicNoneOutlined'
import DynamicMenuIcon from '../layout/DynamicMenuIcon'
import PodcastList from './PodcastList'

const podcast = {
  list: PodcastList,
  icon: (
    <DynamicMenuIcon
      path={'podcast'}
      icon={MicNoneOutlinedIcon}
      activeIcon={MicIcon}
    />
  ),
}

export default podcast
