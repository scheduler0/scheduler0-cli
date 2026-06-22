package cmd

import (
	"encoding/json"
	"fmt"

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

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.AddCommand(accountsCreateCmd)
	accountsCmd.AddCommand(accountsGetCmd)
	accountsCmd.AddCommand(accountsTokensCmd)
	accountsTokensCmd.AddCommand(accountsTokensGetCmd)
	accountsTokensCmd.AddCommand(accountsTokensAddCmd)

	accountsCreateCmd.Flags().String("name", "", "Account name (required)")
	accountsCreateCmd.MarkFlagRequired("name")

	accountsTokensAddCmd.Flags().Int64("amount", 0, "Number of tokens to add (must be > 0, required)")
	accountsTokensAddCmd.MarkFlagRequired("amount")
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


