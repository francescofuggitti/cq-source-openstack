package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudquery/plugin-sdk/v4/message"
	"github.com/cloudquery/plugin-sdk/v4/plugin"
	"github.com/cloudquery/plugin-sdk/v4/scheduler"
	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/pattern_matcher"
	"github.com/dihedron/cq-source-openstack/client"

	"github.com/dihedron/cq-source-openstack/resources/services/baremetal"
	"github.com/dihedron/cq-source-openstack/resources/services/blockstorage"
	"github.com/dihedron/cq-source-openstack/resources/services/compute"
	"github.com/dihedron/cq-source-openstack/resources/services/identity"
	"github.com/dihedron/cq-source-openstack/resources/services/image"
	"github.com/dihedron/cq-source-openstack/resources/services/networking"
	"github.com/rs/zerolog"
)

type Client struct {
	logger     zerolog.Logger
	config     client.Spec
	tables     schema.Tables
	syncClient *client.Client
	scheduler  *scheduler.Scheduler

	plugin.UnimplementedDestination
}

func Configure(ctx context.Context, logger zerolog.Logger, spec []byte, opts plugin.NewClientOptions) (plugin.Client, error) {
	config := &client.Spec{}
	// this prevents go doc to fail when spec is empty
	if len(spec) != 0 {
		if err := json.Unmarshal(spec, config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	tables := getTables(config)

	if opts.NoConnection {
		return &Client{
			logger: logger,
			tables: tables,
		}, nil
	}

	syncClient, err := client.New(ctx, logger, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Client{
		logger:     logger,
		config:     *config,
		tables:     tables,
		syncClient: syncClient,
		scheduler:  scheduler.NewScheduler(scheduler.WithLogger(logger)),
	}, nil
}

func (c *Client) Sync(ctx context.Context, options plugin.SyncOptions, res chan<- message.SyncMessage) error {
	tt, err := c.tables.FilterDfs(options.Tables, options.SkipTables, options.SkipDependentTables)
	if err != nil {
		return err
	}

	return c.scheduler.Sync(ctx, c.syncClient, tt, res, scheduler.WithSyncDeterministicCQID(options.DeterministicCQID))
}

func (c *Client) Tables(_ context.Context, options plugin.TableOptions) (schema.Tables, error) {
	tt, err := c.tables.FilterDfs(options.Tables, options.SkipTables, options.SkipDependentTables)
	if err != nil {
		return nil, err
	}

	return tt, nil
}

func (*Client) Close(_ context.Context) error {
	// TODO: Add your client cleanup here
	return nil
}

func getTables(spec *client.Spec) schema.Tables {
	os_installation := spec.Installation
	available_tables := schema.Tables{
		baremetal.Allocations(*os_installation),
		baremetal.Drivers(*os_installation),
		baremetal.Nodes(*os_installation),
		baremetal.Ports(*os_installation),
		blockstorage.Attachments(*os_installation),
		blockstorage.AvailabilityZones(*os_installation),
		blockstorage.Limits(*os_installation),
		blockstorage.QoS(*os_installation),
		blockstorage.QuotaSets(*os_installation),
		blockstorage.QuotaSetsUsage(*os_installation),
		blockstorage.Services(*os_installation),
		blockstorage.Snapshots(*os_installation),
		blockstorage.Volumes(*os_installation),
		compute.Aggregates(*os_installation),
		compute.Flavors(*os_installation),
		compute.Hypervisors(*os_installation),
		compute.Instances(*os_installation),
		compute.ServerUsage(*os_installation),
		identity.Domains(*os_installation),
		identity.Projects(*os_installation),
		identity.Regions(*os_installation),
		identity.RegisteredLimits(*os_installation),
		identity.Roles(*os_installation),
		identity.Users(*os_installation),
		identity.Services(*os_installation),
		image.Images(*os_installation),
		networking.Networks(*os_installation),
		networking.Ports(*os_installation),
		networking.SecurityGroups(*os_installation),
		networking.SecurityGroupRules(*os_installation),
	}

	// must compile these patterns to be included
	includesFromSpec := spec.IncludedTables
	// must compile these patterns to be excluded
	excludesFromSpec := spec.ExcludedTables

	pm := pattern_matcher.New(
		pattern_matcher.WithReplaceIncludes(includesFromSpec),
		pattern_matcher.WithReplaceExcludes(excludesFromSpec),
	)

	tables := schema.Tables{}
	for _, t := range available_tables {
		if pm.Match(t.Name) {
			tables = append(tables, t)
		}
	}

	if err := transformers.TransformTables(tables); err != nil {
		panic(err)
	}
	for _, t := range tables {
		schema.AddCqIDs(t)
	}
	return tables
}
