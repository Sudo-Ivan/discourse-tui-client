// Copyright (c) 2025 Sudo-Ivan
// MIT License

package discourse

import (
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

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &response, nil
}

func (c *Client) GetTopicPosts(topicID int) (*TopicResponse, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/t/%d.json", c.baseURL, topicID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch topic posts: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var response TopicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &response, nil
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

	var csrfResp struct {
		CSRF string `json:"csrf"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&csrfResp); err != nil {
		return "", fmt.Errorf("failed to decode CSRF response: %v", err)
	}

	if csrfResp.CSRF == "" {
		return "", fmt.Errorf("empty CSRF token in response")
	}

	return csrfResp.CSRF, nil
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

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &response, nil
}

func (c *Client) GetCategories() (*CategoryResponse, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %v", err)
	}

	instanceDir := filepath.Join(userCacheDir, "discourse-tui-client", "instances", strings.TrimPrefix(strings.TrimPrefix(c.baseURL, "https://"), "http://"))
	cachePath := filepath.Join(instanceDir, "categories.json")
	
	if data, err := os.ReadFile(cachePath); err == nil {
		var response CategoryResponse
		if err := json.Unmarshal(data, &response); err == nil {
			return &response, nil
		}
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

	var response CategoryResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &response, nil
} 