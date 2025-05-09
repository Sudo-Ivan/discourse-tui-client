// Copyright (c) 2025 Sudo-Ivan
// MIT License

package tui

import (
	"fmt"
	"strings"
	"log"

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

func (i topicItem) Title() string       { return i.topic.Title }
func (i topicItem) Description() string { return fmt.Sprintf("%d replies • %d views", i.topic.ReplyCount, i.topic.Views) }
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

	return Model{
		List:      l,
		Viewport:  vp,
		Client:    client,
		Topics:    topics,
		Search:    search,
	}
}

func (m Model) Init() tea.Cmd {
	log.Printf("Initializing model with %d topics", len(m.Topics))
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "f":
			m.Fullscreen = !m.Fullscreen
			if m.Fullscreen {
				m.Viewport.Width = m.Viewport.Width
				m.Viewport.Height = m.Viewport.Height
			}
			return m, nil
		case "/":
			m.Searching = !m.Searching
			if m.Searching {
				return m, m.Search.Focus()
			}
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
					m.Viewport.SetContent(fmt.Sprintf("Error fetching posts: %v", err))
					return m, nil
				}

				var content strings.Builder
				for _, post := range posts.PostStream.Posts {
					content.WriteString(FormatPost(post))
					content.WriteString("\n\n---\n\n")
				}

				m.Viewport.SetContent(content.String())
			}
		}
	case tea.WindowSizeMsg:
		if !m.Ready {
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
			m.Ready = true
		} else if m.Fullscreen {
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height
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

	if m.Fullscreen {
		return lipgloss.NewStyle().
			Width(m.Viewport.Width).
			Height(m.Viewport.Height).
			Render(m.Viewport.View())
	}

	var view string
	if m.Searching {
		searchBox := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			Render(m.Search.View())

		view = lipgloss.JoinVertical(
			lipgloss.Left,
			searchBox,
			m.List.View(),
			lipgloss.NewStyle().MarginTop(1).Render(m.Viewport.View()),
		)
	} else {
		view = lipgloss.JoinVertical(
			lipgloss.Left,
			m.List.View(),
			lipgloss.NewStyle().MarginTop(1).Render(m.Viewport.View()),
		)
	}

	help := "\nPress 'f' for fullscreen, '/' to search, 'esc' to exit fullscreen/search"
	return view + help
}

func FormatPost(post discourse.Post) string {
	p := bluemonday.StrictPolicy()
	cleanHTML := p.Sanitize(post.Cooked)

	return fmt.Sprintf("Post #%d by %s (%s)\nPosted: %s\n\n%s\n\nReads: %d | Score: %.1f",
		post.PostNumber,
		post.Name,
		post.Username,
		post.CreatedAt.Format("2006-01-02 15:04:05"),
		cleanHTML,
		post.Reads,
		post.Score)
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