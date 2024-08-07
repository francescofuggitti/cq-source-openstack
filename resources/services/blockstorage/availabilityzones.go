package blockstorage

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/availabilityzones"
)

func AvailabilityZones(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_blockstorage_availabilityzones_" + installation,
		Resolver: fetchAvailabilityZones,
		Transform: transformers.TransformWithStruct(
			&availabilityzones.AvailabilityZone{},
		),
	}
}

func fetchAvailabilityZones(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
	api := meta.(*client.Client)

	blockstorage, err := api.GetServiceClient(client.BlockStorageV3)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error retrieving client")
		return err
	}

	allPages, err := availabilityzones.List(blockstorage).AllPages()
	if err != nil {
		api.Logger().Err(err).Msg("error listing availabilityzones")
		return err
	}

	allAvailabilityZones, err := availabilityzones.ExtractAvailabilityZones(allPages)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error getting availabilityzones")
		return err
	}

	for _, zoneInfo := range allAvailabilityZones {
		res <- zoneInfo
	}

	return nil
}
