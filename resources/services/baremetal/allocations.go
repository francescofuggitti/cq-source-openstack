package baremetal

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/allocations"
)

func Allocations(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_baremetal_allocations_" + installation,
		Resolver: fetchAllocation,
		Transform: transformers.TransformWithStruct(
			&allocations.Allocation{},
			transformers.WithSkipFields("Links"),
		),
	}
}

func fetchAllocation(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
	api := meta.(*client.Client)

	baremetal, err := api.GetServiceClient(client.BareMetalV1)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error retrieving client")
		return err
	}
	opts := allocations.ListOpts{}

	allPages, err := allocations.List(baremetal, opts).AllPages()
	if err != nil {
		api.Logger().Error().Err(err).Str("opts", format.ToPrettyJSON(opts)).Msg("error listing allocations with options")
		return err
	}

	allAllocations, err := allocations.ExtractAllocations(allPages)
	if err != nil {
		api.Logger().Err(err).Msg("error extracting allocations")
		return err
	}
	for _, allocation := range allAllocations {
		if ctx.Err() != nil {
			api.Logger().Debug().Msg("context done, exit")
			break
		}
		api.Logger().Debug().Str("name", allocation.Name).Msg("streaming allocation")
		res <- allocation
	}
	return nil
}
