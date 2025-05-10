// Copyright (c) 2025 Sudo-Ivan
// MIT License

package discourse

import (
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
	ID                  int       `json:"id"`
	Title               string    `json:"title"`
	FancyTitle          string    `json:"fancy_title"`
	Slug                string    `json:"slug"`
	PostsCount          int       `json:"posts_count"`
	ReplyCount          int       `json:"reply_count"`
	HighestPostNumber   int       `json:"highest_post_number"`
	ImageURL            string    `json:"image_url"`
	CreatedAt           time.Time `json:"created_at"`
	LastPostedAt        time.Time `json:"last_posted_at"`
	Bumped              bool      `json:"bumped"`
	BumpedAt            time.Time `json:"bumped_at"`
	Archetype           string    `json:"archetype"`
	Unseen              bool      `json:"unseen"`
	LastReadPostNumber  int       `json:"last_read_post_number"`
	Unread              int       `json:"unread"`
	NewPosts            int       `json:"new_posts"`
	UnreadPosts         int       `json:"unread_posts"`
	Pinned              bool      `json:"pinned"`
	Unpinned            *bool     `json:"unpinned"`
	Visible             bool      `json:"visible"`
	Closed              bool      `json:"closed"`
	Archived            bool      `json:"archived"`
	NotificationLevel   int       `json:"notification_level"`
	Bookmarked          bool      `json:"bookmarked"`
	Liked               bool      `json:"liked"`
	Tags                []string  `json:"tags"`
	Views               int       `json:"views"`
	LikeCount           int       `json:"like_count"`
	LastPosterUsername  string    `json:"last_poster_username"`
	CategoryID          int       `json:"category_id"`
	CategoryName        string    `json:"category_name"`
	CategoryColor       string    `json:"category_color"`
}

type TopicList struct {
	CanCreateTopic bool     `json:"can_create_topic"`
	MoreTopicsURL  string   `json:"more_topics_url"`
	PerPage        int      `json:"per_page"`
	TopTags        []string `json:"top_tags"`
	Topics         []Topic  `json:"topics"`
}

type Response struct {
	Users         []User     `json:"users"`
	PrimaryGroups []string   `json:"primary_groups"`
	FlairGroups   []string   `json:"flair_groups"`
	TopicList     TopicList  `json:"topic_list"`
}

type Post struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	CreatedAt   time.Time `json:"created_at"`
	Cooked      string    `json:"cooked"`
	PostNumber  int       `json:"post_number"`
	ReplyCount  int       `json:"reply_count"`
	TopicID     int       `json:"topic_id"`
	TopicSlug   string    `json:"topic_slug"`
	Reads       int       `json:"reads"`
	Score       float64   `json:"score"`
}

type PostStream struct {
	Posts []Post `json:"posts"`
	Stream []int `json:"stream"`
}

type TopicResponse struct {
	PostStream PostStream `json:"post_stream"`
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
	Topics      []Topic `json:"topics"`
}

type CategoryList struct {
	CanCreateCategory bool       `json:"can_create_category"`
	CanCreateTopic   bool       `json:"can_create_topic"`
	Categories       []Category `json:"categories"`
}

type CategoryResponse struct {
	CategoryList CategoryList `json:"category_list"`
}

type Client struct {
	client      *http.Client
	baseURL     string
	cookiesPath string
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
		client:      client,
		baseURL:     baseURL,
		cookiesPath: cookiesPath,
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

	// Fetch and map categories
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
	// Step 1: Fetch initial topic data to get post IDs
	initialResp, err := c.client.Get(fmt.Sprintf("%s/t/%d.json", c.baseURL, topicID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial topic data: %w", err)
	}
	defer initialResp.Body.Close()

	if initialResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(initialResp.Body)
		return nil, fmt.Errorf("API error fetching initial topic data: %s - %s", initialResp.Status, string(body))
	}

	initialBody, err := io.ReadAll(initialResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial topic response body: %w", err)
	}

	initialResult := gjson.ParseBytes(initialBody)
	postStreamIDsResult := initialResult.Get("post_stream.stream")
	if !postStreamIDsResult.Exists() {
		return nil, fmt.Errorf("post_stream.stream not found in initial topic response")
	}

	var postIDs []int
	for _, idEntry := range postStreamIDsResult.Array() {
		postIDs = append(postIDs, int(idEntry.Int()))
	}

	if len(postIDs) == 0 {
		// If there are no post IDs, return an empty response or handle as appropriate
		// For now, we can assume the existing behavior for topics with no posts from stream
		// or return an empty PostStream.
		// Let's try to parse the initial body as if it might contain posts directly (for very short topics)
		response := &TopicResponse{}
		posts := initialResult.Get("post_stream.posts")
		posts.ForEach(func(_, value gjson.Result) bool {
			post := Post{
				ID:          int(value.Get("id").Int()),
				Name:        value.Get("name").Str,
				Username:    value.Get("username").Str,
				CreatedAt:   value.Get("created_at").Time(),
				Cooked:      value.Get("cooked").Str,
				PostNumber:  int(value.Get("post_number").Int()),
				ReplyCount:  int(value.Get("reply_count").Int()),
				TopicID:     int(value.Get("topic_id").Int()),
				TopicSlug:   value.Get("topic_slug").Str,
				Reads:       int(value.Get("reads").Int()),
				Score:       value.Get("score").Float(),
			}
			response.PostStream.Posts = append(response.PostStream.Posts, post)
			return true
		})
		return response, nil
	}

	// Step 2: Construct URL with all post_ids
	postsURL := fmt.Sprintf("%s/t/%d/posts.json", c.baseURL, topicID)
	req, err := http.NewRequest("GET", postsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for all posts: %w", err)
	}

	q := req.URL.Query()
	for _, postID := range postIDs {
		q.Add("post_ids[]", fmt.Sprintf("%d", postID))
	}
	// As seen in the user's cURL:
	q.Add("include_suggested", "false")
	req.URL.RawQuery = q.Encode()

	// Step 3: Fetch all posts
	allPostsResp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all topic posts: %w", err)
	}
	defer allPostsResp.Body.Close()

	if allPostsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(allPostsResp.Body)
		return nil, fmt.Errorf("API error fetching all posts: %s - %s", allPostsResp.Status, string(body))
	}

	allPostsBody, err := io.ReadAll(allPostsResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read all posts response body: %w", err)
	}

	// Step 4: Parse posts from the response
	result := gjson.ParseBytes(allPostsBody)
	response := &TopicResponse{}

	posts := result.Get("post_stream.posts") // Assuming the structure is the same
	posts.ForEach(func(_, value gjson.Result) bool {
		post := Post{
			ID:          int(value.Get("id").Int()),
			Name:        value.Get("name").Str,
			Username:    value.Get("username").Str,
			CreatedAt:   value.Get("created_at").Time(),
			Cooked:      value.Get("cooked").Str,
			PostNumber:  int(value.Get("post_number").Int()),
			ReplyCount:  int(value.Get("reply_count").Int()),
			TopicID:     int(value.Get("topic_id").Int()),
			TopicSlug:   value.Get("topic_slug").Str,
			Reads:       int(value.Get("reads").Int()),
			Score:       value.Get("score").Float(),
		}
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

	return response, nil
}

func (c *Client) GetCategories() (*CategoryResponse, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %v", err)
	}

	instanceDir := filepath.Join(userCacheDir, "discourse-tui-client", "instances", strings.TrimPrefix(strings.TrimPrefix(c.baseURL, "https://"), "http://"))
	cachePath := filepath.Join(instanceDir, "categories.json")
	
	if data, err := os.ReadFile(cachePath); err == nil {
		result := gjson.ParseBytes(data)
		response := &CategoryResponse{}
		
		// Parse categories
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

	// Parse categories
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