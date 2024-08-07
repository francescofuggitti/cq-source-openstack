package blockstorage

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/transform"
	"github.com/dihedron/cq-source-openstack/client"
)

func AttachmentHosts(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_blockstorage_attachment_hosts_" + installation,
		Resolver: fetchAttachmentHosts,
		Transform: transformers.TransformWithStruct(
			&Host{},
			transformers.WithNameTransformer(transform.TagNameTransformer), // use cq-name tags to translate name
			transformers.WithTypeTransformer(transform.TagTypeTransformer), // use cq-type tags to translate type
			//transformers.WithSkipFields("OriginalName", "ExtraSpecs"),
		),
	}
}

func fetchAttachmentHosts(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {

	api := meta.(*client.Client)

	attachment := parent.Item.(*Attachment)

	if len(attachment.ConnectionInfo.Hosts) == len(attachment.ConnectionInfo.Ports) {
		for i := 0; i < len(attachment.ConnectionInfo.Hosts); i++ {
			host := &Host{
				Host: attachment.ConnectionInfo.Hosts[i],
				Port: attachment.ConnectionInfo.Ports[i],
			}
			api.Logger().Debug().Str("attachment id", attachment.ID).Msg("streaming attachment host")
			res <- host
		}
	}
	return nil
}

type Host struct {
	Host string `cq-name:"host"`
	Port string `cq-name:"port" cq-type:"int"`
}
