package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

// allowedCredentialScopes mirrors the scopes accepted by the Scheduler0 API,
// kept in sync with constants.CredentialScope* on the server. The "admin" scope
// is accepted here, but the API only lets it be granted by an operator or an
// existing admin credential.
var allowedCredentialScopes = map[string]struct{}{
	"read":    {},
	"write":   {},
	"execute": {},
	"admin":   {},
}

// parseCredentialScopes turns a `--scopes` flag value (a comma-separated list)
// into a validated, de-duplicated slice ready to send to the API. It returns a
// helpful error so users immediately know which value tripped validation.
func parseCredentialScopes(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("--scopes cannot be empty; choose at least one of read,write,execute,admin")
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
			return nil, fmt.Errorf("invalid scope %q (allowed: read, write, execute, admin)", part)
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		scopes = append(scopes, s)
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("--scopes cannot be empty; choose at least one of read,write,execute,admin")
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

func init() {
	rootCmd.AddCommand(credentialsCmd)
	credentialsCmd.AddCommand(credentialsListCmd)
	credentialsCmd.AddCommand(credentialsGetCmd)
	credentialsCmd.AddCommand(credentialsCreateCmd)
	credentialsCmd.AddCommand(credentialsUpdateCmd)
	credentialsCmd.AddCommand(credentialsDeleteCmd)
	credentialsCmd.AddCommand(credentialsArchiveCmd)

	credentialsListCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	credentialsListCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	credentialsListCmd.Flags().Int("offset", 0, "Number of items to skip")
	credentialsListCmd.Flags().String("order-by", "date_created", "Field to order by")
	credentialsListCmd.Flags().String("order-direction", "desc", "Order direction (asc/desc)")
	credentialsListCmd.Flags().String("output", "json", "Output format (json or table)")

	credentialsCreateCmd.Flags().Bool("archived", false, "Whether the credential is archived")
	credentialsCreateCmd.Flags().String("scopes", "read,write,execute", "Comma-separated scopes for the credential (read,write,execute)")

	credentialsUpdateCmd.Flags().Bool("archived", false, "Whether the credential is archived")
}

func runCredentialsList(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	applyAccountIDFlag(cmd, cfg)

	if cfg.AccountID == "" {
		return fmt.Errorf("account id is required to list credentials; provide --account-id or set account_id in your config")
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

	// Parse account ID to int64 for the explicit per-request override path in
	// ListCredentials, so it wins over any c.AccountID set from config.
	var accountIDInt int64
	if cfg.AccountID != "" {
		accountIDInt, _ = strconv.ParseInt(cfg.AccountID, 10, 64)
	}

	params := scheduler0_client.ListCredentialsParams{
		AccountID:        accountIDInt,
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
	createdBy, err := actor(cfg)
	if err != nil {
		return err
	}
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
		_, _ = fmt.Fprintf(cmd.OutOrStderr(), "\nCredential expires at: %s. Store the api_secret returned above — it is shown once and cannot be retrieved again.\n", *result.Data.ExpiresAt)
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
	modifiedBy, err := actor(cfg)
	if err != nil {
		return err
	}

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
	deletedBy, err := actor(cfg)
	if err != nil {
		return err
	}

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
	archivedBy, err := actor(cfg)
	if err != nil {
		return err
	}

	err = cl.ArchiveCredential(credentialID, archivedBy)
	if err != nil {
		return fmt.Errorf("failed to archive credential: %w", err)
	}

	fmt.Printf("Credential %s archived successfully\n", credentialID)
	return nil
}
