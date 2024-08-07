package blockstorage

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
)

func Snapshots(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_blockstorage_snapshots_" + installation,
		Resolver: fetchSnapshots,
		Transform: transformers.TransformWithStruct(
			&snapshots.Snapshot{},
			transformers.WithPrimaryKeys("ID"),
		),
	}
}

func fetchSnapshots(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
	api := meta.(*client.Client)

	blockstorage, err := api.GetServiceClient(client.BlockStorageV3)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error retrieving client")
		return err
	}

	opts := snapshots.ListOpts{
		AllTenants: true,
	}

	allPages, err := snapshots.List(blockstorage, opts).AllPages()
	if err != nil {
		api.Logger().Error().Err(err).Str("options", format.ToPrettyJSON(opts)).Msg("error listing snapshots with options")
		return err
	}
	allSnapshots, err := snapshots.ExtractSnapshots(allPages)
	if err != nil {
		api.Logger().Err(err).Msg("error extracting snapshots")
		return err
	}
	for _, snapshot := range allSnapshots {
		if ctx.Err() != nil {
			api.Logger().Debug().Msg("context done, exit")
			break
		}
		api.Logger().Debug().Str("data", snapshot.ID).Msg("streaming snapshot")
		res <- snapshot
	}
	return nil
}
