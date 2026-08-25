package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage accounts",
	Long:  "Commands for managing Scheduler0 accounts",
}

var accountsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new account",
	Long:  "Create a new account in Scheduler0",
	RunE:  runAccountsCreate,
}

var accountsGetCmd = &cobra.Command{
	Use:   "get [account-id]",
	Short: "Get account details",
	Long:  "Get details of a specific account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsGet,
}

var accountsUpdateCmd = &cobra.Command{
	Use:   "update [account-id]",
	Short: "Update an account",
	Long:  "Rename an account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsUpdate,
}

var accountsExecutionCountCmd = &cobra.Command{
	Use:   "execution-count",
	Short: "Manage account execution count",
	Long:  "Commands for reading and increasing an account's monthly execution count",
}

var accountsExecutionCountGetCmd = &cobra.Command{
	Use:   "get [account-id]",
	Short: "Get account execution count",
	Long:  "Get the current execution count and next reset date for an account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsExecutionCountGet,
}

var accountsExecutionCountIncreaseCmd = &cobra.Command{
	Use:   "increase [account-id]",
	Short: "Increase account execution count",
	Long:  "Increase the execution count for an account by the given amount",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsExecutionCountIncrease,
}

var accountsAIUsageCmd = &cobra.Command{
	Use:   "ai-usage [account-id]",
	Short: "Get account AI request usage",
	Long:  "Get the account's log-derived AI prompt/classify usage for the current period (limits, used, remaining, estimated cost).",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsAIUsage,
}

var accountsFeatureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage a single feature on an account",
	Long:  "Commands for adding/removing a single feature on an account",
}

var accountsFeatureAddCmd = &cobra.Command{
	Use:   "add [account-id]",
	Short: "Add a feature to an account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsFeatureAdd,
}

var accountsFeatureRemoveCmd = &cobra.Command{
	Use:   "remove [account-id]",
	Short: "Remove a feature from an account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsFeatureRemove,
}

var accountsFeaturesAllCmd = &cobra.Command{
	Use:   "features-all [account-id]",
	Short: "Add or remove every feature on an account",
	Long:  "Commands for adding/removing every known feature on an account at once",
	Args:  cobra.ExactArgs(1),
}

var accountsFeaturesAllAddCmd = &cobra.Command{
	Use:   "add [account-id]",
	Short: "Add every feature to an account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsFeaturesAllAdd,
}

var accountsFeaturesAllRemoveCmd = &cobra.Command{
	Use:   "remove [account-id]",
	Short: "Remove every feature from an account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsFeaturesAllRemove,
}

var accountsTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Manage account token balance",
	Long:  "Commands for managing Scheduler0 account token balance",
}

var accountsTokensGetCmd = &cobra.Command{
	Use:   "get [account-id]",
	Short: "Get account token balance",
	Long:  "Get the current token balance for an account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsTokensGet,
}

var accountsTokensAddCmd = &cobra.Command{
	Use:   "add [account-id]",
	Short: "Add tokens to account balance",
	Long:  "Add the given number of tokens to an account's balance",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsTokensAdd,
}

var accountsGenerateSecretKeyCmd = &cobra.Command{
	Use:   "generate-secret-key",
	Short: "Generate random AES-256 secret keys for use during secret rotation",
	Long: `Generate one or more cryptographically random 64-character hex strings suitable
for use as a Scheduler0 SecretKey (AES-256).

This command runs entirely offline — no API call or authentication is required.
Use it as the first step of the key-rotation workflow:

  1. scheduler0 accounts generate-secret-key                    # produce a new key
  2. Update SecretKey in your secrets source (file, env var, SSM, etc.) and reload the server
  3. scheduler0 accounts rotate-secret --old-secret-key <old>   # re-encrypt stored secrets

Use --count to generate multiple keys in a single call, which is convenient
when you want to stage several rotations (A → B → C) without re-running the
command each time.`,
	RunE: runAccountsGenerateSecretKey,
}

var accountsRotateSecretCmd = &cobra.Command{
	Use:   "rotate-secret",
	Short: "Re-encrypt stored secrets with a new SecretKey (self-hosting only)",
	Long: `Re-encrypt all secrets stored under the server's SecretKey — credential api
secrets, executor cloud provider credentials, and per-account AI provider keys —
from the old SecretKey to the server's currently-loaded (new) one.

Credentials keep working across rotation: the api_secret is stored encrypted and
verified by decrypting it at auth time, and the api_key identifier is left unchanged,
so re-encrypting the stored secret does not affect the secret the client holds.

This command is for operators who have rotated the server's SecretKey in their
secrets source (file, SSM, AWS Secrets Manager, or env var).

Steps:
  1. Generate a new key (scheduler0 accounts generate-secret-key).
  2. Update SecretKey in your secrets source and reload/restart the server so the
     new key is the one loaded in memory.
  3. Run this command with the PREVIOUS key: --old-secret-key <old>. It calls
     POST /api/v1/account/rotate-secret on the server, which decrypts every stored
     secret with the old key and re-encrypts it with the loaded new key.

Requires an admin-scoped session: sign in as an operator with 'scheduler0 login'
(or set the SCHEDULER0_* environment variables for a CI credential carrying the
admin scope). There are no --username/--password flags; the API is authenticated
with the signed-in credential.`,
	RunE: runAccountsRotateSecret,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.AddCommand(accountsCreateCmd)
	accountsCmd.AddCommand(accountsGetCmd)
	accountsCmd.AddCommand(accountsUpdateCmd)
	accountsCmd.AddCommand(accountsExecutionCountCmd)
	accountsExecutionCountCmd.AddCommand(accountsExecutionCountGetCmd)
	accountsExecutionCountCmd.AddCommand(accountsExecutionCountIncreaseCmd)
	accountsCmd.AddCommand(accountsAIUsageCmd)
	accountsCmd.AddCommand(accountsFeatureCmd)
	accountsFeatureCmd.AddCommand(accountsFeatureAddCmd)
	accountsFeatureCmd.AddCommand(accountsFeatureRemoveCmd)
	accountsCmd.AddCommand(accountsFeaturesAllCmd)
	accountsFeaturesAllCmd.AddCommand(accountsFeaturesAllAddCmd)
	accountsFeaturesAllCmd.AddCommand(accountsFeaturesAllRemoveCmd)
	accountsCmd.AddCommand(accountsTokensCmd)
	accountsTokensCmd.AddCommand(accountsTokensGetCmd)
	accountsTokensCmd.AddCommand(accountsTokensAddCmd)
	accountsCmd.AddCommand(accountsRotateSecretCmd)
	accountsCmd.AddCommand(accountsGenerateSecretKeyCmd)

	accountsCreateCmd.Flags().String("name", "", "Account name (required)")
	_ = accountsCreateCmd.MarkFlagRequired("name")

	accountsUpdateCmd.Flags().String("name", "", "New account name (required)")
	_ = accountsUpdateCmd.MarkFlagRequired("name")

	accountsExecutionCountIncreaseCmd.Flags().Uint64("count", 0, "Amount to increase the execution count by (required)")
	_ = accountsExecutionCountIncreaseCmd.MarkFlagRequired("count")

	accountsFeatureAddCmd.Flags().Int64("feature-id", 0, "ID of the feature to add (required)")
	_ = accountsFeatureAddCmd.MarkFlagRequired("feature-id")

	accountsFeatureRemoveCmd.Flags().Int64("feature-id", 0, "ID of the feature to remove (required)")
	_ = accountsFeatureRemoveCmd.MarkFlagRequired("feature-id")

	accountsTokensAddCmd.Flags().Int64("amount", 0, "Number of tokens to add (must be > 0, required)")
	_ = accountsTokensAddCmd.MarkFlagRequired("amount")

	accountsRotateSecretCmd.Flags().String("old-secret-key", "", "The previous SecretKey used to decrypt existing secrets (required)")
	_ = accountsRotateSecretCmd.MarkFlagRequired("old-secret-key")

	accountsGenerateSecretKeyCmd.Flags().Int("count", 1, "Number of secret keys to generate")
}

func runAccountsCreate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")

	account := &scheduler0_client.AccountCreateRequestBody{
		Name: name,
	}

	result, err := cl.CreateAccount(account)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	result, err := cl.GetAccount(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	name, _ := cmd.Flags().GetString("name")

	result, err := cl.UpdateAccount(accountID, &scheduler0_client.AccountUpdateRequestBody{Name: name})
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsExecutionCountGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	result, err := cl.GetAccountExecutionCount(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account execution count: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsExecutionCountIncrease(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	count, _ := cmd.Flags().GetUint64("count")

	result, err := cl.IncreaseAccountExecutionCount(accountID, count)
	if err != nil {
		return fmt.Errorf("failed to increase account execution count: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsAIUsage(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	result, err := cl.GetAIUsage(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account AI usage: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsFeatureAdd(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	featureID, _ := cmd.Flags().GetInt64("feature-id")

	result, err := cl.AddFeatureToAccount(accountID, &scheduler0_client.FeatureRequest{FeatureID: featureID})
	if err != nil {
		return fmt.Errorf("failed to add feature to account: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsFeatureRemove(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	featureID, _ := cmd.Flags().GetInt64("feature-id")

	if err := cl.RemoveFeatureFromAccount(accountID, &scheduler0_client.FeatureRequest{FeatureID: featureID}); err != nil {
		return fmt.Errorf("failed to remove feature from account: %w", err)
	}

	fmt.Printf("Feature %d removed from account %s\n", featureID, accountID)
	return nil
}

func runAccountsFeaturesAllAdd(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	if err := cl.AddAllFeaturesToAccount(accountID); err != nil {
		return fmt.Errorf("failed to add all features to account: %w", err)
	}

	fmt.Printf("All features added to account %s\n", accountID)
	return nil
}

func runAccountsFeaturesAllRemove(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	if err := cl.RemoveAllFeaturesFromAccount(accountID); err != nil {
		return fmt.Errorf("failed to remove all features from account: %w", err)
	}

	fmt.Printf("All features removed from account %s\n", accountID)
	return nil
}

func runAccountsTokensGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	result, err := cl.GetAccountTokens(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account tokens: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runAccountsTokensAdd(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	accountID := args[0]
	amount, _ := cmd.Flags().GetInt64("amount")

	result, err := cl.AddAccountTokens(accountID, amount)
	if err != nil {
		return fmt.Errorf("failed to add account tokens: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

// runAccountsRotateSecret calls POST /account/rotate-secret on the server to re-encrypt
// credential api secrets, executor cloud credentials, and AI provider keys from the old
// SecretKey to the server's currently-loaded new one. Requires an admin-scoped session.
func runAccountsRotateSecret(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	// rotate-secret is an operator (admin) operation; the signed-in credential must
	// carry the admin scope, which the API enforces.
	oldSecretKey, _ := cmd.Flags().GetString("old-secret-key")
	if strings.TrimSpace(oldSecretKey) == "" {
		return fmt.Errorf("--old-secret-key is required")
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Starting secret rotation...")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Ensure you have already updated SecretKey in your secrets source and reloaded the server before proceeding.")

	result, err := cl.RotateSecret(oldSecretKey)
	if err != nil {
		return fmt.Errorf("rotation failed: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Rotation complete: %d credential(s), %d executor(s), and %d AI setting(s) re-encrypted.\n", result.Data.CredentialsRotated, result.Data.ExecutorsRotated, result.Data.AISettingsRotated)
	return nil
}

func runAccountsGenerateSecretKey(cmd *cobra.Command, args []string) error {
	count, _ := cmd.Flags().GetInt("count")
	if count < 1 {
		return fmt.Errorf("--count must be at least 1")
	}

	keys := make([]string, count)
	for i := range keys {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return fmt.Errorf("failed to generate random bytes: %w", err)
		}
		keys[i] = hex.EncodeToString(b)
	}

	output, _ := json.MarshalIndent(map[string][]string{"secret_keys": keys}, "", "  ")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(output))
	return nil
}
