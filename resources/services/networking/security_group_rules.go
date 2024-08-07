package networking

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-plugin-utils/transform"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
)

func SecurityGroupRules(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_networking_security_group_rules_" + installation,
		Resolver: fetchSecurityGroupRules,
		Transform: transformers.TransformWithStruct(
			&rules.SecGroupRule{},
			transformers.WithPrimaryKeys("ID"),
			transformers.WithNameTransformer(transform.TagNameTransformer), // use cq-name tags to translate name
			transformers.WithTypeTransformer(transform.TagTypeTransformer), // use cq-type tags to translate type
			transformers.WithSkipFields("Links"),
		),
		// Columns: []schema.Column{
		// 	{
		// 		Name:        "security_group_rule_ids",
		// 		Type:        schema.TypeStringArray,
		// 		Description: "The collection of IP addresses associated with the port.",
		// 		Resolver: transform.Apply(
		// 			transform.OnObjectField("Rules.ID"),
		// 			transform.NilIfZero(),
		// 		),
		// 	},
		// },
	}
}

func fetchSecurityGroupRules(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {

	api := meta.(*client.Client)

	networking, err := api.GetServiceClient(client.NetworkingV2)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error retrieving client")
		return err
	}

	opts := rules.ListOpts{}

	allPages, err := rules.List(networking, opts).AllPages()
	if err != nil {
		api.Logger().Error().Err(err).Str("options", format.ToPrettyJSON(opts)).Msg("error listing security group rules with options")
		return err
	}
	allSecurityGroupRules, err := rules.ExtractRules(allPages)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error extracting security group rules")
		return err
	}
	api.Logger().Debug().Int("count", len(allSecurityGroupRules)).Msg("security group rules retrieved")

	for _, rule := range allSecurityGroupRules {
		if ctx.Err() != nil {
			api.Logger().Debug().Msg("context done, exit")
			break
		}
		rule := rule
		// api.Logger().Debug().Str("data", format.ToPrettyJSON(rule)).Msg("streaming security group rules")
		api.Logger().Debug().Str("id", rule.ID).Msg("streaming security group rule")
		res <- rule
	}
	return nil
}
