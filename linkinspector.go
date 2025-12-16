package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/logrusorgru/aurora/v4"
	"github.com/projectdiscovery/goflags"
	"github.com/rix4uni/linkinspector/banner"
	"gopkg.in/yaml.v3"
)

type Options struct {
	InputTargetHost string
	InputFile       string
	Passive         bool
	MatchCode       string
	MatchLength     string
	MatchType       string
	MatchSuffix     string
	FilterCode      string
	FilterLength    string
	FilterType      string
	FilterSuffix    string
	Output          string
	JSONOutput      bool
	JSONtype        string
	Threads         int
	UserAgent       string
	Verbose         bool
	Version         bool
	Silent          bool
	NoColor         bool
	Timeout         int
	Insecure        bool
	Delay           time.Duration
}

type Config struct {
	ValidExtensions   map[string]string `yaml:"valid_extensions"`
	PassiveExtensions map[string]string `yaml:"passive_extensions"`
}

// downloadConfig downloads the config file from the given URL to the specified filepath.
func downloadConfig(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download config: received status code %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// loadConfig loads the configuration from extensions.yaml file.
// Exits with an error if the file is missing, cannot be read, or is invalid.
func loadConfig() *Config {
	config := &Config{
		ValidExtensions:   make(map[string]string),
		PassiveExtensions: make(map[string]string),
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get user home directory: %v\n", err)
		os.Exit(1)
	}

	// Build config directory and file paths
	configDir := filepath.Join(homeDir, ".config", "linkinspector")
	configPath := filepath.Join(configDir, "extensions.yaml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create config directory: %v\n", err)
		os.Exit(1)
	}

	// Check if extensions.yaml exists, download if it doesn't
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configURL := "https://raw.githubusercontent.com/rix4uni/linkinspector/refs/heads/main/extensions.yaml"
		if err := downloadConfig(configURL, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Read extensions.yaml file
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read extensions.yaml: %v\n", err)
		os.Exit(1)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to parse extensions.yaml: %v\n", err)
		os.Exit(1)
	}

	// Validate that config is not empty
	if len(config.ValidExtensions) == 0 && len(config.PassiveExtensions) == 0 {
		fmt.Fprintf(os.Stderr, "Error: extensions.yaml is empty or contains no valid configuration\n")
		os.Exit(1)
	}

	return config
}

// Define the flags
func ParseOptions() *Options {
	options := &Options{}
	flagSet := goflags.NewFlagSet()
	flagSet.SetDescription(`linkinspector is a fast command-line tool for inspecting URLs and retrieving HTTP status codes, content lengths, and content types. It supports filtering and matching responses, and can process URLs from stdin or files.`)

	createGroup(flagSet, "input", "Input",
		flagSet.StringVarP(&options.InputTargetHost, "target", "u", "", "Single URL to check"),
		flagSet.StringVarP(&options.InputFile, "list", "l", "", "File containing list of URLs to check"),
	)

	createGroup(flagSet, "probes", "Probes",
		flagSet.BoolVar(&options.Passive, "passive", false, "Enable passive mode to skip requests for specific extensions"),
	)

	createGroup(flagSet, "matchers", "Matchers",
		flagSet.StringVarP(&options.MatchCode, "match-code", "mc", "", "Match response with specified status code (e.g., -mc 200,302)"),
		flagSet.StringVarP(&options.MatchLength, "match-length", "ml", "", "Match response with specified content length (e.g., -ml 100,102)"),
		flagSet.StringVarP(&options.MatchType, "match-type", "mt", "", "Match response with specified content type (e.g., -mt \"application/octet-stream,text/html\")"),
		flagSet.StringVarP(&options.MatchSuffix, "match-suffix", "ms", "", "Match response with specified suffix name (e.g., -ms \"ZIP,PHP,7Z\")"),
	)

	createGroup(flagSet, "filters", "Filters",
		flagSet.StringVarP(&options.FilterCode, "filter-code", "fc", "", "Filter response with specified status code (e.g., -fc 403,401)"),
		flagSet.StringVarP(&options.FilterLength, "filter-length", "fl", "", "Filter response with specified content length (e.g., -fl 23,33)"),
		flagSet.StringVarP(&options.FilterType, "filter-type", "ft", "", "Filter response with specified content type (e.g., -ft \"text/html,image/jpeg\")"),
		flagSet.StringVarP(&options.FilterSuffix, "filter-suffix", "fs", "", "Filter response with specified suffix name (e.g., -fs \"CSS,Plain Text,html\")"),
	)

	createGroup(flagSet, "output", "Output",
		flagSet.StringVarP(&options.Output, "output", "o", "", "File to write output results"),
		flagSet.BoolVar(&options.JSONOutput, "json", false, "Output in JSON format"),
		flagSet.StringVar(&options.JSONtype, "json-type", "MarshalIndent", "Output in JSON type, MarshalIndent or Marshal"),
	)

	createGroup(flagSet, "rate-limit", "RATE-LIMIT",
		flagSet.IntVarP(&options.Threads, "threads", "t", 50, "Number of threads to use"),
	)

	createGroup(flagSet, "configurations", "Configurations",
		flagSet.StringVar(&options.UserAgent, "H", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36", "Custom User-Agent header for HTTP requests"),
	)

	createGroup(flagSet, "debug", "Debug",
		flagSet.BoolVar(&options.Verbose, "verbose", false, "Enable verbose output for debugging purposes"),
		flagSet.BoolVar(&options.Version, "version", false, "Print the version of the tool and exit"),
		flagSet.BoolVar(&options.Silent, "silent", false, "silent mode"),
		flagSet.BoolVarP(&options.NoColor, "no-color", "nc", false, "disable colors in cli output"),
	)

	createGroup(flagSet, "optimizations", "OPTIMIZATIONS",
		flagSet.IntVar(&options.Timeout, "timeout", 30, "HTTP request timeout duration (in seconds)"),
		flagSet.BoolVar(&options.Insecure, "insecure", false, "Disable TLS certificate verification"),
		flagSet.DurationVar(&options.Delay, "delay", -1*time.Nanosecond, "Duration between each HTTP request (e.g., 200ms, 1s)"),
	)

	_ = flagSet.Parse()

	return options
}

func createGroup(flagSet *goflags.FlagSet, groupName, description string, flags ...*goflags.FlagData) {
	flagSet.SetGroup(groupName, description)
	for _, currentFlag := range flags {
		currentFlag.Group(groupName)
	}
}

// Struct for JSON output
type JSONOutput struct {
	Host string `json:"host"`
	Type string `json:"type"`
	Data struct {
		StatusCode    int64  `json:"status_code,omitempty"`
		ContentLength int64  `json:"content_length,omitempty"`
		ContentType   string `json:"content_type,omitempty"`
		Suffix        string `json:"suffix,omitempty"`
	} `json:"data"`
}

// Function to check if a value matches any of the specified filters
func matches(value string, filter string) bool {
	if filter == "" {
		return true // No filter applied
	}
	filters := strings.Split(filter, ",")
	for _, f := range filters {
		if strings.TrimSpace(f) == value {
			return true
		}
	}
	return false
}

// Check URL information and return the required output format with custom timeout, TLS, and User-Agent settings.
func getURLInfo(url string, verbose bool, timeout time.Duration, insecure bool, userAgent string, jsonOutput bool, jsonTypeFlag string, outputFile *os.File, options *Options, validExtensions map[string]string) {
	// Create a custom HTTP client with the specified timeout and TLS settings.
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		},
	}

	// Create a new HTTP request with the custom User-Agent header.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", url, err)
		return
	}
	req.Header.Set("User-Agent", userAgent)

	// Perform the HTTP request.
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()

	// Extract response details.
	statusCode := resp.StatusCode

	// Try to get Content-Length from header first, fallback to resp.ContentLength
	contentLength := resp.ContentLength
	if contentLengthHeader := resp.Header.Get("Content-Length"); contentLengthHeader != "" {
		trimmedHeader := strings.TrimSpace(contentLengthHeader)
		if parsedLength, err := strconv.ParseInt(trimmedHeader, 10, 64); err == nil {
			contentLength = parsedLength
		}
	}

	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])

	suffix, exists := validExtensions[contentType]
	if !exists {
		suffix = ""
	}

	// Read and discard the response body to free up resources
	// If Content-Length is still -1, calculate it from the body size
	bytesRead, _ := io.Copy(io.Discard, resp.Body)
	if contentLength == -1 && bytesRead > 0 {
		contentLength = bytesRead
	}

	// Apply matchers to filter the response.
	if !matches(fmt.Sprintf("%d", statusCode), options.MatchCode) {
		return // Skip if status code does not match.
	}
	if !matches(fmt.Sprintf("%d", contentLength), options.MatchLength) {
		return // Skip if content length does not match.
	}
	if !matches(contentType, options.MatchType) {
		return // Skip if content type does not match.
	}
	if !matches(strings.Trim(suffix, "[]"), options.MatchSuffix) {
		return // Skip if suffix does not match.
	}

	// Apply filters to exclude matching responses.
	if options.FilterCode != "" && matches(fmt.Sprintf("%d", statusCode), options.FilterCode) {
		return // Skip if status code matches filter (exclude).
	}
	if options.FilterLength != "" && matches(fmt.Sprintf("%d", contentLength), options.FilterLength) {
		return // Skip if content length matches filter (exclude).
	}
	if options.FilterType != "" && matches(contentType, options.FilterType) {
		return // Skip if content type matches filter (exclude).
	}
	if options.FilterSuffix != "" && matches(strings.Trim(suffix, "[]"), options.FilterSuffix) {
		return // Skip if suffix matches filter (exclude).
	}

	// Handle JSON output.
	if jsonOutput {
		output := JSONOutput{
			Host: url,
			Type: "REQUEST BASED",
		}
		output.Data.StatusCode = int64(statusCode)
		output.Data.ContentLength = int64(contentLength)
		output.Data.ContentType = contentType
		output.Data.Suffix = strings.Trim(suffix, "[]") // Remove brackets.

		var jsonData []byte
		if jsonTypeFlag == "Marshal" {
			jsonData, _ = json.Marshal(output)
		} else {
			jsonData, _ = json.MarshalIndent(output, "", "  ") // Pretty print the JSON.
		}
		fmt.Println(string(jsonData))
		if outputFile != nil {
			outputFile.WriteString(string(jsonData) + "\n")
		}
		return
	}

	// Handle non-verbose and verbose output.
	outputLine := ""
	if verbose {
		if options.NoColor {
			outputLine = fmt.Sprintf("REQUEST BASED: %s [%d] [%d] [%s] %s\n", url, statusCode, contentLength, contentType, suffix)
		} else {
			outputLine = fmt.Sprintf("%s: %s [%d] [%d] [%s] %s\n", aurora.Bold(aurora.Blue("REQUEST BASED")), url, aurora.Green(statusCode), aurora.Magenta(contentLength), aurora.Magenta(contentType), aurora.Yellow(suffix))
		}
	} else {
		if options.NoColor {
			outputLine = fmt.Sprintf("%s [%d] [%d] [%s] %s\n", url, statusCode, contentLength, contentType, suffix)
		} else {
			outputLine = fmt.Sprintf("%s [%d] [%d] [%s] %s\n", url, aurora.Green(statusCode), aurora.Magenta(contentLength), aurora.Magenta(contentType), aurora.Yellow(suffix))
		}
	}
	fmt.Print(outputLine)
	if outputFile != nil {
		outputFile.WriteString(outputLine)
	}
}

// Skip requests based on file extensions when the -passive flag is true.
func processURL(url string, passive bool, verbose bool, timeout time.Duration, insecure bool, userAgent string, wg *sync.WaitGroup, sem chan struct{}, delay time.Duration, jsonOutput bool, jsonTypeFlag string, outputFile *os.File, options *Options, passiveExtensions map[string]string, validExtensions map[string]string) {
	defer wg.Done()
	// Acquire a spot in the semaphore
	sem <- struct{}{}

	defer func() {
		// Release the spot in the semaphore when done
		<-sem
	}()

	// Check if the URL ends with one of the passive extensions
	for extGroup, label := range passiveExtensions {
		// Split the extensions into a slice
		extensions := strings.Split(extGroup, ", ")
		for _, ext := range extensions {
			if strings.HasSuffix(url, ext) {
				if passive {
					// Apply FilterSuffix filter if specified
					if options.FilterSuffix != "" && matches(strings.Trim(label, "[]"), options.FilterSuffix) {
						return // Skip if suffix matches filter (exclude).
					}
					// If passive mode is on, just print the URL and its label.
					if jsonOutput {
						output := JSONOutput{
							Host: url,
							Type: "EXTENSION BASED",
						}
						output.Data.Suffix = strings.Trim(label, "[]") // Remove both brackets

						var jsonData []byte
						if jsonTypeFlag == "Marshal" {
							jsonData, _ = json.Marshal(output)
						} else {
							jsonData, _ = json.MarshalIndent(output, "", "  ") // Pretty print the JSON
						}
						fmt.Println(string(jsonData))
						if outputFile != nil {
							outputFile.WriteString(string(jsonData) + "\n")
						}
						return
					}

					// Handle non-verbose and verbose output.
					outputLine := ""
					if verbose {
						if options.NoColor {
							outputLine = fmt.Sprintf("EXTENSION BASED: %s %s\n", url, label)
						} else {
							outputLine = fmt.Sprintf("%s: %s %s\n", aurora.Cyan("EXTENSION BASED"), url, aurora.Yellow(label))
						}
					} else {
						if options.NoColor {
							outputLine = fmt.Sprintf("%s %s\n", url, label)
						} else {
							outputLine = fmt.Sprintf("%s %s\n", url, aurora.Yellow(label))
						}
					}
					fmt.Print(outputLine)
					if outputFile != nil {
						outputFile.WriteString(outputLine)
					}

					time.Sleep(delay) // Apply delay between requests
					return
				}
			}
		}
	}

	// If not passive or extension not in map, proceed with the request.
	getURLInfo(url, verbose, timeout, insecure, userAgent, jsonOutput, jsonTypeFlag, outputFile, options, validExtensions)
	time.Sleep(delay) // Apply delay between requests
}

func main() {
	// Command-line flags
	options := ParseOptions()

	if options.Version {
		banner.PrintBanner()
		banner.PrintVersion()
		return
	}

	if !options.Silent {
		banner.PrintBanner()
	}

	// Load configuration from extensions.yaml
	config := loadConfig()

	// Convert Timeout to a time.Duration
	timeout := time.Duration(options.Timeout) * time.Second

	// Set up a WaitGroup and a semaphore (channel) to control concurrency
	var wg sync.WaitGroup
	sem := make(chan struct{}, options.Threads)

	var outputFile *os.File
	var err error
	if options.Output != "" {
		outputFile, err = os.OpenFile(options.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error opening output file %s: %v\n", options.Output, err)
			return
		}
		defer outputFile.Close()
	}

	if options.InputTargetHost != "" {
		wg.Add(1)
		go processURL(options.InputTargetHost, options.Passive, options.Verbose, timeout, options.Insecure, options.UserAgent, &wg, sem, options.Delay, options.JSONOutput, options.JSONtype, outputFile, options, config.PassiveExtensions, config.ValidExtensions)
		wg.Wait()
		return
	}

	if options.InputFile != "" {
		file, err := os.Open(options.InputFile)
		if err != nil {
			fmt.Printf("Error opening file %s: %v\n", options.InputFile, err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			url := strings.TrimSpace(scanner.Text())
			if url != "" {
				wg.Add(1)
				go processURL(url, options.Passive, options.Verbose, timeout, options.Insecure, options.UserAgent, &wg, sem, options.Delay, options.JSONOutput, options.JSONtype, outputFile, options, config.PassiveExtensions, config.ValidExtensions)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file: %v\n", err)
		}
		wg.Wait()
		return
	}

	// Read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			wg.Add(1)
			go processURL(url, options.Passive, options.Verbose, timeout, options.Insecure, options.UserAgent, &wg, sem, options.Delay, options.JSONOutput, options.JSONtype, outputFile, options, config.PassiveExtensions, config.ValidExtensions)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading stdin: %v\n", err)
	}
	wg.Wait()
}
