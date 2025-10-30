package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bcspragu/kagi/api"

	"github.com/pkg/errors"
)

var (
	errUsage         = errors.New("usage: kagi [flags] query")
	errMissingAPIKey = errors.New("missing Kagi API key")
)

func main() {
	command, err := newCommand(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		command.flags.Usage()
		os.Exit(1)
	}

	if err := invoke(command); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

type Command struct {
	query      string
	KagiAPIKey string
	cacheDir   string
	verbose    bool
	flags      flag.FlagSet
}

func newCommand(args []string) (command Command, err error) {
	var flags = flag.NewFlagSet(args[0], flag.ExitOnError)
	command.flags = *flags

	var (
		kagiAPIKey = flags.String("kagi_api_key", os.Getenv("KAGI_API_KEY"), "API key to use with the Kagi FastGPT API")
		cacheDir   = flags.String("cache_dir", "", "Directory to cache API responses in.  If not set, responses will not be cached.")
		verbose    = flags.Bool("verbose", false, "Enable verbose logging")
	)

	if len(os.Args) == 0 {
		return command, errUsage
	}

	if err := flags.Parse(args[1:]); err != nil {
		return command, fmt.Errorf("failed to parse flags: %w", errors.Wrap(err, errUsage.Error()))
	}

	if kagiAPIKey == nil || *kagiAPIKey == "" {
		return command, errMissingAPIKey
	}

	command.KagiAPIKey = *kagiAPIKey
	command.cacheDir = *cacheDir
	command.verbose = *verbose

	// If no arguments provided, read from stdin
	if flags.NArg() == 0 {
		// Check if stdin is a terminal (interactive) to show a prompt
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprint(os.Stderr, "Enter your query (Ctrl+D when done):\n")
		}

		// Read from stdin (works for both piped input and interactive terminal)
		reader := bufio.NewReader(os.Stdin)
		input, err := io.ReadAll(reader)
		if err != nil {
			return command, fmt.Errorf("failed to read from stdin: %w", err)
		}
		command.query = strings.TrimSpace(string(input))
		if command.query == "" {
			return command, errUsage
		}
	} else {
		command.query = strings.Join(flags.Args(), " ")
	}

	return command, nil
}

func invoke(command Command) error {
	client := api.NewClient(command.KagiAPIKey)

	req := api.FastGPTRequest{
		Query:     command.query,
		WebSearch: true,
		Cache:     true,
	}
	// Log the request if verbose is enabled
	if command.verbose {
		log.Printf("Request: %+v\n", req)
	}

	resp, err := client.FastGPTRequest(req)
	if err != nil {
		return fmt.Errorf("error performing query: %w", err)
	}

	response := respond(resp, command.query)

	// Send response to stdout
	fmt.Print(response)

	if command.cacheDir != "" {
		cache(command.cacheDir, command.query, response)
	}
	return nil
}

func respond(resp *api.FastGPTResponse, query string) (response string) {
	// remove all repeated newlines or empty lines from the output
	answer := strings.ReplaceAll(resp.Data.Output, "\n\n", "\n")

	response = "# " + query + "\n" + answer + "\n"

	// If there are no references, return early
	if len(resp.Data.References) == 0 {
		return
	}

	response += "\n# References\n"

	for i, ref := range resp.Data.References {
		response += fmt.Sprintf("%d. %s - %s  - %s\n", i+1, ref.Title, ref.Link, ref.Snippet)
	}

	return
}

type CacheEntry struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

func cache(cacheDir string, question string, answer string) error {
	// create cache directory if it doesn't exist
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	// write response to cache file
	// filename is a sha256 hash of the query with a json extension
	// the filecontent is the json response from the API and the query
	entry := CacheEntry{
		Question: question,
		Answer:   answer,
	}

	jsonEntry, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	cacheFile := fmt.Sprintf("%s/%s.json", cacheDir, fmt.Sprintf("%x", sha256.Sum256([]byte(question)))[0:8])
	if err := os.WriteFile(cacheFile, jsonEntry, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}
