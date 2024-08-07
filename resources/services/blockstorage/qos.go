package blockstorage

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/qos"
)

func QoS(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_blockstorage_qos_" + installation,
		Resolver: fetchQoS,
		Transform: transformers.TransformWithStruct(
			&qos.QoS{},
			transformers.WithPrimaryKeys("ID"),
		),
	}
}

func fetchQoS(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
	api := meta.(*client.Client)

	blockstorage, err := api.GetServiceClient(client.BlockStorageV3)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error retrieving client")
		return err
	}

	opts := qos.ListOpts{}

	allPages, err := qos.List(blockstorage, opts).AllPages()
	if err != nil {
		api.Logger().Error().Err(err).Str("options", format.ToPrettyJSON(opts)).Msg("error listing qos with options")
		return err
	}
	allQoS, err := qos.ExtractQoS(allPages)
	if err != nil {
		api.Logger().Err(err).Msg("error extracting QoS")
		return err
	}
	for _, qos := range allQoS {
		if ctx.Err() != nil {
			api.Logger().Debug().Msg("context done, exit")
			break
		}
		api.Logger().Debug().Str("data", qos.ID).Msg("streaming qos")
		res <- qos
	}
	return nil
}
