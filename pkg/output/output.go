// Copyright (c) 2025 Sudo-Ivan
// MIT License

package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git.quad4.io/discourse-tui-client/pkg/discourse"
)

type Formatter interface {
	Format(topics *discourse.Response) ([]byte, error)
}

type JSONFormatter struct{}

func (f *JSONFormatter) Format(topics *discourse.Response) ([]byte, error) {
	return json.MarshalIndent(topics, "", "  ")
}

type TextFormatter struct{}

func (f *TextFormatter) Format(topics *discourse.Response) ([]byte, error) {
	var content strings.Builder
	for _, topic := range topics.TopicList.Topics {
		content.WriteString(fmt.Sprintf("Topic: %s\n", topic.Title))
		if topic.CategoryName != "" {
			content.WriteString(fmt.Sprintf("Category: %s\n", topic.CategoryName))
		}
		if len(topic.Tags) > 0 {
			content.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(topic.Tags, ", ")))
		}
		content.WriteString(fmt.Sprintf("Created: %s\n", topic.CreatedAt.Format("2006-01-02 15:04:05")))
		content.WriteString(fmt.Sprintf("Replies: %d\n", topic.ReplyCount))
		content.WriteString(fmt.Sprintf("Views: %d\n", topic.Views))
		content.WriteString("\nPosts:\n")

		posts, err := getTopicPosts(topic.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch posts for topic %d: %w", topic.ID, err)
		}

		for _, post := range posts.PostStream.Posts {
			content.WriteString(fmt.Sprintf("\nPost #%d by %s (%s)\n", post.PostNumber, post.Name, post.Username))
			content.WriteString(fmt.Sprintf("Posted: %s\n", post.CreatedAt.Format("2006-01-02 15:04:05")))
			content.WriteString(fmt.Sprintf("Content:\n%s\n", post.Cooked))
			content.WriteString(fmt.Sprintf("Reads: %d | Score: %.1f\n", post.Reads, post.Score))
			content.WriteString("\n---\n")
		}
		content.WriteString("\n========================================\n\n")
	}
	return []byte(content.String()), nil
}

type HTMLFormatter struct{}

func (f *HTMLFormatter) Format(topics *discourse.Response) ([]byte, error) {
	var content strings.Builder
	content.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Discourse Topics</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .topic { margin-bottom: 40px; border-bottom: 2px solid #eee; padding-bottom: 20px; }
        .post { margin: 20px 0; padding: 15px; background: #f9f9f9; border-radius: 5px; }
        .meta { color: #666; font-size: 0.9em; }
        .content { margin-top: 10px; }
        .tags { color: #0066cc; }
        .category { color: #666; }
    </style>
</head>
<body>
`)

	for _, topic := range topics.TopicList.Topics {
		content.WriteString(fmt.Sprintf(`<div class="topic">
    <h2>%s</h2>`, topic.Title))

		if topic.CategoryName != "" {
			content.WriteString(fmt.Sprintf(`<div class="category">Category: %s</div>`, topic.CategoryName))
		}
		if len(topic.Tags) > 0 {
			content.WriteString(fmt.Sprintf(`<div class="tags">Tags: %s</div>`, strings.Join(topic.Tags, ", ")))
		}
		content.WriteString(fmt.Sprintf(`<div class="meta">
    Created: %s<br>
    Replies: %d<br>
    Views: %d
</div>`, topic.CreatedAt.Format("2006-01-02 15:04:05"), topic.ReplyCount, topic.Views))

		posts, err := getTopicPosts(topic.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch posts for topic %d: %w", topic.ID, err)
		}

		content.WriteString(`<div class="posts">`)
		for _, post := range posts.PostStream.Posts {
			content.WriteString(fmt.Sprintf(`<div class="post">
    <div class="meta">
        Post #%d by %s (%s)<br>
        Posted: %s<br>
        Reads: %d | Score: %.1f
    </div>
    <div class="content">%s</div>
</div>`, post.PostNumber, post.Name, post.Username,
				post.CreatedAt.Format("2006-01-02 15:04:05"),
				post.Reads, post.Score, post.Cooked))
		}
		content.WriteString(`</div></div>`)
	}

	content.WriteString(`</body></html>`)
	return []byte(content.String()), nil
}

func WriteToFile(path string, topics *discourse.Response) error {
	if !strings.HasSuffix(path, ".txt") && !strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, ".html") {
		return fmt.Errorf("output file must end with .txt, .json, or .html")
	}

	var formatter Formatter
	switch {
	case strings.HasSuffix(path, ".json"):
		formatter = &JSONFormatter{}
	case strings.HasSuffix(path, ".html"):
		formatter = &HTMLFormatter{}
	default:
		formatter = &TextFormatter{}
	}

	data, err := formatter.Format(topics)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
