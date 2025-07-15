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
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Sudo-Ivan/discourse-tui-client/internal/config"
	"github.com/Sudo-Ivan/discourse-tui-client/internal/tui"
	"github.com/Sudo-Ivan/discourse-tui-client/pkg/discourse"
	"github.com/Sudo-Ivan/discourse-tui-client/pkg/output"
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
	flag.StringVar(cookiesPath, "c", "", "Path to cookies file (shorthand).")
	instanceURL := flag.String("url", "", "Discourse instance URL (e.g. https://forum.example.com).")
	flag.StringVar(instanceURL, "u", "", "Discourse instance URL (shorthand).")
	logout := flag.Bool("logout", false, "Logout and delete cookies.")
	flag.BoolVar(logout, "l", false, "Logout and delete cookies (shorthand).")
	resetCache := flag.Bool("reset-cache", false, "Reset cache and force fresh fetch.")
	flag.BoolVar(resetCache, "r", false, "Reset cache and force fresh fetch (shorthand).")
	outputPath := flag.String("output", "", "Output posts to file (txt, json, or html)")
	flag.StringVar(outputPath, "o", "", "Output posts to file (shorthand)")
	cooldown := flag.Duration("cooldown", 500*time.Millisecond, "Cooldown between page fetches (e.g. 500ms)")
	loadAll := flag.Bool("load-all", false, "Load all available topics at startup (may be slow)")
	flag.BoolVar(loadAll, "a", false, "Load all available topics at startup (shorthand)")
	noAuth := flag.Bool("no-auth", false, "Run in unauthenticated mode.")
	flag.BoolVar(noAuth, "na", false, "Run in unauthenticated mode (shorthand).")
	flag.Parse()

	if *outputPath != "" {
		if !strings.HasSuffix(*outputPath, ".txt") && !strings.HasSuffix(*outputPath, ".json") && !strings.HasSuffix(*outputPath, ".html") {
			fmt.Println("Output file must end with .txt, .json, or .html")
			os.Exit(1)
		}
	}

	if *resetCache {
		userCacheDir, err := os.UserCacheDir()
		if err != nil {
			fmt.Printf("Failed to get user cache directory: %v\n", err)
			os.Exit(1)
		}
		cacheDir := filepath.Join(userCacheDir, "discourse-tui-client", "instances")
		if err := os.RemoveAll(cacheDir); err != nil {
			fmt.Printf("Failed to reset cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Cache reset successfully.")
		os.Exit(0)
	}

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
	var clientCookiesPath string

	if *noAuth {
		log.Println("Running in unauthenticated mode. Skipping login.")
		clientCookiesPath = "" // No cookies path needed for unauthenticated
		// If no instance URL is provided in unauthenticated mode, use a default one
		if *instanceURL == "" {
			*instanceURL = "https://meta.discourse.org" // A common public Discourse instance
			log.Printf("No instance URL provided in unauthenticated mode, using default: %s", *instanceURL)
		}
	} else {
		clientCookiesPath = defaultCookiesPath
		if _, statErr := os.Stat(defaultCookiesPath); os.IsNotExist(statErr) {
			log.Printf("Cookies file not found at %s. Initiating login.", defaultCookiesPath)
			loginModel := tui.InitialLoginModel(nil) // Pass nil client initially, it will be created after login
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

			*instanceURL = loginModel.GetInstanceURL() // Update instanceURL from login model
		}

		if err := client.LoadCookies(defaultCookiesPath); err != nil {
			log.Printf("Failed to load cookies from %s: %v", defaultCookiesPath, err)
			fmt.Printf("Failed to load cookies from %s: %v\n", defaultCookiesPath, err)
			os.Exit(1)
		}
		log.Printf("Successfully loaded cookies from %s", defaultCookiesPath)
	}

	// Create the client after determining the instanceURL and cookiesPath
	if *instanceURL == "" {
		savedInstance, err := config.LoadInstance()
		if err != nil {
			log.Printf("Failed to load saved instance: %v", err)
		}
		if savedInstance != "" {
			*instanceURL = savedInstance
		}
	}

	if *instanceURL == "" {
		*instanceURL = "https://placeholder.com" // Fallback if no URL is provided and not in no-auth mode
	}

	client, err = discourse.NewClient(*instanceURL, clientCookiesPath)
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}
	client.SetPageCooldown(*cooldown)

	instanceName = strings.TrimPrefix(strings.TrimPrefix(*instanceURL, "https://"), "http://")
	latestTopicsCachePath = filepath.Join(appCacheDir, "instances", instanceName, "latest.json")
	log.Printf("Updated latest topics cache path: %s", latestTopicsCachePath)

	// Fetch categories only if not in no-auth mode and after successful login/cookie load
	if !*noAuth {
		categories, err := client.GetCategories()
		if err != nil {
			log.Printf("Warning: Failed to fetch categories after login: %v", err)
		} else {
			log.Printf("Successfully fetched %d categories after login", len(categories.CategoryList.Categories))
		}
	}

	var topicsResponse *discourse.Response

	/* #nosec G304 */
	cachedData, err := os.ReadFile(latestTopicsCachePath)
	if err == nil {
		log.Printf("Attempting to load latest topics from cache: %s", latestTopicsCachePath)
		var cachedResp discourse.Response
		if unmarshalErr := json.Unmarshal(cachedData, &cachedResp); unmarshalErr == nil {
			if len(cachedResp.TopicList.Topics) > 0 || len(cachedResp.Users) > 0 {
				topicsResponse = &cachedResp
				log.Printf("Successfully parsed latest topics from cache using encoding/json: %s", latestTopicsCachePath)
			} else {
				log.Printf("Cached data in %s parsed but seems empty or invalid (no topics/users). Fetching from network.", latestTopicsCachePath)
				topicsResponse = nil
			}
		} else {
			log.Printf("Failed to parse cached topics from %s with encoding/json: %v. Fetching from network.", latestTopicsCachePath, unmarshalErr)
			topicsResponse = nil
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Error reading cache file %s: %v. Fetching from network.", latestTopicsCachePath, err)
	} else {
		log.Printf("Cache file %s not found. Fetching from network.", latestTopicsCachePath)
	}

	if topicsResponse == nil {
		log.Println("Fetching latest topics from network.")
		var networkResponse *discourse.Response
		var fetchErr error

		if *loadAll {
			log.Println("Loading all available topics (this may take a while)...")
			networkResponse, fetchErr = client.LoadAllTopics(20)
		} else {
			networkResponse, fetchErr = client.GetLatestTopics()
		}

		if fetchErr != nil {
			log.Printf("Failed to fetch topics: %v", fetchErr)
			fmt.Printf("Failed to fetch topics: %v\n", fetchErr)
			os.Exit(1)
		}
		topicsResponse = networkResponse

		categories, err := client.GetCategories()
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

			for i := range topicsResponse.TopicList.Topics {
				if cat, ok := categoryMap[topicsResponse.TopicList.Topics[i].CategoryID]; ok {
					topicsResponse.TopicList.Topics[i].CategoryName = cat.Name
					topicsResponse.TopicList.Topics[i].CategoryColor = cat.Color
				}
			}
		}

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

	if *outputPath != "" {
		output.SetClient(client)
		if err := output.WriteToFile(*outputPath, topicsResponse); err != nil {
			log.Printf("Failed to write output file: %v", err)
			fmt.Printf("Failed to write output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully wrote output to %s\n", *outputPath)
		os.Exit(0)
	}

	log.Printf("Using %d topics for TUI", len(topicsResponse.TopicList.Topics))

	initialModel := tui.InitialModel(client, topicsResponse.TopicList.Topics)
	initialModel.MoreTopicsURL = topicsResponse.TopicList.MoreTopicsURL

	p := tea.NewProgram(
		initialModel,
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
