// Copyright (c) 2025 Sudo-Ivan
// MIT License

package output

import (
	"fmt"
	"github.com/Sudo-Ivan/discourse-tui-client/pkg/discourse"
)

var client *discourse.Client

func SetClient(c *discourse.Client) {
	client = c
}

func getTopicPosts(topicID int) (*discourse.TopicResponse, error) {
	if client == nil {
		return nil, fmt.Errorf("client not set")
	}
	return client.GetTopicPosts(topicID)
}
