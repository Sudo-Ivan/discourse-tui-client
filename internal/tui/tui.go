// Copyright (c) 2025 Sudo-Ivan
// MIT License

package tui

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/microcosm-cc/bluemonday"

	"github.com/Sudo-Ivan/discourse-tui-client/internal/config"
	"github.com/Sudo-Ivan/discourse-tui-client/pkg/discourse"
)

type topicItem struct {
	topic discourse.Topic
}

func (i topicItem) Title() string {
	var title strings.Builder
	title.WriteString(i.topic.Title)
	
	if i.topic.CategoryName != "" {
		title.WriteString(" [")
		title.WriteString(i.topic.CategoryName)
		title.WriteString("]")
	}
	
	if len(i.topic.Tags) > 0 {
		title.WriteString(" {")
		title.WriteString(strings.Join(i.topic.Tags, ", "))
		title.WriteString("}")
	}
	
	return title.String()
}

func (i topicItem) Description() string {
	return fmt.Sprintf("%d replies • %d views", i.topic.ReplyCount, i.topic.Views)
}

func (i topicItem) FilterValue() string { return i.topic.Title }

type Model struct {
	List       list.Model
	Viewport   viewport.Model
	Client     *discourse.Client
	Topics     []discourse.Topic
	Ready      bool
	Fullscreen bool
	Search     textinput.Model
	Searching  bool
	LastRefresh time.Time
	Width      int
	Height     int
	InstanceURL string
}

func InitialModel(client *discourse.Client, topics []discourse.Topic) Model {
	items := make([]list.Item, len(topics))
	for i, topic := range topics {
		items[i] = topicItem{topic: topic}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = config.SelectedItemStyle
	delegate.Styles.SelectedDesc = config.SelectedItemStyle
	delegate.Styles.NormalTitle = config.ItemStyle
	delegate.Styles.NormalDesc = config.ItemStyle
	delegate.SetHeight(2)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Latest Topics"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = config.TitleStyle
	l.Styles.FilterPrompt = config.StatusStyle
	l.Styles.FilterCursor = config.StatusStyle.Copy().Foreground(lipgloss.Color("170"))
	l.SetShowHelp(true)

	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	search := textinput.New()
	search.Placeholder = "Search topics..."
	search.Width = 30

	instanceURL := strings.TrimPrefix(strings.TrimPrefix(client.BaseURL(), "https://"), "http://")

	return Model{
		List:      l,
		Viewport:  vp,
		Client:    client,
		Topics:    topics,
		Search:    search,
		LastRefresh: time.Now(),
		InstanceURL: instanceURL,
	}
}

func (m Model) Init() tea.Cmd {
	log.Printf("Initializing model with %d topics", len(m.Topics))
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return refreshMsg{}
	})
}

type refreshMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case refreshMsg:
		response, err := m.Client.RefreshTopics()
		if err != nil {
			log.Printf("Failed to refresh topics: %v", err)
			return m, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
				return refreshMsg{}
			})
		}

		categories, err := m.Client.GetCategories()
		if err != nil {
			log.Printf("Failed to fetch categories: %v", err)
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

		items := make([]list.Item, len(response.TopicList.Topics))
		for i, topic := range response.TopicList.Topics {
			items[i] = topicItem{topic: topic}
		}
		m.List.SetItems(items)
		m.Topics = response.TopicList.Topics
		m.LastRefresh = time.Now()
		return m, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
			return refreshMsg{}
		})
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "f":
			m.Fullscreen = !m.Fullscreen
			if m.Fullscreen {
				m.Viewport.Width = m.Width
				m.Viewport.Height = m.Height
			} else {
				listHeight := m.Height / 2
				m.List.SetWidth(m.Width)
				m.List.SetHeight(listHeight)
				m.Viewport.Width = m.Width
				m.Viewport.Height = m.Height - listHeight - 1
			}
			return m, nil
		case "/":
			m.Searching = !m.Searching
			if m.Searching {
				return m, m.Search.Focus()
			}
			return m, nil
		case "R":
			response, err := m.Client.RefreshTopics()
			if err != nil {
				log.Printf("Failed to refresh topics: %v", err)
				return m, nil
			}

			categories, err := m.Client.GetCategories()
			if err != nil {
				log.Printf("Failed to fetch categories: %v", err)
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

			items := make([]list.Item, len(response.TopicList.Topics))
			for i, topic := range response.TopicList.Topics {
				items[i] = topicItem{topic: topic}
			}
			m.List.SetItems(items)
			m.Topics = response.TopicList.Topics
			m.LastRefresh = time.Now()
			return m, nil
		case "esc":
			if m.Searching {
				m.Searching = false
				return m, nil
			}
			if m.Fullscreen {
				m.Fullscreen = false
				return m, nil
			}
		case "enter":
			if m.Searching {
				query := m.Search.Value()
				if query != "" {
					var filteredTopics []discourse.Topic
					for _, topic := range m.Topics {
						if strings.Contains(strings.ToLower(topic.Title), strings.ToLower(query)) {
							filteredTopics = append(filteredTopics, topic)
						}
					}
					items := make([]list.Item, len(filteredTopics))
					for i, topic := range filteredTopics {
						items[i] = topicItem{topic: topic}
					}
					m.List.SetItems(items)
					m.Searching = false
				}
				return m, nil
			}
			if i, ok := m.List.SelectedItem().(topicItem); ok {
				posts, err := m.Client.GetTopicPosts(i.topic.ID)
				if err != nil {
					errorContentWidth := m.Viewport.Width - 2
					if errorContentWidth < 1 {
						errorContentWidth = 1
					}
					errorStyle := lipgloss.NewStyle().Width(errorContentWidth)
					m.Viewport.SetContent(errorStyle.Render(fmt.Sprintf("Error fetching posts: %v", err)))
					return m, nil
				}

				var content strings.Builder
				postContentWidth := m.Viewport.Width - 2
				if postContentWidth < 1 {
					postContentWidth = 1
				}

				for _, post := range posts.PostStream.Posts {
					content.WriteString(FormatPost(post, postContentWidth))
					content.WriteString("\n\n---\n\n")
				}

				m.Viewport.SetContent(content.String())
			}
		}
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		
		if !m.Ready {
			m.Ready = true
		}

		if m.Fullscreen {
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height
		} else {
			listHeight := msg.Height / 2
			m.List.SetWidth(msg.Width)
			m.List.SetHeight(listHeight)
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height - listHeight - 1
		}
	}

	if m.Searching {
		var cmd tea.Cmd
		m.Search, cmd = m.Search.Update(msg)
		cmds = append(cmds, cmd)
	}

	var listCmd tea.Cmd
	m.List, listCmd = m.List.Update(msg)
	cmds = append(cmds, listCmd)

	var viewportCmd tea.Cmd
	m.Viewport, viewportCmd = m.Viewport.Update(msg)
	cmds = append(cmds, viewportCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.Ready {
		return "\nInitializing..."
	}

	headerHeight := 2
	helpHeight := 2
	availableHeight := m.Height - headerHeight - helpHeight - 2
	listHeight := (availableHeight * 2) / 3
	viewportHeight := availableHeight - listHeight

	instanceHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(m.Width - 2).
		Align(lipgloss.Center).
		Render(m.InstanceURL)

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(fmt.Sprintf("Press 'f' for fullscreen, '/' to search, 'R' to refresh, 'esc' to exit fullscreen/search • Last refresh: %s", m.LastRefresh.Format("15:04:05")))

	if m.Fullscreen {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			instanceHeader,
			lipgloss.NewStyle().
				Width(m.Width).
				Height(m.Height - 4).
				MaxWidth(m.Width).
				MaxHeight(m.Height - 4).
				Render(m.Viewport.View()),
			help,
		)
	}

	m.List.SetWidth(m.Width - 2)
	m.List.SetHeight(listHeight)
	m.Viewport.Width = m.Width - 2
	m.Viewport.Height = viewportHeight

	var view string
	if m.Searching {
		searchBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			Width(m.Width - 2).
			Render(m.Search.View())

		view = lipgloss.JoinVertical(
			lipgloss.Left,
			instanceHeader,
			lipgloss.NewStyle().MarginTop(1).Render(searchBox),
			lipgloss.NewStyle().MarginTop(1).Render(m.List.View()),
			lipgloss.NewStyle().MarginTop(1).Render(m.Viewport.View()),
			help,
		)
	} else {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			instanceHeader,
			lipgloss.NewStyle().MarginTop(1).Render(m.List.View()),
			lipgloss.NewStyle().MarginTop(1).Render(m.Viewport.View()),
			help,
		)
	}

	return view
}

func FormatPost(post discourse.Post, contentWidth int) string {
	p := bluemonday.StrictPolicy()
	sanitizedContent := p.Sanitize(post.Cooked)

	text := strings.ReplaceAll(sanitizedContent, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		}
	}
	text = strings.Join(lines, "\n")

	multipleNewlinesRegex := regexp.MustCompile(`\n{3,}`)
	text = multipleNewlinesRegex.ReplaceAllString(text, "\n\n")

	text = strings.TrimSpace(text)

	if contentWidth < 1 {
		contentWidth = 1
	}
	contentWrappingStyle := lipgloss.NewStyle().Width(contentWidth)

	paragraphsSource := strings.Split(text, "\n\n")
	var renderedParagraphs []string
	for _, paraStr := range paragraphsSource {
		trimmedParaStr := strings.TrimSpace(paraStr)
		if trimmedParaStr != "" {
			renderedBlock := contentWrappingStyle.Render(trimmedParaStr)
			renderedBlock = strings.TrimRight(renderedBlock, "\n") 
			renderedParagraphs = append(renderedParagraphs, renderedBlock)
		}
	}
	wrappedPostBody := strings.Join(renderedParagraphs, "\n\n")

	postHeader := fmt.Sprintf("Post #%d by %s (%s)\nPosted: %s",
		post.PostNumber,
		post.Name,
		post.Username,
		post.CreatedAt.Format("2006-01-02 15:04:05"))

	postFooter := fmt.Sprintf("Reads: %d | Score: %.1f",
		post.Reads,
		post.Score)

	var likeInfo string
	for _, action := range post.ActionsSummary {
		if action.ID == 2 { // 2 is usually the ID for 'like'
			likeCount := action.Count
			if action.Acted {
				likeInfo = fmt.Sprintf("Likes: %d (You liked this)", likeCount)
			} else {
				likeInfo = fmt.Sprintf("Likes: %d", likeCount)
			}
			break
		}
	}

	return strings.Join([]string{
		postHeader,
		"",
		wrappedPostBody,
		"",
		postFooter,
		likeInfo,
	}, "\n")
}

type loginModel struct {
	client     *discourse.Client
	inputs     []textinput.Model
	focusIndex int
	err        error
	done       bool
}

func (m loginModel) GetInstanceURL() string {
	return m.inputs[0].Value()
}

func InitialLoginModel(client *discourse.Client) loginModel {
	url := textinput.New()
	url.Placeholder = "Instance URL (e.g. forum.example.com)"
	url.Focus()
	url.CharLimit = 100
	url.Width = 40

	username := textinput.New()
	username.Placeholder = "Username"
	username.CharLimit = 50
	username.Width = 30

	password := textinput.New()
	password.Placeholder = "Password"
	password.CharLimit = 50
	password.Width = 30
	password.EchoMode = textinput.EchoPassword

	return loginModel{
		client:     client,
		inputs:     []textinput.Model{url, username, password},
		focusIndex: 0,
	}
}

func (m loginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.focusIndex == len(m.inputs)-1 {
				instanceURL := m.inputs[0].Value()
				username := m.inputs[1].Value()
				password := m.inputs[2].Value()

				if instanceURL == "" {
					m.err = fmt.Errorf("instance URL is required")
					return m, nil
				}
				if username == "" {
					m.err = fmt.Errorf("username is required")
					return m, nil
				}
				if password == "" {
					m.err = fmt.Errorf("password is required")
					return m, nil
				}

				newClient, err := discourse.NewClient(instanceURL, m.client.CookiesPath())
				if err != nil {
					m.err = fmt.Errorf("failed to create client: %v", err)
					return m, nil
				}
				m.client = newClient

				if err := m.client.Login(username, password); err != nil {
					m.err = fmt.Errorf("login failed: %v", err)
					return m, nil
				}
				if err := config.SaveInstance(instanceURL); err != nil {
					log.Printf("Failed to save instance URL: %v", err)
				}
				m.done = true
				return m, tea.Quit
			} else {
				m.focusIndex++
				for i := 0; i < len(m.inputs); i++ {
					if i == m.focusIndex {
						cmds = append(cmds, m.inputs[i].Focus())
					} else {
						m.inputs[i].Blur()
					}
				}
			}
		case tea.KeyTab:
			m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}
		case tea.KeyShiftTab:
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}

	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m loginModel) View() string {
	if m.done {
		return "Login successful!\n"
	}

	var s strings.Builder
	s.WriteString(config.TitleStyle.Render("Discourse Login\n\n"))

	for i, input := range m.inputs {
		s.WriteString(input.View())
		if i < len(m.inputs)-1 {
			s.WriteString("\n")
		}
	}

	s.WriteString("\n\n")
	if m.focusIndex == len(m.inputs)-1 {
		s.WriteString(config.SelectedItemStyle.Render("[ Login ]"))
	} else {
		s.WriteString(config.ItemStyle.Render("[ Login ]"))
	}

	if m.err != nil {
		s.WriteString("\n\n")
		s.WriteString(config.ErrorStyle.Render(m.err.Error()))
	}

	s.WriteString("\n\nPress Tab to switch fields, Enter to submit")

	return s.String()
} 