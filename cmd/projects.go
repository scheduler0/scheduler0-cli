package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
	Long:  "Commands for managing Scheduler0 projects",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  "List all projects with pagination",
	RunE:  runProjectsList,
}

var projectsGetCmd = &cobra.Command{
	Use:   "get [project-id]",
	Short: "Get project details",
	Long:  "Get details of a specific project",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectsGet,
}

var projectsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Long:  "Create a new project in Scheduler0",
	RunE:  runProjectsCreate,
}

var projectsUpdateCmd = &cobra.Command{
	Use:   "update [project-id]",
	Short: "Update a project",
	Long:  "Update an existing project",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectsUpdate,
}

var projectsDeleteCmd = &cobra.Command{
	Use:   "delete [project-id]",
	Short: "Delete a project",
	Long:  "Delete a project and all its jobs",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectsDelete,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsGetCmd)
	projectsCmd.AddCommand(projectsCreateCmd)
	projectsCmd.AddCommand(projectsUpdateCmd)
	projectsCmd.AddCommand(projectsDeleteCmd)

	projectsListCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	projectsListCmd.Flags().Int("limit", 10, "Maximum number of items to return")
	projectsListCmd.Flags().Int("offset", 0, "Number of items to skip")
	projectsListCmd.Flags().String("order-by", "date_created", "Field to order by")
	projectsListCmd.Flags().String("order-direction", "desc", "Order direction (asc/desc)")

	projectsGetCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")

	projectsCreateCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	projectsCreateCmd.Flags().String("name", "", "Project name (required)")
	projectsCreateCmd.Flags().String("description", "", "Project description")
	projectsCreateCmd.Flags().String("created-by", "", "User who created the project (required)")
	projectsCreateCmd.MarkFlagRequired("name")
	projectsCreateCmd.MarkFlagRequired("created-by")

	projectsUpdateCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	projectsUpdateCmd.Flags().String("description", "", "Project description")
	projectsUpdateCmd.Flags().String("modified-by", "", "User who modified the project (required)")
	projectsUpdateCmd.MarkFlagRequired("modified-by")

	projectsDeleteCmd.Flags().String("account-id", "", "Account ID (overrides global --account-id for this command)")
	projectsDeleteCmd.Flags().String("deleted-by", "", "User who deleted the project (required)")
	projectsDeleteCmd.MarkFlagRequired("deleted-by")
}

func runProjectsList(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	orderBy, _ := cmd.Flags().GetString("order-by")
	orderDirection, _ := cmd.Flags().GetString("order-direction")

	params := scheduler0_client.ListProjectsParams{
		Limit:            limit,
		Offset:           offset,
		OrderBy:          orderBy,
		OrderByDirection: orderDirection,
	}

	result, err := cl.ListProjects(params)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runProjectsGet(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	projectIDStr := args[0]
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}
	result, err := cl.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runProjectsCreate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	createdBy, _ := cmd.Flags().GetString("created-by")

	project := &scheduler0_client.ProjectRequestBody{
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
	}

	result, err := cl.CreateProject(project)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runProjectsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	projectIDStr := args[0]
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}
	description, _ := cmd.Flags().GetString("description")
	modifiedBy, _ := cmd.Flags().GetString("modified-by")

	update := &scheduler0_client.ProjectUpdateRequestBody{
		Description: description,
		ModifiedBy:  modifiedBy,
	}

	result, err := cl.UpdateProject(projectID, update)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
	return nil
}

func runProjectsDelete(cmd *cobra.Command, args []string) error {
	cfg, err := GetClientConfig()
	if err != nil {
		return err
	}
	applyAccountIDFlag(cmd, cfg)

	cl, err := client.NewClient(cfg)
	if err != nil {
		return err
	}

	projectIDStr := args[0]
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}
	deletedBy, _ := cmd.Flags().GetString("deleted-by")

	deleteReq := &scheduler0_client.ProjectDeleteRequestBody{
		DeletedBy: deletedBy,
	}

	err = cl.DeleteProject(projectID, deleteReq)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	fmt.Printf("Project %d deleted successfully\n", projectID)
	return nil
}

