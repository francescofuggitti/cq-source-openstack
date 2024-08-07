package networking

import (
	"context"

	"github.com/apache/arrow/go/v15/arrow"
	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-plugin-utils/transform"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

func Ports(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_networking_ports_" + installation,
		Resolver: fetchPorts,
		Transform: transformers.TransformWithStruct(
			&ports.Port{},
			transformers.WithPrimaryKeys("ID"),
			transformers.WithNameTransformer(transform.TagNameTransformer), // use cq-name tags to translate name
			transformers.WithTypeTransformer(transform.TagTypeTransformer), // use cq-type tags to translate type
			transformers.WithSkipFields("Links"),
		),
		Columns: []schema.Column{
			{
				Name:        "ip_addresses",
				Type:        arrow.ListOf(arrow.BinaryTypes.String),
				Description: "The collection of IP addresses associated with the port.",
				Resolver: transform.Apply(
					transform.OnObjectField("FixedIPs.IPAddress"),
					transform.NilIfZero(),
				),
			},
			{
				Name:        "ip_address",
				Type:        arrow.BinaryTypes.String,
				Description: "The first IP address associated with the port.",
				Resolver: transform.Apply(
					transform.OnObjectField("FixedIPs.IPAddress"),
					transform.GetElementAt(0),
					transform.NilIfZero(),
				),
			},
		},
	}
}

func fetchPorts(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {

	api := meta.(*client.Client)

	networking, err := api.GetServiceClient(client.NetworkingV2)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error retrieving client")
		return err
	}

	opts := ports.ListOpts{}

	allPages, err := ports.List(networking, opts).AllPages()
	if err != nil {
		api.Logger().Error().Err(err).Str("options", format.ToPrettyJSON(opts)).Msg("error listing ports with options")
		return err
	}
	allPorts, err := ports.ExtractPorts(allPages)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error extracting ports")
		return err
	}
	api.Logger().Debug().Int("count", len(allPorts)).Msg("ports retrieved")

	for _, port := range allPorts {
		if ctx.Err() != nil {
			api.Logger().Debug().Msg("context done, exit")
			break
		}
		api.Logger().Debug().Str("id", port.ID).Msg("streaming port")
		res <- port
	}
	return nil
}
