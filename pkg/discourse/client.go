// Copyright (c) 2025 Sudo-Ivan
// MIT License

package discourse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type User struct {
	ID             int    `json:"id"`
	Username       string `json:"username"`
	Name           string `json:"name"`
	AvatarTemplate string `json:"avatar_template"`
	TrustLevel     int    `json:"trust_level"`
	Moderator      bool   `json:"moderator,omitempty"`
}

type Topic struct {
	ID                 int       `json:"id"`
	Title              string    `json:"title"`
	FancyTitle         string    `json:"fancy_title"`
	Slug               string    `json:"slug"`
	PostsCount         int       `json:"posts_count"`
	ReplyCount         int       `json:"reply_count"`
	HighestPostNumber  int       `json:"highest_post_number"`
	ImageURL           string    `json:"image_url"`
	CreatedAt          time.Time `json:"created_at"`
	LastPostedAt       time.Time `json:"last_posted_at"`
	Bumped             bool      `json:"bumped"`
	BumpedAt           time.Time `json:"bumped_at"`
	Archetype          string    `json:"archetype"`
	Unseen             bool      `json:"unseen"`
	LastReadPostNumber int       `json:"last_read_post_number"`
	Unread             int       `json:"unread"`
	NewPosts           int       `json:"new_posts"`
	UnreadPosts        int       `json:"unread_posts"`
	Pinned             bool      `json:"pinned"`
	Unpinned           *bool     `json:"unpinned"`
	Visible            bool      `json:"visible"`
	Closed             bool      `json:"closed"`
	Archived           bool      `json:"archived"`
	NotificationLevel  int       `json:"notification_level"`
	Bookmarked         bool      `json:"bookmarked"`
	Liked              bool      `json:"liked"`
	Tags               []string  `json:"tags"`
	Views              int       `json:"views"`
	LikeCount          int       `json:"like_count"`
	LastPosterUsername string    `json:"last_poster_username"`
	CategoryID         int       `json:"category_id"`
	CategoryName       string    `json:"category_name"`
	CategoryColor      string    `json:"category_color"`
}

type TopicList struct {
	CanCreateTopic bool     `json:"can_create_topic"`
	MoreTopicsURL  string   `json:"more_topics_url"`
	PerPage        int      `json:"per_page"`
	TopTags        []string `json:"top_tags"`
	Topics         []Topic  `json:"topics"`
}

type Response struct {
	Users         []User    `json:"users"`
	PrimaryGroups []string  `json:"primary_groups"`
	FlairGroups   []string  `json:"flair_groups"`
	TopicList     TopicList `json:"topic_list"`
}

type Post struct {
	ID             int              `json:"id"`
	Name           string           `json:"name"`
	Username       string           `json:"username"`
	CreatedAt      time.Time        `json:"created_at"`
	Cooked         string           `json:"cooked"`
	PostNumber     int              `json:"post_number"`
	ReplyCount     int              `json:"reply_count"`
	TopicID        int              `json:"topic_id"`
	TopicSlug      string           `json:"topic_slug"`
	Reads          int              `json:"reads"`
	Score          float64          `json:"score"`
	ActionsSummary []ActionsSummary `json:"actions_summary,omitempty"`
}

type PostStream struct {
	Posts  []Post `json:"posts"`
	Stream []int  `json:"stream"`
}

type TopicResponse struct {
	PostStream PostStream `json:"post_stream"`
}

type ActionsSummary struct {
	ID      int  `json:"id"`
	Count   int  `json:"count"`
	Acted   bool `json:"acted"`
	CanUndo bool `json:"can_undo"`
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	TextColor   string `json:"text_color"`
	Slug        string `json:"slug"`
	TopicCount  int    `json:"topic_count"`
	PostCount   int    `json:"post_count"`
	Position    int    `json:"position"`
	Description string `json:"description"`
}

type CategoryList struct {
	CanCreateCategory bool       `json:"can_create_category"`
	CanCreateTopic    bool       `json:"can_create_topic"`
	Categories        []Category `json:"categories"`
}

type CategoryResponse struct {
	CategoryList CategoryList `json:"category_list"`
}

type apiCreateTopicPayload struct {
	Title     string   `json:"title"`
	Raw       string   `json:"raw"`
	Category  int      `json:"category"`
	Tags      []string `json:"tags,omitempty"`
	Archetype string   `json:"archetype"`
}

type Client struct {
	client       *http.Client
	baseURL      string
	cookiesPath  string
	pageCooldown time.Duration
}

func (c *Client) CookiesPath() string {
	return c.cookiesPath
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func NewClient(baseURL string, cookiesPath string) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required")
	}

	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	baseURL = strings.TrimSuffix(baseURL, "/")

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	return &Client{
		client:       client,
		baseURL:      baseURL,
		cookiesPath:  cookiesPath,
		pageCooldown: 500 * time.Millisecond,
	}, nil
}

func (c *Client) LoadCookies(cookieFile string) error {
	/* #nosec G304 */
	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return fmt.Errorf("failed to read cookie file: %v", err)
	}

	cookies := strings.Split(string(data), "\n")
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %v", err)
	}

	for _, cookie := range cookies {
		if cookie == "" {
			continue
		}
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) != 2 {
			continue
		}
		c.client.Jar.SetCookies(parsedURL, []*http.Cookie{
			{
				Name:  strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			},
		})
	}

	return nil
}

func (c *Client) GetLatestTopics() (*Response, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/latest.json", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest topics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Printf("Warning: failed to get cache directory: %v", err)
	} else {
		instanceDir := filepath.Join(userCacheDir, "discourse-tui-client", "instances", strings.TrimPrefix(strings.TrimPrefix(c.baseURL, "https://"), "http://"))
		if err := os.MkdirAll(instanceDir, 0750); err != nil {
			log.Printf("Warning: failed to create instance cache directory: %v", err)
		} else {
			cachePath := filepath.Join(instanceDir, "latest.json")
			if err := os.WriteFile(cachePath, body, 0600); err != nil { //nosec G306
				log.Printf("Warning: failed to save JSON to file: %v", err)
			}
		}
	}

	result := gjson.ParseBytes(body)
	response := &Response{}

	// Parse users
	users := result.Get("users")
	users.ForEach(func(_, value gjson.Result) bool {
		user := User{
			ID:             int(value.Get("id").Int()),
			Username:       value.Get("username").Str,
			Name:           value.Get("name").Str,
			AvatarTemplate: value.Get("avatar_template").Str,
			TrustLevel:     int(value.Get("trust_level").Int()),
			Moderator:      value.Get("moderator").Bool(),
		}
		response.Users = append(response.Users, user)
		return true
	})

	// Parse topic list
	topicList := result.Get("topic_list")
	response.TopicList.CanCreateTopic = topicList.Get("can_create_topic").Bool()
	response.TopicList.MoreTopicsURL = topicList.Get("more_topics_url").Str
	response.TopicList.PerPage = int(topicList.Get("per_page").Int())

	// Parse topics
	topics := topicList.Get("topics")
	topics.ForEach(func(_, value gjson.Result) bool {
		topic := Topic{
			ID:                 int(value.Get("id").Int()),
			Title:              value.Get("title").Str,
			FancyTitle:         value.Get("fancy_title").Str,
			Slug:               value.Get("slug").Str,
			PostsCount:         int(value.Get("posts_count").Int()),
			ReplyCount:         int(value.Get("reply_count").Int()),
			HighestPostNumber:  int(value.Get("highest_post_number").Int()),
			ImageURL:           value.Get("image_url").Str,
			CreatedAt:          value.Get("created_at").Time(),
			LastPostedAt:       value.Get("last_posted_at").Time(),
			Bumped:             value.Get("bumped").Bool(),
			BumpedAt:           value.Get("bumped_at").Time(),
			Archetype:          value.Get("archetype").Str,
			Unseen:             value.Get("unseen").Bool(),
			LastReadPostNumber: int(value.Get("last_read_post_number").Int()),
			Unread:             int(value.Get("unread").Int()),
			NewPosts:           int(value.Get("new_posts").Int()),
			UnreadPosts:        int(value.Get("unread_posts").Int()),
			Pinned:             value.Get("pinned").Bool(),
			Visible:            value.Get("visible").Bool(),
			Closed:             value.Get("closed").Bool(),
			Archived:           value.Get("archived").Bool(),
			NotificationLevel:  int(value.Get("notification_level").Int()),
			Bookmarked:         value.Get("bookmarked").Bool(),
			Liked:              value.Get("liked").Bool(),
			Views:              int(value.Get("views").Int()),
			LikeCount:          int(value.Get("like_count").Int()),
			LastPosterUsername: value.Get("last_poster_username").Str,
			CategoryID:         int(value.Get("category_id").Int()),
		}

		// Parse tags
		tags := value.Get("tags")
		tags.ForEach(func(_, tag gjson.Result) bool {
			topic.Tags = append(topic.Tags, tag.Str)
			return true
		})

		response.TopicList.Topics = append(response.TopicList.Topics, topic)
		return true
	})

	categories, err := c.GetCategories()
	if err != nil {
		log.Printf("Warning: failed to fetch categories: %v", err)
	} else {
		categoryMap := make(map[int]struct {
			Name  string
			Color string
		})
		for _, category := range categories.CategoryList.Categories {
			categoryMap[category.ID] = struct {
				Name  string
				Color string
			}{
				Name:  category.Name,
				Color: category.Color,
			}
		}

		for i := range response.TopicList.Topics {
			if cat, ok := categoryMap[response.TopicList.Topics[i].CategoryID]; ok {
				response.TopicList.Topics[i].CategoryName = cat.Name
				response.TopicList.Topics[i].CategoryColor = cat.Color
			}
		}
	}

	return response, nil
}

func (c *Client) GetTopicPosts(topicID int) (*TopicResponse, error) {
	// Fetch initial data to collect all post IDs
	resp, err := c.client.Get(fmt.Sprintf("%s/t/%d.json", c.baseURL, topicID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial topic data: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error fetching initial topic data: %s - %s", resp.Status, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial topic response body: %w", err)
	}
	initial := gjson.ParseBytes(data)

	// Collect post IDs
	idsResult := initial.Get("post_stream.stream")
	var postIDs []int
	idsResult.ForEach(func(_, idVal gjson.Result) bool {
		postIDs = append(postIDs, int(idVal.Int()))
		return true
	})

	// If no IDs, parse posts directly and return
	if len(postIDs) == 0 {
		response := &TopicResponse{}
		posts := initial.Get("post_stream.posts")
		posts.ForEach(func(_, value gjson.Result) bool {
			post := Post{
				ID:         int(value.Get("id").Int()),
				Name:       value.Get("name").Str,
				Username:   value.Get("username").Str,
				CreatedAt:  value.Get("created_at").Time(),
				Cooked:     value.Get("cooked").Str,
				PostNumber: int(value.Get("post_number").Int()),
				ReplyCount: int(value.Get("reply_count").Int()),
				TopicID:    int(value.Get("topic_id").Int()),
				TopicSlug:  value.Get("topic_slug").Str,
				Reads:      int(value.Get("reads").Int()),
				Score:      value.Get("score").Float(),
			}
			actions := value.Get("actions_summary")
			actions.ForEach(func(_, a gjson.Result) bool {
				action := ActionsSummary{
					ID:      int(a.Get("id").Int()),
					Count:   int(a.Get("count").Int()),
					Acted:   a.Get("acted").Bool(),
					CanUndo: a.Get("can_undo").Bool(),
				}
				post.ActionsSummary = append(post.ActionsSummary, action)
				return true
			})
			response.PostStream.Posts = append(response.PostStream.Posts, post)
			return true
		})
		return response, nil
	}

	// Throttle before fetching all posts
	time.Sleep(c.pageCooldown)

	// Fetch all posts by ID
	allURL := fmt.Sprintf("%s/t/%d/posts.json", c.baseURL, topicID)
	req, err := http.NewRequest("GET", allURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create full posts request: %w", err)
	}
	q := req.URL.Query()
	for _, id := range postIDs {
		q.Add("post_ids[]", fmt.Sprintf("%d", id))
	}
	q.Add("include_suggested", "false")
	req.URL.RawQuery = q.Encode()

	fullResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch full posts: %w", err)
	}
	defer fullResp.Body.Close()
	if fullResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(fullResp.Body)
		return nil, fmt.Errorf("API error fetching full posts: %s - %s", fullResp.Status, string(body))
	}
	fullData, err := io.ReadAll(fullResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read full posts response: %w", err)
	}
	result := gjson.ParseBytes(fullData)
	response := &TopicResponse{}
	postsArray := result.Get("post_stream.posts")
	postsArray.ForEach(func(_, value gjson.Result) bool {
		post := Post{
			ID:         int(value.Get("id").Int()),
			Name:       value.Get("name").Str,
			Username:   value.Get("username").Str,
			CreatedAt:  value.Get("created_at").Time(),
			Cooked:     value.Get("cooked").Str,
			PostNumber: int(value.Get("post_number").Int()),
			ReplyCount: int(value.Get("reply_count").Int()),
			TopicID:    int(value.Get("topic_id").Int()),
			TopicSlug:  value.Get("topic_slug").Str,
			Reads:      int(value.Get("reads").Int()),
			Score:      value.Get("score").Float(),
		}
		actions := value.Get("actions_summary")
		actions.ForEach(func(_, a gjson.Result) bool {
			action := ActionsSummary{
				ID:      int(a.Get("id").Int()),
				Count:   int(a.Get("count").Int()),
				Acted:   a.Get("acted").Bool(),
				CanUndo: a.Get("can_undo").Bool(),
			}
			post.ActionsSummary = append(post.ActionsSummary, action)
			return true
		})
		response.PostStream.Posts = append(response.PostStream.Posts, post)
		return true
	})
	return response, nil
}

func (c *Client) GetTopicPostsPage(topicID, page int) (*TopicResponse, error) {
	if page != 1 {
		// Only initial page supported; fall back to full fetch
		return c.GetTopicPosts(topicID)
	}
	resp, err := c.client.Get(fmt.Sprintf("%s/t/%d.json", c.baseURL, topicID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial topic page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error fetching initial topic page: %s - %s", resp.Status, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial topic page: %w", err)
	}
	result := gjson.ParseBytes(data)
	response := &TopicResponse{}
	posts := result.Get("post_stream.posts")
	posts.ForEach(func(_, value gjson.Result) bool {
		post := Post{
			ID:         int(value.Get("id").Int()),
			Name:       value.Get("name").Str,
			Username:   value.Get("username").Str,
			CreatedAt:  value.Get("created_at").Time(),
			Cooked:     value.Get("cooked").Str,
			PostNumber: int(value.Get("post_number").Int()),
			ReplyCount: int(value.Get("reply_count").Int()),
			TopicID:    int(value.Get("topic_id").Int()),
			TopicSlug:  value.Get("topic_slug").Str,
			Reads:      int(value.Get("reads").Int()),
			Score:      value.Get("score").Float(),
		}
		actions := value.Get("actions_summary")
		actions.ForEach(func(_, a gjson.Result) bool {
			action := ActionsSummary{
				ID:      int(a.Get("id").Int()),
				Count:   int(a.Get("count").Int()),
				Acted:   a.Get("acted").Bool(),
				CanUndo: a.Get("can_undo").Bool(),
			}
			post.ActionsSummary = append(post.ActionsSummary, action)
			return true
		})
		response.PostStream.Posts = append(response.PostStream.Posts, post)
		return true
	})
	return response, nil
}

func (c *Client) GetCSRFToken() (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/session/csrf", c.baseURL), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create CSRF request: %v", err)
	}

	req.Header.Set("accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("discourse-present", "true")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch CSRF token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get CSRF token: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	result := gjson.ParseBytes(body)
	csrf := result.Get("csrf").Str
	if csrf == "" {
		return "", fmt.Errorf("empty CSRF token in response")
	}

	return csrf, nil
}

func (c *Client) Login(username, password string) error {
	csrfToken, err := c.GetCSRFToken()
	if err != nil {
		return fmt.Errorf("failed to get CSRF token: %v", err)
	}

	data := url.Values{}
	data.Set("login", username)
	data.Set("password", password)
	data.Set("authenticity_token", csrfToken)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/session", c.baseURL), strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s - %s", resp.Status, string(body))
	}

	if err := c.SaveCookies(c.cookiesPath); err != nil {
		return fmt.Errorf("failed to save cookies after login: %v", err)
	}

	return nil
}

func (c *Client) SaveCookies(cookieFile string) error {
	parsedURL, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %v", err)
	}

	cookies := c.client.Jar.Cookies(parsedURL)
	if len(cookies) == 0 {
		return fmt.Errorf("no cookies to save")
	}

	var cookieStrings []string
	for _, cookie := range cookies {
		cookieStrings = append(cookieStrings, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}

	return os.WriteFile(cookieFile, []byte(strings.Join(cookieStrings, "\n")), 0600) //nosec G306
}

func (c *Client) RefreshTopics() (*Response, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/latest.json", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest topics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Printf("Warning: failed to get cache directory: %v", err)
	} else {
		instanceDir := filepath.Join(userCacheDir, "discourse-tui-client", "instances", strings.TrimPrefix(strings.TrimPrefix(c.baseURL, "https://"), "http://"))
		if err := os.MkdirAll(instanceDir, 0750); err != nil {
			log.Printf("Warning: failed to create instance cache directory: %v", err)
		} else {
			cachePath := filepath.Join(instanceDir, "latest.json")
			if err := os.WriteFile(cachePath, body, 0600); err != nil { //nosec G306
				log.Printf("Warning: failed to save JSON to file: %v", err)
			}
		}
	}

	result := gjson.ParseBytes(body)
	response := &Response{}

	users := result.Get("users")
	users.ForEach(func(_, value gjson.Result) bool {
		user := User{
			ID:             int(value.Get("id").Int()),
			Username:       value.Get("username").Str,
			Name:           value.Get("name").Str,
			AvatarTemplate: value.Get("avatar_template").Str,
			TrustLevel:     int(value.Get("trust_level").Int()),
			Moderator:      value.Get("moderator").Bool(),
		}
		response.Users = append(response.Users, user)
		return true
	})

	// Parse topic list
	topicList := result.Get("topic_list")
	response.TopicList.CanCreateTopic = topicList.Get("can_create_topic").Bool()
	response.TopicList.MoreTopicsURL = topicList.Get("more_topics_url").Str
	response.TopicList.PerPage = int(topicList.Get("per_page").Int())

	// Parse topics
	topics := topicList.Get("topics")
	topics.ForEach(func(_, value gjson.Result) bool {
		topic := Topic{
			ID:                 int(value.Get("id").Int()),
			Title:              value.Get("title").Str,
			FancyTitle:         value.Get("fancy_title").Str,
			Slug:               value.Get("slug").Str,
			PostsCount:         int(value.Get("posts_count").Int()),
			ReplyCount:         int(value.Get("reply_count").Int()),
			HighestPostNumber:  int(value.Get("highest_post_number").Int()),
			ImageURL:           value.Get("image_url").Str,
			CreatedAt:          value.Get("created_at").Time(),
			LastPostedAt:       value.Get("last_posted_at").Time(),
			Bumped:             value.Get("bumped").Bool(),
			BumpedAt:           value.Get("bumped_at").Time(),
			Archetype:          value.Get("archetype").Str,
			Unseen:             value.Get("unseen").Bool(),
			LastReadPostNumber: int(value.Get("last_read_post_number").Int()),
			Unread:             int(value.Get("unread").Int()),
			NewPosts:           int(value.Get("new_posts").Int()),
			UnreadPosts:        int(value.Get("unread_posts").Int()),
			Pinned:             value.Get("pinned").Bool(),
			Visible:            value.Get("visible").Bool(),
			Closed:             value.Get("closed").Bool(),
			Archived:           value.Get("archived").Bool(),
			NotificationLevel:  int(value.Get("notification_level").Int()),
			Bookmarked:         value.Get("bookmarked").Bool(),
			Liked:              value.Get("liked").Bool(),
			Views:              int(value.Get("views").Int()),
			LikeCount:          int(value.Get("like_count").Int()),
			LastPosterUsername: value.Get("last_poster_username").Str,
			CategoryID:         int(value.Get("category_id").Int()),
		}

		tags := value.Get("tags")
		tags.ForEach(func(_, tag gjson.Result) bool {
			topic.Tags = append(topic.Tags, tag.Str)
			return true
		})

		response.TopicList.Topics = append(response.TopicList.Topics, topic)
		return true
	})

	return response, nil
}

func (c *Client) GetCategories() (*CategoryResponse, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %v", err)
	}

	instanceDir := filepath.Join(userCacheDir, "discourse-tui-client", "instances", strings.TrimPrefix(strings.TrimPrefix(c.baseURL, "https://"), "http://"))
	cachePath := filepath.Join(instanceDir, "categories.json")

	// #nosec G304
	if data, err := os.ReadFile(cachePath); err == nil {
		result := gjson.ParseBytes(data)
		response := &CategoryResponse{}

		categories := result.Get("category_list.categories")
		categories.ForEach(func(_, value gjson.Result) bool {
			category := Category{
				ID:          int(value.Get("id").Int()),
				Name:        value.Get("name").Str,
				Color:       value.Get("color").Str,
				TextColor:   value.Get("text_color").Str,
				Slug:        value.Get("slug").Str,
				TopicCount:  int(value.Get("topic_count").Int()),
				PostCount:   int(value.Get("post_count").Int()),
				Position:    int(value.Get("position").Int()),
				Description: value.Get("description").Str,
			}
			response.CategoryList.Categories = append(response.CategoryList.Categories, category)
			return true
		})

		response.CategoryList.CanCreateCategory = result.Get("category_list.can_create_category").Bool()
		response.CategoryList.CanCreateTopic = result.Get("category_list.can_create_topic").Bool()

		return response, nil
	}

	resp, err := c.client.Get(fmt.Sprintf("%s/categories.json", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if err := os.MkdirAll(instanceDir, 0750); err != nil {
		log.Printf("Warning: failed to create instance cache directory: %v", err)
	} else {
		if err := os.WriteFile(cachePath, body, 0600); err != nil {
			log.Printf("Warning: failed to save categories to cache: %v", err)
		}
	}

	result := gjson.ParseBytes(body)
	response := &CategoryResponse{}

	categories := result.Get("category_list.categories")
	categories.ForEach(func(_, value gjson.Result) bool {
		category := Category{
			ID:          int(value.Get("id").Int()),
			Name:        value.Get("name").Str,
			Color:       value.Get("color").Str,
			TextColor:   value.Get("text_color").Str,
			Slug:        value.Get("slug").Str,
			TopicCount:  int(value.Get("topic_count").Int()),
			PostCount:   int(value.Get("post_count").Int()),
			Position:    int(value.Get("position").Int()),
			Description: value.Get("description").Str,
		}
		response.CategoryList.Categories = append(response.CategoryList.Categories, category)
		return true
	})

	response.CategoryList.CanCreateCategory = result.Get("category_list.can_create_category").Bool()
	response.CategoryList.CanCreateTopic = result.Get("category_list.can_create_topic").Bool()

	return response, nil
}

func (c *Client) PerformPostAction(postID int, postActionTypeID int, flagTopic bool) (*Post, error) {
	csrfToken, err := c.GetCSRFToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get CSRF token for post action: %w", err)
	}

	data := url.Values{}
	data.Set("id", fmt.Sprintf("%d", postID))
	data.Set("post_action_type_id", fmt.Sprintf("%d", postActionTypeID))
	data.Set("flag_topic", fmt.Sprintf("%t", flagTopic))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/post_actions", c.baseURL), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create post action request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform post action: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read post action response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post action API error: %s - %s", resp.Status, string(body))
	}

	// The response is the updated post object
	result := gjson.ParseBytes(body)
	post := Post{
		ID:         int(result.Get("id").Int()),
		Name:       result.Get("name").Str,
		Username:   result.Get("username").Str,
		CreatedAt:  result.Get("created_at").Time(),
		Cooked:     result.Get("cooked").Str,
		PostNumber: int(result.Get("post_number").Int()),
		ReplyCount: int(result.Get("reply_count").Int()),
		TopicID:    int(result.Get("topic_id").Int()),
		TopicSlug:  result.Get("topic_slug").Str,
		Reads:      int(result.Get("reads").Int()),
		Score:      result.Get("score").Float(),
	}

	actionsSummaryResult := result.Get("actions_summary")
	actionsSummaryResult.ForEach(func(_, actionData gjson.Result) bool {
		action := ActionsSummary{
			ID:      int(actionData.Get("id").Int()),
			Count:   int(actionData.Get("count").Int()),
			Acted:   actionData.Get("acted").Bool(),
			CanUndo: actionData.Get("can_undo").Bool(),
		}
		post.ActionsSummary = append(post.ActionsSummary, action)
		return true
	})

	return &post, nil
}

func (c *Client) CreateTopic(title, rawContent string, categoryID int, tags []string) (*Post, error) {
	csrfToken, err := c.GetCSRFToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get CSRF token for creating topic: %w", err)
	}

	payload := apiCreateTopicPayload{
		Title:     title,
		Raw:       rawContent,
		Category:  categoryID,
		Tags:      tags,
		Archetype: "regular",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create topic payload: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/posts.json", c.baseURL), bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create new topic request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create topic request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read create topic response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("create topic API error: %s (status code: %d) - %s", resp.Status, resp.StatusCode, string(body))
	}

	var createdPost Post
	if err := json.Unmarshal(body, &createdPost); err != nil {
		log.Printf("Error unmarshalling created topic/post response body: %v. Body: %s", err, string(body))
		return nil, fmt.Errorf("failed to parse create topic response (body: %s): %w", string(body), err)
	}

	if createdPost.ID == 0 {
		log.Printf("Created post has ID 0. Body: %s", string(body))
		return nil, fmt.Errorf("created post has ID 0, which is invalid (body: %s)", string(body))
	}

	return &createdPost, nil
}

func (c *Client) SetPageCooldown(d time.Duration) {
	c.pageCooldown = d
}

func (c *Client) GetMoreTopics(moreURL string) (*Response, error) {
	if moreURL == "" {
		return nil, fmt.Errorf("no more topics URL provided")
	}

	var fullURL string
	if strings.HasPrefix(moreURL, "http") {
		fullURL = moreURL
	} else {
		fullURL = c.baseURL + moreURL
	}

	resp, err := c.client.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch more topics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	result := gjson.ParseBytes(body)
	response := &Response{}

	users := result.Get("users")
	users.ForEach(func(_, value gjson.Result) bool {
		user := User{
			ID:             int(value.Get("id").Int()),
			Username:       value.Get("username").Str,
			Name:           value.Get("name").Str,
			AvatarTemplate: value.Get("avatar_template").Str,
			TrustLevel:     int(value.Get("trust_level").Int()),
			Moderator:      value.Get("moderator").Bool(),
		}
		response.Users = append(response.Users, user)
		return true
	})

	topicList := result.Get("topic_list")
	response.TopicList.CanCreateTopic = topicList.Get("can_create_topic").Bool()
	response.TopicList.MoreTopicsURL = topicList.Get("more_topics_url").Str
	response.TopicList.PerPage = int(topicList.Get("per_page").Int())

	topics := topicList.Get("topics")
	topics.ForEach(func(_, value gjson.Result) bool {
		topic := Topic{
			ID:                 int(value.Get("id").Int()),
			Title:              value.Get("title").Str,
			FancyTitle:         value.Get("fancy_title").Str,
			Slug:               value.Get("slug").Str,
			PostsCount:         int(value.Get("posts_count").Int()),
			ReplyCount:         int(value.Get("reply_count").Int()),
			HighestPostNumber:  int(value.Get("highest_post_number").Int()),
			ImageURL:           value.Get("image_url").Str,
			CreatedAt:          value.Get("created_at").Time(),
			LastPostedAt:       value.Get("last_posted_at").Time(),
			Bumped:             value.Get("bumped").Bool(),
			BumpedAt:           value.Get("bumped_at").Time(),
			Archetype:          value.Get("archetype").Str,
			Unseen:             value.Get("unseen").Bool(),
			LastReadPostNumber: int(value.Get("last_read_post_number").Int()),
			Unread:             int(value.Get("unread").Int()),
			NewPosts:           int(value.Get("new_posts").Int()),
			UnreadPosts:        int(value.Get("unread_posts").Int()),
			Pinned:             value.Get("pinned").Bool(),
			Visible:            value.Get("visible").Bool(),
			Closed:             value.Get("closed").Bool(),
			Archived:           value.Get("archived").Bool(),
			NotificationLevel:  int(value.Get("notification_level").Int()),
			Bookmarked:         value.Get("bookmarked").Bool(),
			Liked:              value.Get("liked").Bool(),
			Views:              int(value.Get("views").Int()),
			LikeCount:          int(value.Get("like_count").Int()),
			LastPosterUsername: value.Get("last_poster_username").Str,
			CategoryID:         int(value.Get("category_id").Int()),
		}

		tags := value.Get("tags")
		tags.ForEach(func(_, tag gjson.Result) bool {
			topic.Tags = append(topic.Tags, tag.Str)
			return true
		})

		response.TopicList.Topics = append(response.TopicList.Topics, topic)
		return true
	})

	categories, err := c.GetCategories()
	if err != nil {
		log.Printf("Warning: failed to fetch categories: %v", err)
	} else {
		categoryMap := make(map[int]struct {
			Name  string
			Color string
		})
		for _, category := range categories.CategoryList.Categories {
			categoryMap[category.ID] = struct {
				Name  string
				Color string
			}{
				Name:  category.Name,
				Color: category.Color,
			}
		}

		for i := range response.TopicList.Topics {
			if cat, ok := categoryMap[response.TopicList.Topics[i].CategoryID]; ok {
				response.TopicList.Topics[i].CategoryName = cat.Name
				response.TopicList.Topics[i].CategoryColor = cat.Color
			}
		}
	}

	return response, nil
}

func (c *Client) LoadAllTopics(maxPages int) (*Response, error) {
	if maxPages <= 0 {
		maxPages = 10
	}

	initialResp, err := c.GetLatestTopics()
	if err != nil {
		return nil, fmt.Errorf("failed to get initial topics: %v", err)
	}

	allTopics := initialResp.TopicList.Topics
	allUsers := initialResp.Users
	currentMoreURL := initialResp.TopicList.MoreTopicsURL

	for page := 1; page < maxPages && currentMoreURL != ""; page++ {
		time.Sleep(c.pageCooldown)

		moreResp, err := c.GetMoreTopics(currentMoreURL)
		if err != nil {
			log.Printf("Warning: failed to fetch page %d: %v", page+1, err)
			break
		}

		allTopics = append(allTopics, moreResp.TopicList.Topics...)
		allUsers = append(allUsers, moreResp.Users...)
		currentMoreURL = moreResp.TopicList.MoreTopicsURL

		if len(moreResp.TopicList.Topics) == 0 {
			break
		}
	}

	result := &Response{
		Users:         allUsers,
		PrimaryGroups: initialResp.PrimaryGroups,
		FlairGroups:   initialResp.FlairGroups,
		TopicList: TopicList{
			CanCreateTopic: initialResp.TopicList.CanCreateTopic,
			MoreTopicsURL:  currentMoreURL,
			PerPage:        initialResp.TopicList.PerPage,
			TopTags:        initialResp.TopicList.TopTags,
			Topics:         allTopics,
		},
	}

	return result, nil
}
