package main

import (
	mmplugin "github.com/kaakaa/mmplugin-channel-recommender/server/plugin"
	"github.com/mattermost/mattermost-server/plugin"
)

func main() {
	plugin.ClientMain(&mmplugin.Plugin{})
}
