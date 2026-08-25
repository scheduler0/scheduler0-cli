package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/scheduler0/scheduler0-cli/internal/client"
	scheduler0_client "github.com/scheduler0/scheduler0-go-client/v2"
	"github.com/spf13/cobra"
)

// clusterCmd groups the Raft cluster / operator commands under /api/v1/cluster/*.
// These require an admin-scoped session (or peer/basic auth) and are intended for
// self-hosting operators managing node membership, not application traffic.
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage Raft cluster membership and inspect operator-only state",
	Long:  "Commands for adding/removing/promoting nodes, transferring leadership, and dumping internal scheduler state. Requires an admin-scoped session.",
}

var clusterRemoveSelfCmd = &cobra.Command{
	Use:   "remove-self",
	Short: "Remove this node from the cluster",
	Long:  "Remove the node this command is run against from Raft membership and unregister it from etcd. Intended to be called by a node against itself during shutdown.",
	RunE:  runClusterRemoveSelf,
}

var clusterAddSelfCmd = &cobra.Command{
	Use:   "add-self",
	Short: "Add this node to the cluster",
	Long:  "Ensure the node this command is run against is registered in etcd and part of the Raft cluster. Intended to be called by a node against itself during startup.",
	RunE:  runClusterAddSelf,
}

var clusterForceRebuildCmd = &cobra.Command{
	Use:   "force-rebuild",
	Short: "Force a rebuild of the Raft cluster from a seed node",
	Long:  "Forces a rebuild of the Raft cluster. This should only be run against the seed node.",
	RunE:  runClusterForceRebuild,
}

var clusterResetRaftCmd = &cobra.Command{
	Use:   "reset-raft",
	Short: "Clear local Raft state on this node and exit the process",
	Long:  "Clears local Raft state on the node this command is run against and causes that node's process to exit. Destructive: the node loses all local Raft state and must rejoin the cluster.",
	RunE:  runClusterResetRaft,
}

var clusterRemoveNodeCmd = &cobra.Command{
	Use:   "remove-node",
	Short: "Remove a node from the cluster",
	Long:  "Removes a node from the Raft cluster. Only the leader can perform this operation.",
	RunE:  runClusterRemoveNode,
}

var clusterAddNodeCmd = &cobra.Command{
	Use:   "add-node",
	Short: "Add a node to the cluster",
	Long:  "Adds a node to the Raft cluster. Only the leader can perform this operation.",
	RunE:  runClusterAddNode,
}

var clusterPromoteNodeCmd = &cobra.Command{
	Use:   "promote-node",
	Short: "Promote a non-voter node to a voter",
	Long:  "Promotes a non-voter node to a voter in the Raft cluster. Only the leader can perform this operation.",
	RunE:  runClusterPromoteNode,
}

var clusterDemoteNodeCmd = &cobra.Command{
	Use:   "demote-node",
	Short: "Demote a voter node to a non-voter",
	Long:  "Demotes a voter node to a non-voter in the Raft cluster. Only the leader can perform this operation.",
	RunE:  runClusterDemoteNode,
}

var clusterTransferLeadershipCmd = &cobra.Command{
	Use:   "transfer-leadership",
	Short: "Transfer Raft leadership to another voter node",
	Long:  "Transfers leadership to another voter node in the cluster. Only the leader can perform this operation.",
	RunE:  runClusterTransferLeadership,
}

var clusterListNodesCmd = &cobra.Command{
	Use:   "list-nodes",
	Short: "List every node in the cluster",
	RunE:  runClusterListNodes,
}

var clusterDumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump internal scheduler state for debugging",
	Long:  "Inspect in-memory/persisted scheduler state. These are debug endpoints: the shape of the returned data is an implementation detail, not a stable contract.",
}

var clusterDumpScheduleQueueCmd = &cobra.Command{
	Use:   "schedule-queue",
	Short: "Dump the in-memory schedule queue",
	RunE:  runClusterDumpScheduleQueue,
}

var clusterDumpJobExecutionsCacheCmd = &cobra.Command{
	Use:   "job-executions-cache",
	Short: "Dump the in-memory job executions cache",
	RunE:  runClusterDumpJobExecutionsCache,
}

var clusterDumpJobQueuesCmd = &cobra.Command{
	Use:   "job-queues",
	Short: "Dump every persisted job queue",
	RunE:  runClusterDumpJobQueues,
}

var clusterDumpJobQueueVersionsCmd = &cobra.Command{
	Use:   "job-queue-versions",
	Short: "Dump every persisted job queue version",
	RunE:  runClusterDumpJobQueueVersions,
}

func init() {
	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterRemoveSelfCmd)
	clusterCmd.AddCommand(clusterAddSelfCmd)
	clusterCmd.AddCommand(clusterForceRebuildCmd)
	clusterCmd.AddCommand(clusterResetRaftCmd)
	clusterCmd.AddCommand(clusterRemoveNodeCmd)
	clusterCmd.AddCommand(clusterAddNodeCmd)
	clusterCmd.AddCommand(clusterPromoteNodeCmd)
	clusterCmd.AddCommand(clusterDemoteNodeCmd)
	clusterCmd.AddCommand(clusterTransferLeadershipCmd)
	clusterCmd.AddCommand(clusterListNodesCmd)
	clusterCmd.AddCommand(clusterDumpCmd)
	clusterDumpCmd.AddCommand(clusterDumpScheduleQueueCmd)
	clusterDumpCmd.AddCommand(clusterDumpJobExecutionsCacheCmd)
	clusterDumpCmd.AddCommand(clusterDumpJobQueuesCmd)
	clusterDumpCmd.AddCommand(clusterDumpJobQueueVersionsCmd)

	clusterForceRebuildCmd.Flags().Uint64("seed-node-id", 0, "ID of the seed node to rebuild from (required)")
	_ = clusterForceRebuildCmd.MarkFlagRequired("seed-node-id")

	clusterRemoveNodeCmd.Flags().Uint64("node-id", 0, "ID of the node to remove (required)")
	_ = clusterRemoveNodeCmd.MarkFlagRequired("node-id")

	clusterAddNodeCmd.Flags().Uint64("node-id", 0, "ID of the node to add (required)")
	clusterAddNodeCmd.Flags().String("node-address", "", "Raft/node-to-node address of the node (required)")
	clusterAddNodeCmd.Flags().String("client-address", "", "Client-facing (SDK/API) address of the node (required)")
	_ = clusterAddNodeCmd.MarkFlagRequired("node-id")
	_ = clusterAddNodeCmd.MarkFlagRequired("node-address")
	_ = clusterAddNodeCmd.MarkFlagRequired("client-address")

	clusterPromoteNodeCmd.Flags().Uint64("node-id", 0, "ID of the node to promote to voter (required)")
	_ = clusterPromoteNodeCmd.MarkFlagRequired("node-id")

	clusterDemoteNodeCmd.Flags().Uint64("node-id", 0, "ID of the node to demote to non-voter (required)")
	_ = clusterDemoteNodeCmd.MarkFlagRequired("node-id")
}

func runClusterRemoveSelf(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.RemoveSelfFromCluster()
	if err != nil {
		return fmt.Errorf("failed to remove self from cluster: %w", err)
	}
	return printJSON(result)
}

func runClusterAddSelf(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.AddSelfToCluster()
	if err != nil {
		return fmt.Errorf("failed to add self to cluster: %w", err)
	}
	return printJSON(result)
}

func runClusterForceRebuild(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	seedNodeID, _ := cmd.Flags().GetUint64("seed-node-id")
	result, err := cl.ForceRebuildCluster(seedNodeID)
	if err != nil {
		return fmt.Errorf("failed to force rebuild cluster: %w", err)
	}
	return printJSON(result)
}

func runClusterResetRaft(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.ResetRaftState()
	if err != nil {
		return fmt.Errorf("failed to reset raft state: %w", err)
	}
	return printJSON(result)
}

func runClusterRemoveNode(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	nodeID, _ := cmd.Flags().GetUint64("node-id")
	result, err := cl.RemoveNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to remove node: %w", err)
	}
	return printJSON(result)
}

func runClusterAddNode(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	nodeID, _ := cmd.Flags().GetUint64("node-id")
	nodeAddress, _ := cmd.Flags().GetString("node-address")
	clientAddress, _ := cmd.Flags().GetString("client-address")
	result, err := cl.AddNode(nodeID, nodeAddress, clientAddress)
	if err != nil {
		return fmt.Errorf("failed to add node: %w", err)
	}
	return printJSON(result)
}

func runClusterPromoteNode(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	nodeID, _ := cmd.Flags().GetUint64("node-id")
	result, err := cl.PromoteNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to promote node: %w", err)
	}
	return printJSON(result)
}

func runClusterDemoteNode(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	nodeID, _ := cmd.Flags().GetUint64("node-id")
	result, err := cl.DemoteNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to demote node: %w", err)
	}
	return printJSON(result)
}

func runClusterTransferLeadership(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.TransferLeadership()
	if err != nil {
		return fmt.Errorf("failed to transfer leadership: %w", err)
	}
	return printJSON(result)
}

func runClusterListNodes(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}
	return printJSON(result)
}

func runClusterDumpScheduleQueue(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.DumpScheduleQueue()
	if err != nil {
		return fmt.Errorf("failed to dump schedule queue: %w", err)
	}
	return printJSON(result)
}

func runClusterDumpJobExecutionsCache(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.DumpJobExecutionsCache()
	if err != nil {
		return fmt.Errorf("failed to dump job executions cache: %w", err)
	}
	return printJSON(result)
}

func runClusterDumpJobQueues(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.DumpJobQueues()
	if err != nil {
		return fmt.Errorf("failed to dump job queues: %w", err)
	}
	return printJSON(result)
}

func runClusterDumpJobQueueVersions(cmd *cobra.Command, args []string) error {
	cl, err := newClusterClient()
	if err != nil {
		return err
	}
	result, err := cl.DumpJobQueueVersions()
	if err != nil {
		return fmt.Errorf("failed to dump job queue versions: %w", err)
	}
	return printJSON(result)
}

func newClusterClient() (*scheduler0_client.Client, error) {
	cfg, err := GetClientConfig()
	if err != nil {
		return nil, err
	}
	cl, err := client.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return cl, nil
}

func printJSON(v interface{}) error {
	output, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}
