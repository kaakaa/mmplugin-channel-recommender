package plugin

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const numOfChannels = 3

type ChannelCount struct {
	ChannelId string
	Count     int
}

func (p *Plugin) UserHasJoinedChannel(c *plugin.Context, channelMember *model.ChannelMember, actor *model.User) {
	p.API.LogDebug("Start UserHasJoinedChannel")
	userId := channelMember.UserId
	channelId := channelMember.ChannelId
	// Get team id
	channel, appErr := p.API.GetChannel(channelId)
	if appErr != nil {
		p.API.LogError("Failed to get channel.", "channel_id", channelId, "details", appErr)
		return
	}
	teamId := channel.TeamId

	// Exclude
	if includes(p.getConfiguration().excludedChannels, channel.Name) {
		p.API.LogDebug("This channel exclude for recommends", "channel_id", channelId)
		return
	}

	// Get all channel members
	stats, appErr := p.API.GetChannelStats(channelId)
	if appErr != nil {
		p.API.LogError("Failed to get channel stats.", "channel_id", channelId, "details", appErr)
		return
	}
	var members []model.ChannelMember
	perPage := 50
	for page := 0; page*perPage < int(stats.MemberCount); page += perPage {
		channelMembers, appErr := p.API.GetChannelMembers(channelId, page, perPage)
		if appErr != nil {
			p.API.LogError("Failed to get channel.", "channel_id", channelId, "page", page, "per_page", perPage, "details", appErr)
			return
		}
		for _, m := range *channelMembers {
			if m.UserId != userId {
				members = append(members, m)
			}
		}
	}
	p.API.LogDebug("Found channel_members", "number", len(members))
	if len(members) == 0 {
		return
	}

	// Count channels
	counter := map[string]int{}
	includeDeleted := false
	for _, member := range members {
		channels, appErr := p.API.GetChannelsForTeamForUser(teamId, member.UserId, includeDeleted)
		if appErr != nil {
			p.API.LogError("Failed to get channel for team for user.", "team_id", teamId, "user_id", member.UserId, "details", appErr)
			return
		}
		for _, ch := range channels {
			if v, ok := counter[ch.Id]; ok {
				counter[ch.Id] = v + 1
			} else {
				counter[ch.Id] = 1
			}
		}
	}
	if len(counter) == 0 {
		return
	}

	// Sort counter in DESC
	var alreadyJoined []string
	channels, appErr := p.API.GetChannelsForTeamForUser(teamId, userId, includeDeleted)
	if appErr != nil {
		p.API.LogError("Failed to get channel for team for user.", "team_id", teamId, "user_id", userId, "details", appErr)
		return
	}
	for _, ch := range channels {
		alreadyJoined = append(alreadyJoined, ch.Id)
	}
	var countResult []*ChannelCount
	for k, v := range counter {
		if includes(alreadyJoined, k) {
			continue
		}
		countResult = append(countResult, &ChannelCount{
			ChannelId: k,
			Count:     v,
		})
	}
	sort.Slice(countResult, func(i, j int) bool { return countResult[i].Count > countResult[j].Count })

	handlers := p.makeMessage(countResult, numOfChannels)
	if len(handlers) == 0 {
		return
	}
	var buf bytes.Buffer
	p.getConfiguration().text.Execute(&buf, map[string]interface{}{"Channels": strings.Join(handlers, ", ")})
	// GetUsers
	// Create Post for recommends
	p.API.SendEphemeralPost(userId, &model.Post{
		Type:      model.POST_EPHEMERAL,
		UserId:    userId,
		ChannelId: channelId,
		Message:   buf.String(),
	})
}

func includes(excludedChannels []string, channelHandle string) bool {
	for _, v := range excludedChannels {
		if v == channelHandle {
			return true
		}
	}
	return false
}

func (p *Plugin) makeMessage(countResult []*ChannelCount, num int) []string {
	var channelHandlers []string
	for i := 0; i < num && i < len(countResult); i++ {
		ch, appErr := p.API.GetChannel(countResult[i].ChannelId)
		if appErr != nil {
			p.API.LogError("Failed to get channel", "channel_id", ch.Id, "details", appErr)
			continue
		}
		p.API.LogDebug("Check if channel is excluded", "excluded", p.getConfiguration().excludedChannels, "channel_name", ch.Name)
		if includes(p.getConfiguration().excludedChannels, ch.Name) {
			continue
		}
		channelHandlers = append(channelHandlers, (fmt.Sprintf("~%s", ch.Name)))
	}
	return channelHandlers
}
