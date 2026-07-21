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

var aiSettingsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get account AI settings",
	Long:  "Get the AI provider settings for the current account",
	RunE:  runAISettingsGet,
}

var aiSettingsUpsertCmd = &cobra.Command{
	Use:   "upsert",
	Short: "Create or update account AI settings",
	Long:  "Create or update the AI provider settings for the current account",
	RunE:  runAISettingsUpsert,
}

func init() {
	rootCmd.AddCommand(aiSettingsCmd)
	aiSettingsCmd.AddCommand(aiSettingsGetCmd)
	aiSettingsCmd.AddCommand(aiSettingsUpsertCmd)

	aiSettingsGetCmd.Flags().String("account-id", "", "Account ID (overrides configured account)")

	aiSettingsUpsertCmd.Flags().String("account-id", "", "Account ID (overrides configured account)")
	aiSettingsUpsertCmd.Flags().String("provider", "", "AI provider (e.g. openai, anthropic, bedrock)")
	aiSettingsUpsertCmd.Flags().String("model", "", "AI model name")
	aiSettingsUpsertCmd.Flags().String("openai-api-key", "", "OpenAI API key")
	aiSettingsUpsertCmd.Flags().String("anthropic-api-key", "", "Anthropic API key")
	aiSettingsUpsertCmd.Flags().String("bedrock-access-key-id", "", "AWS Bedrock access key ID")
	aiSettingsUpsertCmd.Flags().String("bedrock-secret-key", "", "AWS Bedrock secret key")
	aiSettingsUpsertCmd.Flags().String("bedrock-region", "", "AWS Bedrock region")
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
	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	openAIAPIKey, _ := cmd.Flags().GetString("openai-api-key")
	anthropicAPIKey, _ := cmd.Flags().GetString("anthropic-api-key")
	bedrockAccessKeyID, _ := cmd.Flags().GetString("bedrock-access-key-id")
	bedrockSecretKey, _ := cmd.Flags().GetString("bedrock-secret-key")
	bedrockRegion, _ := cmd.Flags().GetString("bedrock-region")

	settings := &scheduler0_client.AccountAISettings{
		Provider:           provider,
		Model:              model,
		OpenAIAPIKey:       openAIAPIKey,
		AnthropicAPIKey:    anthropicAPIKey,
		BedrockAccessKeyID: bedrockAccessKeyID,
		BedrockSecretKey:   bedrockSecretKey,
		BedrockRegion:      bedrockRegion,
	}

	result, err := cl.UpsertAccountAISettings(accountID, settings)
	if err != nil {
		return fmt.Errorf("failed to upsert AI settings: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}
