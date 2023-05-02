package resources

import (
	"context"

	"github.com/cloudquery/plugin-sdk/schema"
	"github.com/cloudquery/plugin-sdk/transformers"
	"github.com/dihedron/cq-plugin-utils/format"
	"github.com/dihedron/cq-plugin-utils/transform"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

func Instances() *schema.Table {
	return &schema.Table{
		Name:     "openstack_instances",
		Resolver: fetchInstances,
		Transform: transformers.TransformWithStruct(
			&Instance{},
			transformers.WithPrimaryKeys("ID"),
			transformers.WithNameTransformer(transform.TagNameTransformer), // use cq-name tags to translate name
			transformers.WithTypeTransformer(transform.TagTypeTransformer), // use cq-type tags to translate type
			transformers.WithSkipFields("Links"),
		),
		Columns: []schema.Column{
			{
				Name:        "flavor_name",
				Type:        schema.TypeString,
				Description: "The original name of the flavor used to start the instance.",
				Resolver:    schema.PathResolver("Flavor.OriginalName"),
			},
			{
				Name:        "flavor_vcpus",
				Type:        schema.TypeInt,
				Description: "The number of virtual CPUs in the flavor used to start the instance.",
				Resolver:    schema.PathResolver("Flavor.VCPUs"),
			},
			{
				Name:        "flavor_vgpus",
				Type:        schema.TypeInt,
				Description: "The number of virtual GPUs in the flavor used to start the instance.",
				Resolver: transform.Apply(
					transform.GetObjectField("Flavor.ExtraSpecs.VGPUs"),
					transform.ToInt(),
					transform.OrDefault(0),
				),
			},
			{
				Name:        "flavor_cores",
				Type:        schema.TypeInt,
				Description: "The number of virtual CPU cores in the flavor used to start the instance.",
				Resolver: transform.Apply(
					transform.GetObjectField("Flavor.ExtraSpecs.CPUCores"),
					transform.ToInt(),
					transform.OrDefault(0),
				),
			},
			{
				Name:        "flavor_sockets",
				Type:        schema.TypeInt,
				Description: "The number of CPU sockets in the flavor used to start the instance.",
				Resolver: transform.Apply(
					transform.GetObjectField("Flavor.ExtraSpecs.CPUSockets"),
					transform.ToInt(),
					transform.OrDefault(0),
				),
			},
			{
				Name:        "flavor_ram",
				Type:        schema.TypeInt,
				Description: "The amount of RAM in the flavor used to start the instance.",
				Resolver:    schema.PathResolver("Flavor.RAM"),
			},
			{
				Name:        "flavor_disk",
				Type:        schema.TypeInt,
				Description: "The size of the disk in the flavor used to start the instance.",
				Resolver:    schema.PathResolver("Flavor.Disk"),
			},
			{
				Name:        "flavor_swap",
				Type:        schema.TypeInt,
				Description: "The size of the swap disk in the flavor used to start the instance.",
				Resolver:    schema.PathResolver("Flavor.Swap"),
			},
			{
				Name:        "flavor_ephemeral",
				Type:        schema.TypeInt,
				Description: "The size of the ephemeral disk in the flavor used to start the instance.",
				Resolver:    schema.PathResolver("Flavor.Ephemeral"),
			},
			{
				Name:        "flavor_rng_allowed",
				Type:        schema.TypeBool,
				Description: "Whether the RNG is allowed on the flavor used to start the instance.",
				Resolver:    schema.PathResolver("Flavor.ExtraSpecs.RNGAllowed"),
			},
			{
				Name:        "flavor_watchdog_action",
				Type:        schema.TypeString,
				Description: "The action to take when the Nova watchdog detects the instance is not responding.",
				Resolver:    schema.PathResolver("Flavor.ExtraSpecs.WatchdogAction"),
			},
			{
				Name:        "image",
				Type:        schema.TypeString,
				Description: "The Glance image used to start the instance.",
				Resolver: transform.Apply(
					transform.GetObjectField("Image"),
					transform.GetMapEntry[string, any]("id"),
					transform.TrimString(),
					transform.NilIfZero(),
				),
			},
			{
				Name:        "attached_volume_ids",
				Type:        schema.TypeStringArray,
				Description: "The volumes attached to the instance.",
				Resolver: transform.Apply(
					transform.GetObjectField("AttachedVolumes"),
					func(ctx context.Context, _ schema.ClientMeta, _ *schema.Resource, _ schema.Column, v any) (any, error) {
						if v != nil {
							if volumes, ok := v.([]servers.AttachedVolume); ok {
								result := []string{}
								for _, volume := range volumes {
									result = append(result, volume.ID)
								}
								return result, nil
							}
						}
						return nil, nil
					}),
			},
			{
				Name:        "power_state_name",
				Type:        schema.TypeString,
				Description: "The instance power state as a string.",
				Resolver: transform.Apply(
					transform.GetObjectField("PowerState"),
					transform.RemapValue(map[int]string{
						0: "NOSTATE",
						1: "RUNNING",
						3: "PAUSED",
						4: "SHUTDOWN",
						6: "CRASHED",
						7: "SUSPENDED",
					}),
				),
			},
			// {
			// 	Name:        "user_data",
			// 	Type:        schema.TypeString,
			// 	Description: "The user data associated with the VM instance.",
			// 	Resolver: transform.Apply(
			// 		transform.GetObjectField("UserData"),
			// 		transform.DecodeBase64(),
			// 	),
			// },
		},
	}
}

func fetchInstances(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {

	api := meta.(*client.Client)

	compute, err := api.GetServiceClient(client.ComputeV2)
	if err != nil {
		api.Logger.Error().Err(err).Msg("error retrieving client")
		return err
	}

	opts := servers.ListOpts{
		AllTenants: true,
	}

	allPages, err := servers.List(compute, opts).AllPages()
	if err != nil {
		api.Logger.Error().Err(err).Str("options", format.ToPrettyJSON(opts)).Msg("error listing instances with options")
		return err
	}
	allInstances := []*Instance{}
	err = servers.ExtractServersInto(allPages, &allInstances)
	if err != nil {
		api.Logger.Error().Err(err).Msg("error extracting instances")
		return err
	}
	api.Logger.Debug().Int("count", len(allInstances)).Msg("instances retrieved")

	for _, instance := range allInstances {
		if ctx.Err() != nil {
			api.Logger.Debug().Msg("context done, exit")
			break
		}
		instance := instance
		api.Logger.Debug().Str("data", format.ToPrettyJSON(instance)).Msg("streaming instance")
		res <- instance
	}
	return nil
}

// Instance is an internal type used to unmarshal more data from the API
// response than would usually be possible through the ordinary gophercloud
// struct. OpenStack API microversions enable more response data that is not
// taken into account by the gophercloud library, which unmarshals only what
// is available at the base level for each API version, for backward compatibility.
// This is also why there is an ExtractInto function that allows you to pass in
// an arbitrary struct to marshal the response data into.
type Instance struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	CreatedAt    Time   `json:"created" cq-name:"created_at" cq-type:"timestamp"`
	LaunchedAt   Time   `json:"OS-SRV-USG:launched_at" cq-name:"launched_at" cq-type:"timestamp"`
	UpdatedAt    Time   `json:"updated" cq-name:"updated_at" cq-type:"timestamp"`
	TerminatedAt Time   `json:"OS-SRV-USG:terminated_at" cq-name:"terminated_at" cq-type:"timestamp"`
	HostID       string `json:"hostid"`
	Status       string `json:"status"`
	Progress     int    `json:"progress"`
	AccessIPv4   string `json:"accessIPv4"`
	AccessIPv6   string `json:"accessIPv6"`
	Image        any    `json:"image"`
	Flavor       struct {
		Disk       int `json:"disk"`
		Ephemeral  int `json:"ephemeral"`
		ExtraSpecs struct {
			CPUCores        string `json:"hw:cpu_cores"`
			CPUSockets      string `json:"hw:cpu_sockets"`
			RNGAllowed      string `json:"hw_rng:allowed"`
			WatchdogAction  string `json:"hw:watchdog_action"`
			VGPUs           string `json:"resources:VGPU"`
			TraitCustomVGPU string `json:"trait:CUSTOM_VGPU"`
		} `json:"extra_specs"`
		OriginalName string `json:"original_name"`
		RAM          int    `json:"ram"`
		Swap         int    `json:"swap"`
		VCPUs        int    `json:"vcpus"`
	} `json:"flavor"`
	Addresses map[string][]struct {
		MACAddress string `json:"OS-EXT-IPS-MAC:mac_addr"`
		IPType     string `json:"OS-EXT-IPS:type"`
		IPAddress  string `json:"addr"`
		IPVersion  int    `json:"version"`
	} `json:"addresses"`
	Metadata map[string]string `json:"metadata"`
	Links    []struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	} `json:"links"`
	KeyName        string `json:"key_name"`
	AdminPass      string `json:"adminPass"`
	SecurityGroups []struct {
		Name string `json:"name"`
	} `json:"security_groups"`
	AttachedVolumes []servers.AttachedVolume `json:"os-extended-volumes:volumes_attached" cq-name:"attached_volumes"`
	// NO!!! Fault              servers.Fault            `json:"fault"`
	Tags               *[]string `json:"tags"`
	ServerGroups       *[]string `json:"server_groups"`
	DiskConfig         string    `json:"OS-DCF:diskConfig" cq-name:"disk_config"`
	AvailabilityZone   string    `json:"OS-EXT-AZ:availability_zone" cq-name:"availability_zone"`
	Host               string    `json:"OS-EXT-SRV-ATTR:host" cq-name:"host"`
	HostName           string    `json:"OS-EXT-SRV-ATTR:hostname" cq-name:"hostname"`
	HypervisorHostname string    `json:"OS-EXT-SRV-ATTR:hypervisor_hostname" cq-name:"hypervisor_hostname"`
	InstanceName       string    `json:"OS-EXT-SRV-ATTR:instance_name" cq-name:"instance_name"`
	KernelID           string    `json:"OS-EXT-SRV-ATTR:kernel_id" cq-name:"kernel_id"`
	LaunchIndex        int       `json:"OS-EXT-SRV-ATTR:launch_index" cq-name:"launch_index"`
	RAMDiskID          string    `json:"OS-EXT-SRV-ATTR:ramdisk_id" cq-name:"ramdisk_id"`
	ReservationID      string    `json:"OS-EXT-SRV-ATTR:reservation_id" cq-name:"reservation_id"`
	RootDeviceName     string    `json:"OS-EXT-SRV-ATTR:root_device_name" cq-name:"root_device_name"`
	UserData           string    `json:"OS-EXT-SRV-ATTR:user_data" cq-name:"user_data"`
	PowerState         int       `json:"OS-EXT-STS:power_state" cq-name:"power_state_id"`
	VMState            string    `json:"OS-EXT-STS:vm_state" cq-name:"vm_state"`
	ConfigDrive        string    `json:"config_drive"`
	Description        string    `json:"description"`
	//	NO!!! TaskState          interface{}              `json:"OS-EXT-STS:task_state"`
}
