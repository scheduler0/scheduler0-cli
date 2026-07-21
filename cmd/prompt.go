package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Create job configurations from AI prompt",
	Long:  "Generate job configurations from a natural language prompt using AI. This endpoint requires credits (1 credit per prompt).",
	RunE:  runPrompt,
}

// classifyCmd is a subcommand that runs only the intent guardrail — no model execution
// and no credits are consumed.
var classifyCmd = &cobra.Command{
	Use:   "classify",
	Short: "Classify a prompt with the intent guardrail (no AI execution, no credits consumed)",
	Long:  "Run only the edge intent classifier against a prompt and print the full classification output. No AI model is invoked and no credits are consumed.",
	RunE:  runClassify,
}

func init() {
	rootCmd.AddCommand(promptCmd)

	promptCmd.Flags().String("prompt", "", "Natural language prompt describing the jobs to create (required, max 160 characters)")
	promptCmd.Flags().StringSlice("purposes", []string{}, "Array of business purposes (max 5 items, each max 36 characters)")
	promptCmd.Flags().StringSlice("events", []string{}, "Array of event types (max 5 items, each max 36 characters)")
	promptCmd.Flags().StringSlice("recipients", []string{}, "Array of recipient identifiers (max 5 items, each max 36 characters)")
	promptCmd.Flags().StringSlice("channels", []string{}, "Array of delivery channels (max 5 items, each max 36 characters)")
	promptCmd.Flags().String("timezone", "", "Optional IANA timezone for scheduling calculations (e.g., UTC, America/New_York). Defaults to UTC when omitted; invalid values are rejected by the API.")
	promptCmd.Flags().String("locale", "", "Optional BCP-47 locale (e.g., en-US, es-ES) forwarded to the AI model so natural-language output is written in that language. Defaults to en when omitted.")
	_ = promptCmd.MarkFlagRequired("prompt")

	// classify subcommand
	promptCmd.AddCommand(classifyCmd)
	classifyCmd.Flags().String("prompt", "", "Natural language prompt to classify (required)")
	classifyCmd.Flags().String("locale", "", "Optional BCP-47 locale. The intent classifier is English-only, so any non-English (en*) value is rejected by the API with 400.")
	_ = classifyCmd.MarkFlagRequired("prompt")
}

func runPrompt(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	prompt, _ := cmd.Flags().GetString("prompt")
	purposes, _ := cmd.Flags().GetStringSlice("purposes")
	events, _ := cmd.Flags().GetStringSlice("events")
	recipients, _ := cmd.Flags().GetStringSlice("recipients")
	channels, _ := cmd.Flags().GetStringSlice("channels")
	timezone, _ := cmd.Flags().GetString("timezone")
	locale, _ := cmd.Flags().GetString("locale")

	if len(strings.TrimSpace(prompt)) == 0 {
		return fmt.Errorf("prompt cannot be empty")
	}
	if len(prompt) > 160 {
		return fmt.Errorf("prompt must be 160 characters or less (got %d)", len(prompt))
	}
	if len(purposes) > 5 {
		return fmt.Errorf("purposes must have 5 or fewer items (got %d)", len(purposes))
	}
	if len(events) > 5 {
		return fmt.Errorf("events must have 5 or fewer items (got %d)", len(events))
	}
	if len(recipients) > 5 {
		return fmt.Errorf("recipients must have 5 or fewer items (got %d)", len(recipients))
	}
	if len(channels) > 5 {
		return fmt.Errorf("channels must have 5 or fewer items (got %d)", len(channels))
	}

	maxLength := 36
	for i, purpose := range purposes {
		if len(purpose) > maxLength {
			return fmt.Errorf("purpose[%d] must be %d characters or less (got %d)", i, maxLength, len(purpose))
		}
	}
	for i, event := range events {
		if len(event) > maxLength {
			return fmt.Errorf("event[%d] must be %d characters or less (got %d)", i, maxLength, len(event))
		}
	}
	for i, recipient := range recipients {
		if len(recipient) > maxLength {
			return fmt.Errorf("recipient[%d] must be %d characters or less (got %d)", i, maxLength, len(recipient))
		}
	}
	for i, channel := range channels {
		if len(channel) > maxLength {
			return fmt.Errorf("channel[%d] must be %d characters or less (got %d)", i, maxLength, len(channel))
		}
	}

	request := &scheduler0_client.PromptJobRequest{
		Prompt: strings.TrimSpace(prompt),
	}
	if len(purposes) > 0 {
		request.Purposes = purposes
	}
	if len(events) > 0 {
		request.Events = events
	}
	if len(recipients) > 0 {
		request.Recipients = recipients
	}
	if len(channels) > 0 {
		request.Channels = channels
	}
	if timezone != "" {
		request.Timezone = timezone
	}
	if strings.TrimSpace(locale) != "" {
		request.Locale = strings.TrimSpace(locale)
	}

	result, err := cl.CreateJobFromPrompt(request)
	if err != nil {
		// When the intent guardrail rejects the prompt, surface the structured
		// classification so the user can see why and how to rephrase.
		var skipped *scheduler0_client.PromptSkippedError
		if errors.As(err, &skipped) {
			fmt.Printf("Prompt %s: %s\n", skipped.Classification.Decision, skipped.Classification.Reason)
			if skipped.Classification != nil {
				out, _ := json.MarshalIndent(skipped.Classification, "", "  ")
				fmt.Println(string(out))
			}
			return fmt.Errorf("prompt not accepted by intent guardrail (decision=%s)", skipped.Classification.Decision)
		}
		return fmt.Errorf("failed to create job from prompt: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	fmt.Println("\nNote: These are job configurations generated by AI. Use 'scheduler0 jobs create' to create actual jobs from these configurations.")
	return nil
}

func runClassify(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	prompt, _ := cmd.Flags().GetString("prompt")
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	locale, _ := cmd.Flags().GetString("locale")

	classification, err := cl.ClassifyPrompt(&scheduler0_client.ClassifyPromptRequest{
		Prompt: strings.TrimSpace(prompt),
		Locale: strings.TrimSpace(locale),
	})
	if err != nil {
		return fmt.Errorf("failed to classify prompt: %w", err)
	}

	output, _ := json.MarshalIndent(classification, "", "  ")
	fmt.Println(string(output))
	return nil
}
