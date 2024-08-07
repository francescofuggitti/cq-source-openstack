package networking

import (
	"context"

	"github.com/dihedron/cq-plugin-utils/utils"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/transform"
	"github.com/dihedron/cq-source-openstack/client"
)

func NetworkTags(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_networking_network_tags_" + installation,
		Resolver: fetchNetworkTags,
		Transform: transformers.TransformWithStruct(
			&utils.Tag{},
			transformers.WithNameTransformer(transform.TagNameTransformer), // use cq-name tags to translate name
			transformers.WithTypeTransformer(transform.TagTypeTransformer), // use cq-type tags to translate type
		),
	}
}

func fetchNetworkTags(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
	api := meta.(*client.Client)
	network := parent.Item.(*Network)
	if network.Tags != nil {
		for _, v := range network.Tags {
			tag := &utils.Tag{Value: v}
			api.Logger().Debug().Str("network id", network.ID).Msg("streaming network tag")
			res <- tag
		}
	}
	return nil
}
