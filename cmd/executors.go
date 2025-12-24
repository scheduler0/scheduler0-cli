package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
	"github.com/spf13/cobra"
)

var executorsCmd = &cobra.Command{
	Use:   "executors",
	Short: "Manage executors",
	Long:  "Commands for managing Scheduler0 executors",
}

var executorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List executors",
	Long:  "List all executors with pagination",
	RunE:  runExecutorsList,
}

var executorsGetCmd = &cobra.Command{
	Use:   "get [executor-id]",
	Short: "Get executor details",
	Long:  "Get details of a specific executor",
	Args:  cobra.ExactArgs(1),
	RunE:  runExecutorsGet,
}

var executorsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new executor",
	Long:  "Create a new executor (webhook_url, cloud_function, or container)",
	RunE:  runExecutorsCreate,
}

var executorsUpdateCmd = &cobra.Command{
	Use:   "update [executor-id]",
	Short: "Update an executor",
	Long:  "Update an existing executor",
	Args:  cobra.ExactArgs(1),
	RunE:  runExecutorsUpdate,
}

var executorsDeleteCmd = &cobra.Command{
	Use:   "delete [executor-id]",
	Short: "Delete an executor",
	Long:  "Delete an executor",
	Args:  cobra.ExactArgs(1),
	RunE:  runExecutorsDelete,
}

func init() {
	rootCmd.AddCommand(executorsCmd)
	executorsCmd.AddCommand(executorsListCmd)
	executorsCmd.AddCommand(executorsGetCmd)
	executorsCmd.AddCommand(executorsCreateCmd)
	executorsCmd.AddCommand(executorsUpdateCmd)
	executorsCmd.AddCommand(executorsDeleteCmd)

	executorsListCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	executorsListCmd.Flags().Int("offset", 0, "Number of items to skip")
	executorsListCmd.Flags().String("order-by", "date_created", "Field to order by")
	executorsListCmd.Flags().String("order-direction", "desc", "Order direction (asc/desc)")

	executorsCreateCmd.Flags().String("name", "", "Executor name (required)")
	executorsCreateCmd.Flags().String("type", "", "Executor type: webhook_url, cloud_function, or container (required)")
	executorsCreateCmd.Flags().String("region", "", "Cloud region")
	executorsCreateCmd.Flags().String("cloud-provider", "", "Cloud provider")
	executorsCreateCmd.Flags().String("cloud-resource-url", "", "Cloud resource URL")
	executorsCreateCmd.Flags().String("cloud-api-key", "", "Cloud API key")
	executorsCreateCmd.Flags().String("cloud-api-secret", "", "Cloud API secret")
	executorsCreateCmd.Flags().String("webhook-url", "", "Webhook URL")
	executorsCreateCmd.Flags().String("webhook-secret", "", "Webhook secret")
	executorsCreateCmd.Flags().String("webhook-method", "POST", "Webhook HTTP method (GET, POST, PUT, DELETE)")
	executorsCreateCmd.Flags().String("created-by", "", "User who created the executor (required)")
	executorsCreateCmd.MarkFlagRequired("name")
	executorsCreateCmd.MarkFlagRequired("type")
	executorsCreateCmd.MarkFlagRequired("created-by")

	executorsUpdateCmd.Flags().String("name", "", "Executor name")
	executorsUpdateCmd.Flags().String("type", "", "Executor type")
	executorsUpdateCmd.Flags().String("region", "", "Cloud region")
	executorsUpdateCmd.Flags().String("cloud-provider", "", "Cloud provider")
	executorsUpdateCmd.Flags().String("cloud-resource-url", "", "Cloud resource URL")
	executorsUpdateCmd.Flags().String("cloud-api-key", "", "Cloud API key")
	executorsUpdateCmd.Flags().String("cloud-api-secret", "", "Cloud API secret")
	executorsUpdateCmd.Flags().String("webhook-url", "", "Webhook URL")
	executorsUpdateCmd.Flags().String("webhook-secret", "", "Webhook secret")
	executorsUpdateCmd.Flags().String("webhook-method", "", "Webhook HTTP method")
	executorsUpdateCmd.Flags().String("modified-by", "", "User who modified the executor (required)")
	executorsUpdateCmd.MarkFlagRequired("modified-by")

	executorsDeleteCmd.Flags().String("deleted-by", "", "User who deleted the executor (required)")
	executorsDeleteCmd.MarkFlagRequired("deleted-by")
}

func runExecutorsList(cmd *cobra.Command, args []string) error {
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

	params := scheduler0_client.ListExecutorsParams{
		Limit:            limit,
		Offset:           offset,
		OrderBy:          orderBy,
		OrderByDirection: orderDirection,
	}

	result, err := cl.ListExecutors(params)
	if err != nil {
		return fmt.Errorf("failed to list executors: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runExecutorsGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	executorID := args[0]
	result, err := cl.GetExecutor(executorID)
	if err != nil {
		return fmt.Errorf("failed to get executor: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runExecutorsCreate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	executorType, _ := cmd.Flags().GetString("type")
	region, _ := cmd.Flags().GetString("region")
	cloudProvider, _ := cmd.Flags().GetString("cloud-provider")
	cloudResourceURL, _ := cmd.Flags().GetString("cloud-resource-url")
	cloudAPIKey, _ := cmd.Flags().GetString("cloud-api-key")
	cloudAPISecret, _ := cmd.Flags().GetString("cloud-api-secret")
	webhookURL, _ := cmd.Flags().GetString("webhook-url")
	webhookSecret, _ := cmd.Flags().GetString("webhook-secret")
	webhookMethod, _ := cmd.Flags().GetString("webhook-method")
	createdBy, _ := cmd.Flags().GetString("created-by")

	executor := &scheduler0_client.ExecutorRequestBody{
		Name:             name,
		Type:             executorType,
		Region:           region,
		CloudProvider:    cloudProvider,
		CloudResourceURL: cloudResourceURL,
		CloudAPIKey:      cloudAPIKey,
		CloudAPISecret:   cloudAPISecret,
		WebhookURL:       webhookURL,
		WebhookSecret:    webhookSecret,
		WebhookMethod:    webhookMethod,
		CreatedBy:        createdBy,
	}

	result, err := cl.CreateExecutor(executor)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runExecutorsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	executorID := args[0]
	name, _ := cmd.Flags().GetString("name")
	executorType, _ := cmd.Flags().GetString("type")
	region, _ := cmd.Flags().GetString("region")
	cloudProvider, _ := cmd.Flags().GetString("cloud-provider")
	cloudResourceURL, _ := cmd.Flags().GetString("cloud-resource-url")
	cloudAPIKey, _ := cmd.Flags().GetString("cloud-api-key")
	cloudAPISecret, _ := cmd.Flags().GetString("cloud-api-secret")
	webhookURL, _ := cmd.Flags().GetString("webhook-url")
	webhookSecret, _ := cmd.Flags().GetString("webhook-secret")
	webhookMethod, _ := cmd.Flags().GetString("webhook-method")
	modifiedBy, _ := cmd.Flags().GetString("modified-by")

	update := &scheduler0_client.ExecutorUpdateRequestBody{
		Name:             name,
		Type:             executorType,
		Region:           region,
		CloudProvider:    cloudProvider,
		CloudResourceURL: cloudResourceURL,
		CloudAPIKey:      cloudAPIKey,
		CloudAPISecret:   cloudAPISecret,
		WebhookURL:       webhookURL,
		WebhookSecret:    webhookSecret,
		WebhookMethod:    webhookMethod,
		ModifiedBy:       modifiedBy,
	}

	result, err := cl.UpdateExecutor(executorID, update)
	if err != nil {
		return fmt.Errorf("failed to update executor: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runExecutorsDelete(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	executorID := args[0]
	deletedBy, _ := cmd.Flags().GetString("deleted-by")

	deleteReq := &scheduler0_client.ExecutorDeleteRequestBody{
		DeletedBy: deletedBy,
	}

	err = cl.DeleteExecutor(executorID, deleteReq)
	if err != nil {
		return fmt.Errorf("failed to delete executor: %w", err)
	}

	fmt.Printf("Executor %s deleted successfully\n", executorID)
	return nil
}

