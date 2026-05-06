package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
	"github.com/spf13/cobra"
)

// allowedCredentialScopes mirrors the scopes accepted by the Scheduler0 API.
// Kept in sync with constants.CredentialScope* on the server.
var allowedCredentialScopes = map[string]struct{}{
	"read":    {},
	"write":   {},
	"execute": {},
}

// parseCredentialScopes turns a `--scopes` flag value (a comma-separated list)
// into a validated, de-duplicated slice ready to send to the API. It returns a
// helpful error so users immediately know which value tripped validation.
func parseCredentialScopes(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("--scopes cannot be empty; choose at least one of read,write,execute")
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		s := strings.ToLower(strings.TrimSpace(part))
		if s == "" {
			continue
		}
		if _, ok := allowedCredentialScopes[s]; !ok {
			return nil, fmt.Errorf("invalid scope %q (allowed: read, write, execute)", part)
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		scopes = append(scopes, s)
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("--scopes cannot be empty; choose at least one of read,write,execute")
	}
	sort.Strings(scopes)
	return scopes, nil
}

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

var credentialsRotateCmd = &cobra.Command{
	Use:   "rotate [credential-id]",
	Short: "Rotate a credential by creating a new one with the same scopes and archiving the old one",
	Long: `Rotate a credential.

This is a CLI-side helper around the existing Scheduler0 credential API: it loads
the source credential, creates a new credential with the same scopes (or any
override passed via --scopes), then archives the original credential. The new
credential's API key and secret are printed to stdout — store them securely.

Credentials expire 90 days after creation; rotate rather than archive-and-recreate
to preserve the same scope configuration.`,
	Args: cobra.ExactArgs(1),
	RunE: runCredentialsRotate,
}

func init() {
	rootCmd.AddCommand(credentialsCmd)
	credentialsCmd.AddCommand(credentialsListCmd)
	credentialsCmd.AddCommand(credentialsGetCmd)
	credentialsCmd.AddCommand(credentialsCreateCmd)
	credentialsCmd.AddCommand(credentialsUpdateCmd)
	credentialsCmd.AddCommand(credentialsDeleteCmd)
	credentialsCmd.AddCommand(credentialsArchiveCmd)
	credentialsCmd.AddCommand(credentialsRotateCmd)

	credentialsListCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	credentialsListCmd.Flags().Int("offset", 0, "Number of items to skip")
	credentialsListCmd.Flags().String("order-by", "date_created", "Field to order by")
	credentialsListCmd.Flags().String("order-direction", "desc", "Order direction (asc/desc)")
	credentialsListCmd.Flags().String("output", "json", "Output format (json or table)")

	credentialsCreateCmd.Flags().Bool("archived", false, "Whether the credential is archived")
	credentialsCreateCmd.Flags().String("created-by", "", "User who created the credential (required)")
	credentialsCreateCmd.Flags().String("scopes", "read,write,execute", "Comma-separated scopes for the credential (read,write,execute)")
	credentialsCreateCmd.MarkFlagRequired("created-by")

	credentialsUpdateCmd.Flags().Bool("archived", false, "Whether the credential is archived")
	credentialsUpdateCmd.Flags().String("modified-by", "", "User who modified the credential (required)")
	credentialsUpdateCmd.MarkFlagRequired("modified-by")

	credentialsDeleteCmd.Flags().String("deleted-by", "", "User who deleted the credential (required)")
	credentialsDeleteCmd.MarkFlagRequired("deleted-by")

	credentialsArchiveCmd.Flags().String("archived-by", "", "User who archived the credential (required)")
	credentialsArchiveCmd.MarkFlagRequired("archived-by")

	credentialsRotateCmd.Flags().String("created-by", "", "User who created the new credential (required)")
	credentialsRotateCmd.Flags().String("archived-by", "", "User who archived the old credential (defaults to --created-by)")
	credentialsRotateCmd.Flags().String("scopes", "", "Override scopes for the new credential; defaults to the source credential's scopes (csv)")
	credentialsRotateCmd.MarkFlagRequired("created-by")
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
	outputMode, _ := cmd.Flags().GetString("output")

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

	if strings.EqualFold(outputMode, "table") {
		printCredentialsTable(result.Data.Credentials)
		return nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

// printCredentialsTable renders the most relevant fields (including the new
// expires/scopes columns) in a stable column order. We deliberately keep this
// dependency-free to avoid pulling in a tablewriter just for one command.
func printCredentialsTable(credentials []scheduler0_client.Credential) {
	header := []string{"ID", "API KEY", "STATUS", "SCOPES", "CREATED", "EXPIRES"}
	rows := make([][]string, 0, len(credentials))
	for _, cred := range credentials {
		status := "active"
		if cred.Archived {
			status = "archived"
		}
		expires := "never"
		if cred.ExpiresAt != nil && *cred.ExpiresAt != "" {
			expires = *cred.ExpiresAt
		}
		scopes := "-"
		if len(cred.Scopes) > 0 {
			scopes = strings.Join(cred.Scopes, ",")
		}
		rows = append(rows, []string{
			strconv.FormatInt(cred.ID, 10),
			cred.APIKey,
			status,
			scopes,
			cred.DateCreated,
			expires,
		})
	}

	widths := make([]int, len(header))
	for i, h := range header {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	pad := func(values []string) string {
		parts := make([]string, len(values))
		for i, v := range values {
			parts[i] = fmt.Sprintf("%-*s", widths[i], v)
		}
		return strings.Join(parts, "  ")
	}

	fmt.Println(pad(header))
	fmt.Println(strings.Repeat("-", len(pad(header))))
	for _, row := range rows {
		fmt.Println(pad(row))
	}
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
	scopesRaw, _ := cmd.Flags().GetString("scopes")

	scopes, err := parseCredentialScopes(scopesRaw)
	if err != nil {
		return err
	}

	credential := &scheduler0_client.CredentialCreateRequestBody{
		Archived:  archived,
		CreatedBy: createdBy,
		Scopes:    scopes,
	}

	result, err := cl.CreateCredential(credential)
	if err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	if result != nil && result.Data.ExpiresAt != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "\nCredential expires at: %s (rotate before then with `scheduler0 credentials rotate %d`)\n", *result.Data.ExpiresAt, result.Data.ID)
	}
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

// runCredentialsRotate orchestrates a "rotate in place" workflow client-side:
// fetch the source credential to get its scopes, create a fresh credential with
// the same (or overridden) scopes, then archive the original. We deliberately do
// the archive *after* create succeeds so the caller is never left without a
// working credential if the create call fails.
func runCredentialsRotate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	credentialID := args[0]
	createdBy, _ := cmd.Flags().GetString("created-by")
	archivedBy, _ := cmd.Flags().GetString("archived-by")
	if strings.TrimSpace(archivedBy) == "" {
		archivedBy = createdBy
	}
	scopesRaw, _ := cmd.Flags().GetString("scopes")

	source, err := cl.GetCredential(credentialID)
	if err != nil {
		return fmt.Errorf("failed to load source credential %s: %w", credentialID, err)
	}
	if source == nil {
		return fmt.Errorf("source credential %s not found", credentialID)
	}
	if source.Data.Archived {
		return fmt.Errorf("source credential %s is already archived; nothing to rotate", credentialID)
	}

	var scopes []string
	if strings.TrimSpace(scopesRaw) != "" {
		scopes, err = parseCredentialScopes(scopesRaw)
		if err != nil {
			return err
		}
	} else {
		scopes = append(scopes, source.Data.Scopes...)
		if len(scopes) == 0 {
			return fmt.Errorf("source credential %s has no scopes; pass --scopes to set them explicitly", credentialID)
		}
	}

	created, err := cl.CreateCredential(&scheduler0_client.CredentialCreateRequestBody{
		CreatedBy: createdBy,
		Scopes:    scopes,
	})
	if err != nil {
		return fmt.Errorf("failed to create replacement credential: %w", err)
	}

	output, _ := json.MarshalIndent(created, "", "  ")
	fmt.Println(string(output))

	if err := cl.ArchiveCredential(credentialID, archivedBy); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "\nWARNING: created replacement credential %d but failed to archive %s: %v\n", created.Data.ID, credentialID, err)
		fmt.Fprintf(cmd.ErrOrStderr(), "Run `scheduler0 credentials archive %s --archived-by %s` to finish the rotation.\n", credentialID, archivedBy)
		return fmt.Errorf("rotation incomplete: archive of %s failed", credentialID)
	}

	fmt.Fprintf(cmd.OutOrStderr(), "\nRotation complete: store the new API key/secret securely — old credential %s is now archived.\n", credentialID)
	if created.Data.ExpiresAt != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "New credential %d expires at %s.\n", created.Data.ID, *created.Data.ExpiresAt)
	}
	return nil
}

