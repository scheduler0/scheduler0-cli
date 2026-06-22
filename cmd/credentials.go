package cmd

import (
	"crypto/rand"
	"encoding/hex"
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

var credentialsGenerateSecretKeyCmd = &cobra.Command{
	Use:   "generate-secret-key",
	Short: "Generate random AES-256 secret keys for use during secret rotation",
	Long: `Generate one or more cryptographically random 64-character hex strings suitable
for use as a Scheduler0 SecretKey (AES-256).

This command runs entirely offline — no API call or authentication is required.
Use it as the first step of the key-rotation workflow:

  1. scheduler0 credentials generate-secret-key   # produce a new key
  2. Update SecretKey in your secrets source (file, env var, SSM, etc.)
  3. scheduler0 credentials rotate-secret         # re-encrypt stored credentials

Use --count to generate multiple keys in a single call, which is convenient
when you want to stage several rotations (A → B → C) without re-running the
command each time.`,
	RunE: runCredentialsGenerateSecretKey,
}

var credentialsRotateSecretCmd = &cobra.Command{
	Use:   "rotate-secret",
	Short: "Re-encrypt all active credentials with a new secret key (self-hosting only)",
	Long: `Re-encrypt all active, non-expired credentials from the old SecretKey to a new one.

This command is for operators who have rotated the server's SecretKey in their
secrets source (file, SSM, AWS Secrets Manager, or env var) and need to update
the encrypted api_secret stored for every active credential.

Steps:
  1. Update SecretKey in your secrets source.
  2. Run this command — it calls POST /api/v1/credentials/rotate-secret on the server.
  3. The server saves the old key as a savepoint, reloads the new key, and re-encrypts
     credentials in batches. The savepoint is deleted on completion.
  4. If interrupted, re-run this command — the server will resume from the savepoint.

Requires Basic Authentication (--username / --password or saved config with auth_type=basic).`,
	RunE: runCredentialsRotateSecret,
}

func init() {
	rootCmd.AddCommand(credentialsCmd)
	credentialsCmd.AddCommand(credentialsListCmd)
	credentialsCmd.AddCommand(credentialsGetCmd)
	credentialsCmd.AddCommand(credentialsCreateCmd)
	credentialsCmd.AddCommand(credentialsUpdateCmd)
	credentialsCmd.AddCommand(credentialsDeleteCmd)
	credentialsCmd.AddCommand(credentialsArchiveCmd)
	credentialsCmd.AddCommand(credentialsRotateSecretCmd)
	credentialsCmd.AddCommand(credentialsGenerateSecretKeyCmd)

	credentialsGenerateSecretKeyCmd.Flags().Int("count", 1, "Number of secret keys to generate")

	credentialsListCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
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
		fmt.Fprintf(cmd.OutOrStderr(), "\nCredential expires at: %s. Store the api_secret returned above — it is shown once and cannot be retrieved again.\n", *result.Data.ExpiresAt)
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

// runCredentialsRotateSecret calls POST /credentials/rotate-secret on the server to
// re-encrypt all active credentials with the new SecretKey. Requires basic auth.
func runCredentialsRotateSecret(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	if cfg.Username == "" || cfg.Password == "" {
		return fmt.Errorf("rotate-secret requires basic authentication: set username and password in your config or use --username and --password flags")
	}
	cfg.AuthType = "basic"

	cl, err := client.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Starting credential secret rotation...")
	fmt.Fprintln(cmd.OutOrStdout(), "Ensure you have already updated SecretKey in your secrets source before proceeding.")

	result, err := cl.RotateCredentialSecret()
	if err != nil {
		return fmt.Errorf("rotation failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Rotation complete: %d credential(s) re-encrypted.\n", result.Data.Rotated)
	return nil
}

func runCredentialsGenerateSecretKey(cmd *cobra.Command, args []string) error {
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
	fmt.Fprintln(cmd.OutOrStdout(), string(output))
	return nil
}

