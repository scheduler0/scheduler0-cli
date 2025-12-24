package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
	"github.com/spf13/cobra"
)

var credentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Manage credentials",
	Long:  "Commands for managing Scheduler0 credentials",
}

var credentialsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List credentials",
	Long:  "List all credentials with pagination",
	RunE:  runCredentialsList,
}

var credentialsGetCmd = &cobra.Command{
	Use:   "get [credential-id]",
	Short: "Get credential details",
	Long:  "Get details of a specific credential",
	Args:  cobra.ExactArgs(1),
	RunE:  runCredentialsGet,
}

var credentialsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new credential",
	Long:  "Create a new credential",
	RunE:  runCredentialsCreate,
}

var credentialsUpdateCmd = &cobra.Command{
	Use:   "update [credential-id]",
	Short: "Update a credential",
	Long:  "Update an existing credential",
	Args:  cobra.ExactArgs(1),
	RunE:  runCredentialsUpdate,
}

var credentialsDeleteCmd = &cobra.Command{
	Use:   "delete [credential-id]",
	Short: "Delete a credential",
	Long:  "Delete a credential",
	Args:  cobra.ExactArgs(1),
	RunE:  runCredentialsDelete,
}

var credentialsArchiveCmd = &cobra.Command{
	Use:   "archive [credential-id]",
	Short: "Archive a credential",
	Long:  "Archive a credential",
	Args:  cobra.ExactArgs(1),
	RunE:  runCredentialsArchive,
}

func init() {
	rootCmd.AddCommand(credentialsCmd)
	credentialsCmd.AddCommand(credentialsListCmd)
	credentialsCmd.AddCommand(credentialsGetCmd)
	credentialsCmd.AddCommand(credentialsCreateCmd)
	credentialsCmd.AddCommand(credentialsUpdateCmd)
	credentialsCmd.AddCommand(credentialsDeleteCmd)
	credentialsCmd.AddCommand(credentialsArchiveCmd)

	credentialsListCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	credentialsListCmd.Flags().Int("offset", 0, "Number of items to skip")
	credentialsListCmd.Flags().String("order-by", "date_created", "Field to order by")
	credentialsListCmd.Flags().String("order-direction", "desc", "Order direction (asc/desc)")

	credentialsCreateCmd.Flags().Bool("archived", false, "Whether the credential is archived")
	credentialsCreateCmd.Flags().String("created-by", "", "User who created the credential (required)")
	credentialsCreateCmd.MarkFlagRequired("created-by")

	credentialsUpdateCmd.Flags().Bool("archived", false, "Whether the credential is archived")
	credentialsUpdateCmd.Flags().String("modified-by", "", "User who modified the credential (required)")
	credentialsUpdateCmd.MarkFlagRequired("modified-by")

	credentialsDeleteCmd.Flags().String("deleted-by", "", "User who deleted the credential (required)")
	credentialsDeleteCmd.MarkFlagRequired("deleted-by")

	credentialsArchiveCmd.Flags().String("archived-by", "", "User who archived the credential (required)")
	credentialsArchiveCmd.MarkFlagRequired("archived-by")
}

func runCredentialsList(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	orderBy, _ := cmd.Flags().GetString("order-by")
	orderDirection, _ := cmd.Flags().GetString("order-direction")

	params := scheduler0_client.ListCredentialsParams{
		Limit:            limit,
		Offset:           offset,
		OrderBy:          orderBy,
		OrderByDirection: orderDirection,
	}

	result, err := cl.ListCredentials(params)
	if err != nil {
		return fmt.Errorf("failed to list credentials: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runCredentialsGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	credentialID := args[0]
	result, err := cl.GetCredential(credentialID)
	if err != nil {
		return fmt.Errorf("failed to get credential: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runCredentialsCreate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	archived, _ := cmd.Flags().GetBool("archived")
	createdBy, _ := cmd.Flags().GetString("created-by")

	credential := &scheduler0_client.CredentialCreateRequestBody{
		Archived:  archived,
		CreatedBy: createdBy,
	}

	result, err := cl.CreateCredential(credential)
	if err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runCredentialsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	credentialID := args[0]
	archived, _ := cmd.Flags().GetBool("archived")
	modifiedBy, _ := cmd.Flags().GetString("modified-by")

	update := &scheduler0_client.CredentialUpdateRequestBody{
		Archived:   archived,
		ModifiedBy: modifiedBy,
	}

	result, err := cl.UpdateCredential(credentialID, update)
	if err != nil {
		return fmt.Errorf("failed to update credential: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runCredentialsDelete(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	credentialID := args[0]
	deletedBy, _ := cmd.Flags().GetString("deleted-by")

	deleteReq := &scheduler0_client.CredentialDeleteRequestBody{
		DeletedBy: deletedBy,
	}

	err = cl.DeleteCredential(credentialID, deleteReq)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	fmt.Printf("Credential %s deleted successfully\n", credentialID)
	return nil
}

func runCredentialsArchive(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	credentialID := args[0]
	archivedBy, _ := cmd.Flags().GetString("archived-by")

	err = cl.ArchiveCredential(credentialID, archivedBy)
	if err != nil {
		return fmt.Errorf("failed to archive credential: %w", err)
	}

	fmt.Printf("Credential %s archived successfully\n", credentialID)
	return nil
}

