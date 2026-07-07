package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
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

Requires Basic Authentication (--username / --password or saved config with auth_type=basic).`,
	RunE: runAccountsRotateSecret,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.AddCommand(accountsCreateCmd)
	accountsCmd.AddCommand(accountsGetCmd)
	accountsCmd.AddCommand(accountsTokensCmd)
	accountsTokensCmd.AddCommand(accountsTokensGetCmd)
	accountsTokensCmd.AddCommand(accountsTokensAddCmd)
	accountsCmd.AddCommand(accountsRotateSecretCmd)
	accountsCmd.AddCommand(accountsGenerateSecretKeyCmd)

	accountsCreateCmd.Flags().String("name", "", "Account name (required)")
	_ = accountsCreateCmd.MarkFlagRequired("name")

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
// SecretKey to the server's currently-loaded new one. Requires basic auth.
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
