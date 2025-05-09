// Copyright (c) 2025 Sudo-Ivan
// MIT License

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Sudo-Ivan/discourse-tui-client/internal/config"
	"github.com/Sudo-Ivan/discourse-tui-client/internal/tui"
	"github.com/Sudo-Ivan/discourse-tui-client/pkg/discourse"
)

func setupLogging() (*os.File, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user cache directory: %w", err)
	}
	appLogDir := filepath.Join(userCacheDir, "discourse-tui-client", "logs")
	if err := os.MkdirAll(appLogDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", appLogDir, err)
	}
	logFilePath := filepath.Join(appLogDir, "activity.log")

	/* #nosec G304 */
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", logFilePath, err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	return logFile, nil
}

func main() {
	debug := flag.Bool("debug", false, "Enable debug logging.")
	flag.BoolVar(debug, "d", false, "Enable debug logging (shorthand).")
	cookiesPath := flag.String("cookies", "", "Path to cookies file (optional).")
	instanceURL := flag.String("url", "", "Discourse instance URL (e.g. https://forum.example.com).")
	flag.BoolVar(debug, "u", false, "Discourse instance URL (shorthand).")
	logout := flag.Bool("logout", false, "Logout and delete cookies.")
	flag.Parse()

	if *logout {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			fmt.Printf("Failed to get user config directory: %v\n", err)
			os.Exit(1)
		}
		cookieFile := filepath.Join(userConfigDir, "discourse-tui-client", "cookies.txt")
		if err := os.Remove(cookieFile); err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("Failed to delete cookies: %v\n", err)
				os.Exit(1)
			}
		}
		fmt.Println("Successfully logged out.")
		os.Exit(0)
	}

	var logFile *os.File
	var err error

	if *debug {
		logFile, err = setupLogging()
		if err != nil {
			fmt.Printf("Failed to setup logging: %v\n", err)
			os.Exit(1)
		}
		log.Println("Debug logging enabled.")
	} else {
		log.SetOutput(io.Discard)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	log.Println("Starting Discourse client")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Critical: Failed to get user config directory: %v", err)
	}
	appConfigDir := filepath.Join(userConfigDir, "discourse-tui-client")
	if err := os.MkdirAll(appConfigDir, 0750); err != nil {
		log.Fatalf("Critical: Failed to create app config directory %s: %v", appConfigDir, err)
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("Critical: Failed to get user cache directory: %v", err)
	}
	appCacheDir := filepath.Join(userCacheDir, "discourse-tui-client")
	if err := os.MkdirAll(appCacheDir, 0750); err != nil {
		log.Fatalf("Critical: Failed to create app cache directory %s: %v", appCacheDir, err)
	}

	defaultCookiesPath := filepath.Join(appConfigDir, "cookies.txt")
	if *cookiesPath != "" {
		defaultCookiesPath = *cookiesPath
	}

	colorsPath := filepath.Join(appConfigDir, "colors.txt")
	
	instanceName := "placeholder"
	if *instanceURL != "" {
		instanceName = strings.TrimPrefix(strings.TrimPrefix(*instanceURL, "https://"), "http://")
	}
	latestTopicsCachePath := filepath.Join(appCacheDir, "instances", instanceName, "latest.json")

	log.Printf("Using cookies path: %s", defaultCookiesPath)
	log.Printf("Using colors path: %s", colorsPath)
	log.Printf("Using latest topics cache path: %s", latestTopicsCachePath)

	loadedColors, err := config.LoadColors(colorsPath)
	if err != nil {
		log.Printf("Failed to load colors from %s: %v. Using default colors.", colorsPath, err)
	}
	config.UpdateStyles(loadedColors)

	var client *discourse.Client
	if *instanceURL != "" {
		client, err = discourse.NewClient(*instanceURL, defaultCookiesPath)
		if err != nil {
			log.Printf("Failed to create client: %v", err)
			fmt.Printf("Failed to create client: %v\n", err)
			os.Exit(1)
		}
	} else {
		savedInstance, err := config.LoadInstance()
		if err != nil {
			log.Printf("Failed to load saved instance: %v", err)
		}
		if savedInstance != "" {
			client, err = discourse.NewClient(savedInstance, defaultCookiesPath)
			if err != nil {
				log.Printf("Failed to create client with saved instance: %v", err)
			}
		}
		if client == nil {
			client, err = discourse.NewClient("https://placeholder.com", defaultCookiesPath)
			if err != nil {
				log.Printf("Failed to create temporary client: %v", err)
				fmt.Printf("Failed to create temporary client: %v\n", err)
				os.Exit(1)
			}
		}
	}

	if _, statErr := os.Stat(defaultCookiesPath); os.IsNotExist(statErr) {
		log.Printf("Cookies file not found at %s. Initiating login.", defaultCookiesPath)
		loginModel := tui.InitialLoginModel(client)
		p := tea.NewProgram(loginModel)
		if _, runErr := p.Run(); runErr != nil {
			log.Printf("Login program error: %v", runErr)
			fmt.Printf("Login error: %v\n", runErr)
			os.Exit(1)
		}
		if _, statErrAfterLogin := os.Stat(defaultCookiesPath); os.IsNotExist(statErrAfterLogin) {
			log.Printf("Login failed or was quit, cookies file not created at %s.", defaultCookiesPath)
			fmt.Println("Login failed or was quit, cookies file not created.")
			os.Exit(1)
		}
		log.Printf("Cookies file successfully created/found at %s after login.", defaultCookiesPath)
		
		client, err = discourse.NewClient(loginModel.GetInstanceURL(), defaultCookiesPath)
		if err != nil {
			log.Printf("Failed to create client with new URL: %v", err)
			fmt.Printf("Failed to create client with new URL: %v\n", err)
			os.Exit(1)
		}
		instanceName = strings.TrimPrefix(strings.TrimPrefix(loginModel.GetInstanceURL(), "https://"), "http://")
		latestTopicsCachePath = filepath.Join(appCacheDir, "instances", instanceName, "latest.json")
		log.Printf("Updated latest topics cache path: %s", latestTopicsCachePath)
	}

	if err := client.LoadCookies(defaultCookiesPath); err != nil {
		log.Printf("Failed to load cookies from %s: %v", defaultCookiesPath, err)
		fmt.Printf("Failed to load cookies from %s: %v\n", defaultCookiesPath, err)
		os.Exit(1)
	}
	log.Printf("Successfully loaded cookies from %s", defaultCookiesPath)

	var topicsResponse *discourse.Response

	/* #nosec G304 */
	cachedData, err := os.ReadFile(latestTopicsCachePath)
	if err == nil {
		var cr discourse.Response
		if jsonErr := json.Unmarshal(cachedData, &cr); jsonErr == nil {
			topicsResponse = &cr
			log.Printf("Successfully loaded latest topics from cache: %s", latestTopicsCachePath)
		} else {
			log.Printf("Failed to unmarshal cached topics from %s: %v. Fetching from network.", latestTopicsCachePath, jsonErr)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Error reading cache file %s: %v. Fetching from network.", latestTopicsCachePath, err)
	} else {
		log.Printf("Cache file %s not found. Fetching from network.", latestTopicsCachePath)
	}

	if topicsResponse == nil {
		log.Println("Fetching latest topics from network.")
		networkResponse, fetchErr := client.GetLatestTopics()
		if fetchErr != nil {
			log.Printf("Failed to fetch topics: %v", fetchErr)
			fmt.Printf("Failed to fetch topics: %v\n", fetchErr)
			os.Exit(1)
		}
		topicsResponse = networkResponse

		jsonData, marshalErr := json.MarshalIndent(topicsResponse, "", "  ")
		if marshalErr == nil {
			if writeErr := os.WriteFile(latestTopicsCachePath, jsonData, 0600); writeErr == nil {
				log.Printf("Successfully saved latest topics to cache: %s", latestTopicsCachePath)
			} else {
				log.Printf("Failed to write topics cache to %s: %v", latestTopicsCachePath, writeErr)
			}
		} else {
			log.Printf("Failed to marshal topics for caching: %v", marshalErr)
		}
	}

	if topicsResponse == nil || len(topicsResponse.TopicList.Topics) == 0 {
		log.Println("No topics found after attempting cache and network fetch. Exiting.")
		fmt.Println("No topics found. Please check your connection and ensure you are logged in correctly.")
		os.Exit(1)
	}

	log.Printf("Using %d topics for TUI", len(topicsResponse.TopicList.Topics))

	p := tea.NewProgram(
		tui.InitialModel(client, topicsResponse.TopicList.Topics),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, runErr := p.Run(); runErr != nil {
		log.Printf("Main program error: %v", runErr)
		fmt.Printf("Error running TUI: %v\n", runErr)
		os.Exit(1)
	}
	log.Println("Discourse client exited normally.")
} 