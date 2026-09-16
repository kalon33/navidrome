package capabilities

// Podcast provides podcast management and retrieval for the Subsonic API.
// This capability allows plugins to act as a podcast backend: subscribing to
// channels (RSS feeds), listing channels and episodes, refreshing feeds, and
// downloading episodes. Navidrome maps the plugin's data to the Subsonic
// podcast endpoints (getPodcasts, getNewestPodcasts, createPodcastChannel,
// refreshPodcasts, downloadPodcastEpisode, deletePodcastChannel,
// deletePodcastEpisode, getPodcastEpisode).
//
// Plugins implementing this capability can choose which methods to implement.
// GetChannels and GetChannel are required to serve getPodcasts; the remaining
// methods are optional - plugins only need to provide the functionality they
// support. Methods that are not exported by the plugin are reported as not
// implemented to the API layer.
//
//nd:capability name=podcast
type Podcast interface {
	// GetChannels returns all podcast channels the plugin subscribes to,
	// optionally including their episodes.
	//nd:export name=nd_podcast_get_channels
	GetChannels(GetPodcastChannelsRequest) (*GetPodcastChannelsResponse, error)

	// GetChannel returns a single podcast channel by ID, optionally including
	// its episodes.
	//nd:export name=nd_podcast_get_channel
	GetChannel(GetPodcastChannelRequest) (*GetPodcastChannelResponse, error)

	// GetNewestEpisodes returns the most recently published podcast episodes.
	//nd:export name=nd_podcast_get_newest_episodes
	GetNewestEpisodes(GetNewestEpisodesRequest) (*GetNewestEpisodesResponse, error)

	// GetEpisode returns the metadata for a single podcast episode by ID.
	//nd:export name=nd_podcast_get_episode
	GetEpisode(GetPodcastEpisodeRequest) (*GetPodcastEpisodeResponse, error)

	// CreateChannel subscribes to a new podcast channel from the given feed URL.
	//nd:export name=nd_podcast_create_channel
	CreateChannel(CreatePodcastChannelRequest) (*CreatePodcastChannelResponse, error)

	// RefreshChannels requests the plugin to check for new podcast episodes.
	// If ChannelIDs is empty, all channels are refreshed.
	//nd:export name=nd_podcast_refresh_channels
	RefreshChannels(RefreshPodcastsRequest) (*RefreshPodcastsResponse, error)

	// DownloadEpisode requests the plugin to start downloading a given
	// podcast episode.
	//nd:export name=nd_podcast_download_episode
	DownloadEpisode(DownloadPodcastEpisodeRequest) (*DownloadPodcastEpisodeResponse, error)

	// DeleteChannel deletes a podcast channel.
	//nd:export name=nd_podcast_delete_channel
	DeleteChannel(DeletePodcastChannelRequest) (*DeletePodcastChannelResponse, error)

	// DeleteEpisode deletes a podcast episode.
	//nd:export name=nd_podcast_delete_episode
	DeleteEpisode(DeletePodcastEpisodeRequest) (*DeletePodcastEpisodeResponse, error)
}

// PodcastStatus is the status of a podcast channel or episode.
// One of: "new", "downloading", "completed", "error", "deleted", "skipped".
type PodcastStatus string

const (
	PodcastStatusNew        PodcastStatus = "new"
	PodcastStatusDownloading PodcastStatus = "downloading"
	PodcastStatusCompleted   PodcastStatus = "completed"
	PodcastStatusError       PodcastStatus = "error"
	PodcastStatusDeleted     PodcastStatus = "deleted"
	PodcastStatusSkipped     PodcastStatus = "skipped"
)

// GetPodcastChannelsRequest is the request for GetChannels.
type GetPodcastChannelsRequest struct {
	// IncludeEpisodes controls whether episodes are returned with each channel.
	IncludeEpisodes bool `json:"includeEpisodes"`
}

// GetPodcastChannelsResponse is the response for GetChannels.
type GetPodcastChannelsResponse struct {
	// Channels is the list of podcast channels.
	Channels []PodcastChannel `json:"channels"`
}

// GetPodcastChannelRequest is the request for GetChannel.
type GetPodcastChannelRequest struct {
	// ID is the channel ID.
	ID string `json:"id"`
	// IncludeEpisodes controls whether episodes are returned with the channel.
	IncludeEpisodes bool `json:"includeEpisodes"`
}

// GetPodcastChannelResponse is the response for GetChannel.
type GetPodcastChannelResponse struct {
	// Channel is the requested podcast channel.
	Channel PodcastChannel `json:"channel"`
}

// GetNewestEpisodesRequest is the request for GetNewestEpisodes.
type GetNewestEpisodesRequest struct {
	// Count is the maximum number of episodes to return.
	Count int32 `json:"count"`
}

// GetNewestEpisodesResponse is the response for GetNewestEpisodes.
type GetNewestEpisodesResponse struct {
	// Episodes is the list of most recently published episodes.
	Episodes []PodcastEpisode `json:"episodes"`
}

// GetPodcastEpisodeRequest is the request for GetEpisode.
type GetPodcastEpisodeRequest struct {
	// ID is the podcast episode ID.
	ID string `json:"id"`
}

// GetPodcastEpisodeResponse is the response for GetEpisode.
type GetPodcastEpisodeResponse struct {
	// Episode is the requested podcast episode.
	Episode *PodcastEpisode `json:"episode,omitempty"`
}

// CreatePodcastChannelRequest is the request for CreateChannel.
type CreatePodcastChannelRequest struct {
	// URL is the feed URL of the podcast to subscribe to.
	URL string `json:"url"`
}

// CreatePodcastChannelResponse is the response for CreateChannel.
type CreatePodcastChannelResponse struct {
	// Channel is the newly created podcast channel, if the plugin created it
	// synchronously. May be empty when creation is deferred.
	Channel *PodcastChannel `json:"channel,omitempty"`
}

// RefreshPodcastsRequest is the request for RefreshChannels.
type RefreshPodcastsRequest struct {
	// ChannelIDs is the list of channel IDs to refresh. When empty, the plugin
	// should refresh all channels.
	ChannelIDs []string `json:"channelIds,omitempty"`
}

// RefreshPodcastsResponse is the response for RefreshChannels.
type RefreshPodcastsResponse struct {
	// Refreshed is the list of channel IDs that were refreshed.
	Refreshed []string `json:"refreshed,omitempty"`
}

// DownloadPodcastEpisodeRequest is the request for DownloadEpisode.
type DownloadPodcastEpisodeRequest struct {
	// ID is the podcast episode ID to download.
	ID string `json:"id"`
}

// DownloadPodcastEpisodeResponse is the response for DownloadEpisode.
type DownloadPodcastEpisodeResponse struct {
	// Episode is the updated episode, with its new status, if available.
	Episode *PodcastEpisode `json:"episode,omitempty"`
}

// DeletePodcastChannelRequest is the request for DeleteChannel.
type DeletePodcastChannelRequest struct {
	// ID is the channel ID to delete.
	ID string `json:"id"`
}

// DeletePodcastChannelResponse is the response for DeleteChannel.
type DeletePodcastChannelResponse struct {
	// Deleted is true when the channel was deleted.
	Deleted bool `json:"deleted"`
}

// DeletePodcastEpisodeRequest is the request for DeleteEpisode.
type DeletePodcastEpisodeRequest struct {
	// ID is the episode ID to delete.
	ID string `json:"id"`
}

// DeletePodcastEpisodeResponse is the response for DeleteEpisode.
type DeletePodcastEpisodeResponse struct {
	// Deleted is true when the episode was deleted.
	Deleted bool `json:"deleted"`
}

// PodcastChannel represents a podcast channel (RSS feed subscription).
type PodcastChannel struct {
	// ID is the channel ID.
	ID string `json:"id"`
	// URL is the podcast channel feed URL.
	URL string `json:"url"`
	// Title is the channel title.
	Title string `json:"title,omitempty"`
	// Description is the channel description.
	Description string `json:"description,omitempty"`
	// CoverArt is an ID used for retrieving cover art.
	CoverArt string `json:"coverArt,omitempty"`
	// OriginalImageUrl is the URL of the original channel image.
	OriginalImageUrl string `json:"originalImageUrl,omitempty"`
	// Status is the channel status.
	Status PodcastStatus `json:"status"`
	// ErrorMessage is an error message, if status is "error".
	ErrorMessage string `json:"errorMessage,omitempty"`
	// Episodes are the podcast episodes for this channel.
	Episodes []PodcastEpisode `json:"episodes,omitempty"`
}

// PodcastEpisode represents a podcast episode.
type PodcastEpisode struct {
	// ID is the episode ID.
	ID string `json:"id"`
	// StreamID is the ID used for streaming the episode. When the episode is
	// downloaded, this may be a library media file ID; otherwise it may be the
	// episode ID itself or empty.
	StreamID string `json:"streamId,omitempty"`
	// ChannelID is the ID of the podcast channel this episode belongs to.
	ChannelID string `json:"channelId"`
	// Title is the episode title.
	Title string `json:"title,omitempty"`
	// Description is the episode description.
	Description string `json:"description,omitempty"`
	// PublishDate is the publication date as an ISO 8601 string.
	PublishDate string `json:"publishDate,omitempty"`
	// Status is the episode status.
	Status PodcastStatus `json:"status"`
	// ErrorMessage is an error message, if status is "error".
	ErrorMessage string `json:"errorMessage,omitempty"`
	// StreamURL is the URL to stream the episode from (when not downloaded).
	StreamURL string `json:"streamUrl,omitempty"`
	// CoverArt is an ID used for retrieving cover art.
	CoverArt string `json:"coverArt,omitempty"`
	// Year is the publication year.
	Year int32 `json:"year,omitempty"`
	// Genre is the genre (commonly "Podcast").
	Genre string `json:"genre,omitempty"`
	// Duration is the episode duration in seconds.
	Duration int32 `json:"duration,omitempty"`
	// BitRate is the episode bit rate in kbps.
	BitRate int32 `json:"bitRate,omitempty"`
	// Size is the episode size in bytes.
	Size int64 `json:"size,omitempty"`
	// ContentType is the MIME content type of the episode.
	ContentType string `json:"contentType,omitempty"`
	// Suffix is the file suffix (e.g. "mp3").
	Suffix string `json:"suffix,omitempty"`
	// Path is the relative path of the downloaded file, if available.
	Path string `json:"path,omitempty"`
}
