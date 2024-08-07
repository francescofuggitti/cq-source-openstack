package compute

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/transform"
	"github.com/dihedron/cq-plugin-utils/utils"
	"github.com/dihedron/cq-source-openstack/client"
)

func InstanceMetadata(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_compute_instance_metadata_" + installation,
		Resolver: fetchInstanceMetadata,
		Transform: transformers.TransformWithStruct(
			&utils.Pair[string, string]{},
			transformers.WithNameTransformer(transform.TagNameTransformer), // use cq-name tags to translate name
			transformers.WithTypeTransformer(transform.TagTypeTransformer), // use cq-type tags to translate type
			//transformers.WithSkipFields("OriginalName", "ExtraSpecs"),
		),
	}
}

func fetchInstanceMetadata(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {

	api := meta.(*client.Client)

	instance := parent.Item.(*Instance)

	for k, v := range instance.Metadata {
		pair := &utils.Pair[string, string]{
			Key:   k,
			Value: v,
		}
		api.Logger().Debug().Str("instance id", instance.ID).Msg("streaming instance metadata")
		res <- pair
	}

	return nil
}
