package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var aiSettingsCmd = &cobra.Command{
	Use:   "ai-settings",
	Short: "Manage account AI settings",
	Long:  "Commands for managing Scheduler0 account AI provider settings",
}

var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List approved AI models",
	Long:  "List the per-provider approved model catalog from GET /ai/models",
	RunE:  runAIModels,
}

var aiSettingsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get account AI settings",
	Long:  "Get the AI provider settings for the current account",
	RunE:  runAISettingsGet,
}

var aiSettingsUpsertCmd = &cobra.Command{
	Use:   "upsert",
	Short: "Create or update account AI settings",
	Long: `Create or update the AI provider settings for the current account.

Use --active-models to supply an ordered JSON array of {provider, model} objects.
The first entry is the primary model; the rest are fallback models tried in order.

Example:
  scheduler0 ai-settings upsert \
    --active-models '[{"provider":"openai","model":"gpt-4.1-mini"},{"provider":"anthropic","model":"claude-sonnet-4-5"}]' \
    --openai-api-key sk-... \
    --anthropic-api-key sk-ant-...
`,
	RunE: runAISettingsUpsert,
}

func init() {
	rootCmd.AddCommand(aiSettingsCmd)
	aiSettingsCmd.AddCommand(aiSettingsGetCmd)
	aiSettingsCmd.AddCommand(aiSettingsUpsertCmd)
	aiSettingsCmd.AddCommand(aiModelsCmd)

	aiSettingsGetCmd.Flags().String("account-id", "", "Account ID (overrides configured account)")

	aiSettingsUpsertCmd.Flags().String("account-id", "", "Account ID (overrides configured account)")
	aiSettingsUpsertCmd.Flags().String("active-models", "", `JSON array of active models, e.g. '[{"provider":"openai","model":"gpt-4.1-mini"}]'`)
	aiSettingsUpsertCmd.Flags().String("openai-api-key", "", "OpenAI API key")
	aiSettingsUpsertCmd.Flags().String("anthropic-api-key", "", "Anthropic API key")
	aiSettingsUpsertCmd.Flags().String("bedrock-access-key-id", "", "AWS Bedrock access key ID")
	aiSettingsUpsertCmd.Flags().String("bedrock-secret-key", "", "AWS Bedrock secret key")
	aiSettingsUpsertCmd.Flags().String("bedrock-region", "", "AWS Bedrock region")
	aiSettingsUpsertCmd.Flags().String("openrouter-api-key", "", "OpenRouter API key")
}

func runAISettingsGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID, _ := cmd.Flags().GetString("account-id")
	result, err := cl.GetAccountAISettings(accountID)
	if err != nil {
		return fmt.Errorf("failed to get AI settings: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAISettingsUpsert(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID, _ := cmd.Flags().GetString("account-id")
	activeModelsJSON, _ := cmd.Flags().GetString("active-models")
	openAIAPIKey, _ := cmd.Flags().GetString("openai-api-key")
	anthropicAPIKey, _ := cmd.Flags().GetString("anthropic-api-key")
	bedrockAccessKeyID, _ := cmd.Flags().GetString("bedrock-access-key-id")
	bedrockSecretKey, _ := cmd.Flags().GetString("bedrock-secret-key")
	bedrockRegion, _ := cmd.Flags().GetString("bedrock-region")
	openRouterAPIKey, _ := cmd.Flags().GetString("openrouter-api-key")

	settings := &scheduler0_client.AccountAISettings{
		OpenAIAPIKey:       openAIAPIKey,
		AnthropicAPIKey:    anthropicAPIKey,
		BedrockAccessKeyID: bedrockAccessKeyID,
		BedrockSecretKey:   bedrockSecretKey,
		BedrockRegion:      bedrockRegion,
		OpenRouterAPIKey:   openRouterAPIKey,
	}

	// Parse --active-models JSON if provided.
	if activeModelsJSON != "" {
		var activeModels []scheduler0_client.ActiveModel
		if parseErr := json.Unmarshal([]byte(activeModelsJSON), &activeModels); parseErr != nil {
			return fmt.Errorf("invalid --active-models JSON: %w", parseErr)
		}
		settings.ActiveModels = activeModels
	}

	result, err := cl.UpsertAccountAISettings(accountID, settings)
	if err != nil {
		return fmt.Errorf("failed to upsert AI settings: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAIModels(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	result, err := cl.GetAIModels()
	if err != nil {
		return fmt.Errorf("failed to fetch AI models: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
