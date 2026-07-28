package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var suggestionsCmd = &cobra.Command{
	Use:   "suggestions",
	Short: "AI conversation and send-time suggestions",
	Long:  "Commands for analyzing conversations and computing optimal send times using the Scheduler0 AI suggestions engine.",
}

var suggestionsAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze a conversation for suggestions and obligations",
	Long: `Analyze a conversation and return structured, explainable suggestions
(commitments, requests, deadlines, follow-ups, etc.) plus the obligations they map to.

The request body (conversation messages, participants, options) is read as JSON
from the --file flag, or from stdin when --file is omitted or set to '-'.`,
	RunE: runSuggestionsAnalyze,
}

var suggestionsTimeCmd = &cobra.Command{
	Use:   "time",
	Short: "Recommend optimal future send times for a message",
	Long: `Recommend suitable future send times for a message given sender/recipient time
zones, working hours, quiet hours, weekends, priority, and coverage rules. The engine
is deterministic and does not send the message or create a job.

The request body is read as JSON from the --file flag, or from stdin when --file
is omitted or set to '-'.`,
	RunE: runSuggestionsTime,
}

func init() {
	rootCmd.AddCommand(suggestionsCmd)
	suggestionsCmd.AddCommand(suggestionsAnalyzeCmd)
	suggestionsCmd.AddCommand(suggestionsTimeCmd)

	suggestionsAnalyzeCmd.Flags().String("file", "", "Path to a JSON file with the request body; reads stdin when omitted or '-'")
	suggestionsTimeCmd.Flags().String("file", "", "Path to a JSON file with the request body; reads stdin when omitted or '-'")
}

// readJSONInput reads request-body JSON from the --file flag, or from stdin when
// the flag is empty or "-".
func readJSONInput(cmd *cobra.Command) ([]byte, error) {
	file, _ := cmd.Flags().GetString("file")
	if file == "" || file == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body from stdin: %w", err)
		}
		return data, nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body from %s: %w", file, err)
	}
	return data, nil
}

func runSuggestionsAnalyze(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	raw, err := readJSONInput(cmd)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return fmt.Errorf("request body is empty; provide JSON via --file or stdin")
	}

	var request scheduler0_client.AnalyzeSuggestionsRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return fmt.Errorf("failed to parse request JSON: %w", err)
	}

	result, err := cl.AnalyzeSuggestions(&request)
	if err != nil {
		return fmt.Errorf("failed to analyze suggestions: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runSuggestionsTime(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	raw, err := readJSONInput(cmd)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return fmt.Errorf("request body is empty; provide JSON via --file or stdin")
	}

	var request scheduler0_client.SendTimeSuggestionsRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return fmt.Errorf("failed to parse request JSON: %w", err)
	}

	result, err := cl.SendTimeSuggestions(&request)
	if err != nil {
		return fmt.Errorf("failed to get send time suggestions: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
