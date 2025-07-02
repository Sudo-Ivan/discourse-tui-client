// Copyright (c) 2025 Sudo-Ivan
// MIT License

package tui

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
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

type modelState int

const (
	stateTopicList modelState = iota
	stateNewTopic
	stateLogin
)

type topicCreatedMsg struct {
	post    *discourse.Post
	message string
}
type topicCreateErrorMsg struct{ err error }

type postsLoadedMsg struct {
	posts *discourse.TopicResponse
}
type postsLoadErrorMsg struct{ err error }

type topicsRefreshedMsg struct {
	response *discourse.Response
}
type topicsRefreshErrorMsg struct{ err error }

type moreTopicsLoadedMsg struct {
	response *discourse.Response
}
type moreTopicsLoadErrorMsg struct{ err error }

type loadAllTopicsMsg struct {
	response *discourse.Response
}
type loadAllTopicsErrorMsg struct{ err error }

type newTopicModel struct {
	client        *discourse.Client
	titleInput    textinput.Model
	contentInput  textarea.Model
	categoryInput textinput.Model
	tagsInput     textinput.Model
	focusIndex    int
	width, height int
	err           error
	submitting    bool
	message       string
}

func InitialNewTopicModel(client *discourse.Client, width, height int) newTopicModel {
	ti := textinput.New()
	ti.Placeholder = "Topic Title"
	ti.Focus()
	ti.CharLimit = 250
	ti.Width = width - 4

	ta := textarea.New()
	ta.Placeholder = "Topic content..."
	ta.SetWidth(width - 4)
	ta.SetHeight(height / 3)

	ci := textinput.New()
	ci.Placeholder = "Category ID (e.g., 10)"
	ci.CharLimit = 10
	ci.Width = width - 4

	tgi := textinput.New()
	tgi.Placeholder = "Tags (comma-separated, e.g., go,tui)"
	tgi.CharLimit = 255
	tgi.Width = width - 4

	n := newTopicModel{
		client:        client,
		titleInput:    ti,
		contentInput:  ta,
		categoryInput: ci,
		tagsInput:     tgi,
		focusIndex:    0,
		width:         width,
		height:        height,
	}
	n.updateFocus()
	return n
}

func (m *newTopicModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *newTopicModel) updateFocus() {
	inputs := []interface{}{
		&m.titleInput,
		&m.contentInput,
		&m.categoryInput,
		&m.tagsInput,
	}

	for i := 0; i < len(inputs); i++ {
		if i == m.focusIndex {
			switch v := inputs[i].(type) {
			case *textinput.Model:
				v.Focus()
			case *textarea.Model:
				v.Focus()
			}
		} else {
			switch v := inputs[i].(type) {
			case *textinput.Model:
				v.Blur()
			case *textarea.Model:
				v.Blur()
			}
		}
	}
}

func (m *newTopicModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	m.err = nil

	if m.submitting {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlS:
			m.submitting = true
			m.message = "Submitting new topic..."
			title := m.titleInput.Value()
			content := m.contentInput.Value()
			categoryStr := m.categoryInput.Value()
			tagsStr := m.tagsInput.Value()

			if title == "" || content == "" || categoryStr == "" {
				m.err = fmt.Errorf("title, content, and category ID are required")
				m.submitting = false
				m.message = ""
				return m, nil
			}

			categoryID, err := strconv.Atoi(categoryStr)
			if err != nil {
				m.err = fmt.Errorf("invalid category ID: %w", err)
				m.submitting = false
				m.message = ""
				return m, nil
			}

			var tags []string
			if strings.TrimSpace(tagsStr) != "" {
				tags = strings.Split(tagsStr, ",")
				for i := range tags {
					tags[i] = strings.TrimSpace(tags[i])
				}
			}

			return m, func() tea.Msg {
				post, err := m.client.CreateTopic(title, content, categoryID, tags)
				if err != nil {
					return topicCreateErrorMsg{err: err}
				}
				return topicCreatedMsg{post: post, message: fmt.Sprintf("Topic '%s' created!", post.TopicSlug)}
			}

		case tea.KeyTab, tea.KeyShiftTab:
			if msg.Type == tea.KeyShiftTab {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > 3 {
				m.focusIndex = 0
			}
			if m.focusIndex < 0 {
				m.focusIndex = 3
			}
			m.updateFocus()
			var blinkCmd tea.Cmd
			if m.focusIndex == 1 {
				blinkCmd = textarea.Blink
			} else {
				blinkCmd = textinput.Blink
			}
			return m, blinkCmd
		}
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 1:
		m.contentInput, cmd = m.contentInput.Update(msg)
	case 2:
		m.categoryInput, cmd = m.categoryInput.Update(msg)
	case 3:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m newTopicModel) View() string {
	var b strings.Builder
	b.WriteString(config.TitleStyle.Render("Create New Topic"))
	b.WriteString("\n\n")
	b.WriteString(m.titleInput.View())
	b.WriteString("\n\n")
	b.WriteString(m.contentInput.View())
	b.WriteString("\n\n")
	b.WriteString(m.categoryInput.View())
	b.WriteString("\n\n")
	b.WriteString(m.tagsInput.View())
	b.WriteString("\n\n")

	if m.submitting {
		b.WriteString(config.StatusStyle.Render(m.message))
	} else if m.err != nil {
		b.WriteString(config.ErrorStyle.Render(m.err.Error()))
	} else if m.message != "" {
		b.WriteString(config.StatusStyle.Render(m.message))
	}

	help := "Tab/Shift+Tab: navigate | Ctrl+S: submit | Esc: cancel"
	b.WriteString("\n\n" + help)

	return b.String()
}

type Model struct {
	List               list.Model
	Viewport           viewport.Model
	Client             *discourse.Client
	Topics             []discourse.Topic
	Ready              bool
	Fullscreen         bool
	Search             textinput.Model
	Searching          bool
	LastRefresh        time.Time
	Width, Height      int
	InstanceURL        string
	State              modelState
	NewTopicForm       newTopicModel
	StatusMessage      string
	isLoadingPosts     bool
	isRefreshingTopics bool
	MoreTopicsURL      string
	isLoadingMore      bool
	isLoadingAll       bool
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
		List:        l,
		Viewport:    vp,
		Client:      client,
		Topics:      topics,
		Search:      search,
		LastRefresh: time.Now(),
		InstanceURL: instanceURL,
		State:       stateTopicList,
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
	var cmd tea.Cmd

	m.StatusMessage = ""

	switch m.State {
	case stateNewTopic:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEsc {
				m.State = stateTopicList
				m.NewTopicForm.message = ""
				m.NewTopicForm.err = nil
				return m, nil
			}
		case topicCreatedMsg:
			m.State = stateTopicList
			m.StatusMessage = msg.message
			m.NewTopicForm.submitting = false
			m.NewTopicForm.message = ""
			cmds = append(cmds, func() tea.Msg { return refreshMsg{} })
			return m, tea.Batch(cmds...)
		case topicCreateErrorMsg:
			m.NewTopicForm.err = msg.err
			m.NewTopicForm.submitting = false
			m.NewTopicForm.message = ""
			log.Printf("Error creating topic: %v", msg.err)
			return m, nil
		}
		newForm, newCmd := m.NewTopicForm.Update(msg)
		m.NewTopicForm = *(newForm.(*newTopicModel))
		cmds = append(cmds, newCmd)
		return m, tea.Batch(cmds...)

	case stateTopicList:
		switch msg := msg.(type) {
		case refreshMsg:
			if m.isRefreshingTopics {
				return m, nil
			}
			m.isRefreshingTopics = true
			m.StatusMessage = "Refreshing topics..."
			cmds = append(cmds, func() tea.Msg {
				response, err := m.Client.RefreshTopics()
				if err != nil {
					return topicsRefreshErrorMsg{err: err}
				}
				categories, catErr := m.Client.GetCategories()
				if catErr != nil {
					log.Printf("Warning: failed to fetch categories during refresh: %v", catErr)
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
				return topicsRefreshedMsg{response: response}
			})
			return m, tea.Batch(cmds...)
		case topicsRefreshedMsg:
			m.isRefreshingTopics = false
			m.StatusMessage = "Topics refreshed!"
			items := make([]list.Item, len(msg.response.TopicList.Topics))
			for i, topic := range msg.response.TopicList.Topics {
				items[i] = topicItem{topic: topic}
			}
			m.List.SetItems(items)
			m.Topics = msg.response.TopicList.Topics
			m.MoreTopicsURL = msg.response.TopicList.MoreTopicsURL
			m.LastRefresh = time.Now()
			cmds = append(cmds, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
				return refreshMsg{}
			}))
			return m, tea.Batch(cmds...)
		case topicsRefreshErrorMsg:
			m.isRefreshingTopics = false
			m.StatusMessage = fmt.Sprintf("Error refreshing topics: %v", msg.err)
			log.Printf("Failed to refresh topics: %v", msg.err)
			cmds = append(cmds, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
				return refreshMsg{}
			}))
			return m, tea.Batch(cmds...)
		case moreTopicsLoadedMsg:
			m.isLoadingMore = false
			m.StatusMessage = fmt.Sprintf("Loaded %d more topics!", len(msg.response.TopicList.Topics))
			
			// Append new topics to existing ones
			m.Topics = append(m.Topics, msg.response.TopicList.Topics...)
			m.MoreTopicsURL = msg.response.TopicList.MoreTopicsURL
			
			// Update list items
			items := make([]list.Item, len(m.Topics))
			for i, topic := range m.Topics {
				items[i] = topicItem{topic: topic}
			}
			m.List.SetItems(items)
			return m, tea.Batch(cmds...)
		case moreTopicsLoadErrorMsg:
			m.isLoadingMore = false
			m.StatusMessage = fmt.Sprintf("Error loading more topics: %v", msg.err)
			log.Printf("Failed to load more topics: %v", msg.err)
			return m, tea.Batch(cmds...)
		case loadAllTopicsMsg:
			m.isLoadingAll = false
			m.StatusMessage = fmt.Sprintf("Loaded all %d topics!", len(msg.response.TopicList.Topics))
			
			// Replace with all topics
			m.Topics = msg.response.TopicList.Topics
			m.MoreTopicsURL = msg.response.TopicList.MoreTopicsURL
			
			// Update list items
			items := make([]list.Item, len(m.Topics))
			for i, topic := range m.Topics {
				items[i] = topicItem{topic: topic}
			}
			m.List.SetItems(items)
			return m, tea.Batch(cmds...)
		case loadAllTopicsErrorMsg:
			m.isLoadingAll = false
			m.StatusMessage = fmt.Sprintf("Error loading all topics: %v", msg.err)
			log.Printf("Failed to load all topics: %v", msg.err)
			return m, tea.Batch(cmds...)

		case tea.KeyMsg:
			if m.Searching {
				switch msg.String() {
				case "esc", "enter":
					if msg.String() == "enter" {
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
						} else {
							items := make([]list.Item, len(m.Topics))
							for i, topic := range m.Topics {
								items[i] = topicItem{topic: topic}
							}
							m.List.SetItems(items)
						}
					}
					m.Searching = false
					m.Search.Blur()
					m.Search.Reset()
					return m, nil
				default:
					m.Search, cmd = m.Search.Update(msg)
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
			}

			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "n":
				m.State = stateNewTopic
				m.NewTopicForm = InitialNewTopicModel(m.Client, m.Width, m.Height-4)
				m.NewTopicForm.message = ""
				m.NewTopicForm.err = nil
				return m, m.NewTopicForm.Init()
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
				if m.isRefreshingTopics {
					return m, nil
				}
				m.isRefreshingTopics = true
				m.StatusMessage = "Refreshing topics..."
				cmds = append(cmds, func() tea.Msg {
					response, err := m.Client.RefreshTopics()
					if err != nil {
						return topicsRefreshErrorMsg{err: err}
					}
					categories, catErr := m.Client.GetCategories()
					if catErr != nil {
						log.Printf("Warning: failed to fetch categories during refresh: %v", catErr)
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
					return topicsRefreshedMsg{response: response}
				})
				return m, tea.Batch(cmds...)
			case "m":
				if m.isLoadingMore || m.MoreTopicsURL == "" {
					return m, nil
				}
				m.isLoadingMore = true
				m.StatusMessage = "Loading more topics..."
				cmds = append(cmds, func() tea.Msg {
					response, err := m.Client.GetMoreTopics(m.MoreTopicsURL)
					if err != nil {
						return moreTopicsLoadErrorMsg{err: err}
					}
					categories, catErr := m.Client.GetCategories()
					if catErr != nil {
						log.Printf("Warning: failed to fetch categories for more topics: %v", catErr)
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
					return moreTopicsLoadedMsg{response: response}
				})
				return m, tea.Batch(cmds...)
			case "M":
				if m.isLoadingAll {
					return m, nil
				}
				m.isLoadingAll = true
				m.StatusMessage = "Loading all topics (this may take a while)..."
				cmds = append(cmds, func() tea.Msg {
					response, err := m.Client.LoadAllTopics(20)
					if err != nil {
						return loadAllTopicsErrorMsg{err: err}
					}
					return loadAllTopicsMsg{response: response}
				})
				return m, tea.Batch(cmds...)
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
					if m.isLoadingPosts {
						return m, nil
					}
					m.isLoadingPosts = true
					m.Viewport.SetContent("Loading posts...")
					selectedTopicID := i.topic.ID
					// First load only the first page to show content quickly.
					cmd1 := func() tea.Msg {
						postsPage, err := m.Client.GetTopicPostsPage(selectedTopicID, 1)
						if err != nil {
							return postsLoadErrorMsg{err: err}
						}
						return postsLoadedMsg{posts: postsPage}
					}
					// Then load the full topic in background.
					cmd2 := func() tea.Msg {
						fullPosts, err := m.Client.GetTopicPosts(selectedTopicID)
						if err != nil {
							return postsLoadErrorMsg{err: err}
						}
						return postsLoadedMsg{posts: fullPosts}
					}
					cmds = append(cmds, cmd1, cmd2)
				}
			}
		case postsLoadedMsg:
			m.isLoadingPosts = false
			var content strings.Builder
			postContentWidth := m.Viewport.Width - 2
			if postContentWidth < 1 {
				postContentWidth = 1
			}
			for _, post := range msg.posts.PostStream.Posts {
				content.WriteString(FormatPost(post, postContentWidth))
				content.WriteString("\n\n---\n\n")
			}
			m.Viewport.SetContent(content.String())
			m.Viewport.GotoTop()
		case postsLoadErrorMsg:
			m.isLoadingPosts = false
			errorContentWidth := m.Viewport.Width - 2
			if errorContentWidth < 1 {
				errorContentWidth = 1
			}
			errorStyle := lipgloss.NewStyle().Width(errorContentWidth)
			m.Viewport.SetContent(errorStyle.Render(fmt.Sprintf("Error fetching posts: %v", msg.err)))
		case tea.WindowSizeMsg:
			m.Width = msg.Width
			m.Height = msg.Height

			if !m.Ready {
				m.Ready = true
			}

			if m.State == stateNewTopic {
				m.NewTopicForm.width = msg.Width
				m.NewTopicForm.height = msg.Height - 4
				m.NewTopicForm.titleInput.Width = msg.Width - 4
				m.NewTopicForm.contentInput.SetWidth(msg.Width - 4)
				m.NewTopicForm.contentInput.SetHeight((msg.Height - 4) / 3)
				m.NewTopicForm.categoryInput.Width = msg.Width - 4
				m.NewTopicForm.tagsInput.Width = msg.Width - 4
			} else {
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
		}

		m.List, cmd = m.List.Update(msg)
		cmds = append(cmds, cmd)

		m.Viewport, cmd = m.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.Ready {
		return "\nInitializing..."
	}

	if m.State == stateNewTopic {
		return m.NewTopicForm.View()
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
		Render(fmt.Sprintf("Press 'f' for fullscreen, '/' to search, 'R' to refresh, 'm' to load more, 'M' to load all, 'esc' to exit fullscreen/search • Last refresh: %s", m.LastRefresh.Format("15:04:05")))

	if m.StatusMessage != "" {
		help = lipgloss.JoinHorizontal(lipgloss.Left, config.StatusStyle.Render(m.StatusMessage), " • ", help)
	}

	if m.Fullscreen {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			instanceHeader,
			lipgloss.NewStyle().
				Width(m.Width).
				Height(m.Height-4).
				MaxWidth(m.Width).
				MaxHeight(m.Height-4).
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
	p := bluemonday.UGCPolicy()
	p.AllowElements("a").AllowAttrs("href").OnElements("a")
	p.AllowElements("code", "pre", "blockquote", "em", "strong", "br", "p", "div")
	
	sanitizedContent := p.Sanitize(post.Cooked)
	
	text := convertHTMLToText(sanitizedContent)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	potentialParagraphs := strings.Split(text, "\n")
	var paragraphsSource []string
	for _, para := range potentialParagraphs {
		trimmedPara := strings.TrimSpace(para)
		if trimmedPara != "" {
			paragraphsSource = append(paragraphsSource, trimmedPara)
		}
	}

	if contentWidth < 1 {
		contentWidth = 1
	}
	contentWrappingStyle := lipgloss.NewStyle().Width(contentWidth)

	var renderedParagraphs []string
	for _, paraStr := range paragraphsSource {
		renderedBlock := contentWrappingStyle.Render(paraStr)
		renderedBlock = strings.TrimRight(renderedBlock, "\n")
		renderedParagraphs = append(renderedParagraphs, renderedBlock)
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
		if action.ID == 2 {
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

func convertHTMLToText(html string) string {
	html = strings.ReplaceAll(html, "<br/>", "\n")
	html = strings.ReplaceAll(html, "<br>", "\n")
	html = strings.ReplaceAll(html, "</p>", "\n\n")
	html = strings.ReplaceAll(html, "</div>", "\n")
	html = strings.ReplaceAll(html, "</blockquote>", "\n")
	
	var result strings.Builder
	var currentTag strings.Builder
	var inTag bool
	var inAnchor bool
	var anchorHref string
	var anchorText strings.Builder
	
	i := 0
	for i < len(html) {
		char := html[i]
		
		if char == '<' {
			inTag = true
			currentTag.Reset()
		} else if char == '>' && inTag {
			inTag = false
			tag := currentTag.String()
			
			if strings.HasPrefix(tag, "a ") && strings.Contains(tag, "href=") {
				inAnchor = true
				anchorText.Reset()
				start := strings.Index(tag, `href="`) + 6
				if start > 5 {
					end := strings.Index(tag[start:], `"`)
					if end > 0 {
						anchorHref = tag[start : start+end]
					}
				}
			} else if tag == "/a" && inAnchor {
				inAnchor = false
				linkText := anchorText.String()
				if linkText == anchorHref || strings.TrimSpace(linkText) == "" {
					result.WriteString(anchorHref)
				} else {
					result.WriteString(fmt.Sprintf("%s (%s)", linkText, anchorHref))
				}
				anchorHref = ""
			} else if tag == "code" {
				result.WriteString("`")
			} else if tag == "/code" {
				result.WriteString("`")
			} else if tag == "pre" {
				result.WriteString("\n```\n")
			} else if tag == "/pre" {
				result.WriteString("\n```\n")
			} else if tag == "blockquote" {
				result.WriteString("\n> ")
			} else if tag == "strong" || tag == "b" {
				result.WriteString("**")
			} else if tag == "/strong" || tag == "/b" {
				result.WriteString("**")
			} else if tag == "em" || tag == "i" {
				result.WriteString("*")
			} else if tag == "/em" || tag == "/i" {
				result.WriteString("*")
			}
		} else if inTag {
			currentTag.WriteByte(char)
		} else if inAnchor {
			anchorText.WriteByte(char)
		} else {
			result.WriteByte(char)
		}
		
		i++
	}
	
	text := result.String()
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	
	return text
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

	s.WriteString("\n\nPress Tab/Shift+Tab to switch fields, Enter to submit, Esc to quit") // Updated help text for login

	return s.String()
}
