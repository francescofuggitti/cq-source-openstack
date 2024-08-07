package blockstorage

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"
	"github.com/dihedron/cq-plugin-utils/transform"
	"github.com/dihedron/cq-plugin-utils/utils"
	"github.com/dihedron/cq-source-openstack/client"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/backups"
)

func VolumesBackups(installation string) *schema.Table {
	return &schema.Table{
		Name:     "openstack_blockstorage_volumes_backups_" + installation,
		Resolver: fetchVolumesBackups,
		Transform: transformers.TransformWithStruct(
			&Backup{},
			transformers.WithPrimaryKeys("ID"),
			transformers.WithTypeTransformer(transform.TagTypeTransformer), // use cq-type tags to translate type
		),
	}
}

func fetchVolumesBackups(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- interface{}) error {
	api := meta.(*client.Client)

	volume := parent.Item.(*Volume)

	blockstorage, err := api.GetServiceClient(client.BlockStorageV3)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error retrieving client")
		return err
	}

	listOpts := backups.ListOpts{
		VolumeID: volume.ID,
	}

	allPages, err := backups.List(blockstorage, listOpts).AllPages()
	if err != nil {
		api.Logger().Err(err).Msg("error listing backups")
		return err
	}
	allBackups, err := backups.ExtractBackups(allPages)
	if err != nil {
		api.Logger().Error().Err(err).Msg("error extracting backups")
		return err
	}
	api.Logger().Debug().Str("volume id", volume.ID).Msg("streaming volume backups")
	res <- allBackups

	return nil
}

type Backup struct {
	// ID is the Unique identifier of the backup.
	ID string `json:"id"`
	// CreatedAt is the date the backup was created.
	CreatedAt *utils.Time `json:"created_at" cq-type:"timestamp"`
	// UpdatedAt is the date the backup was updated.
	UpdatedAt *utils.Time `json:"updated_at" cq-type:"timestamp"`
	// Name is the display name of the backup.
	Name string `json:"name"`
	// Description is the description of the backup.
	Description string `json:"description"`
	// VolumeID is the ID of the Volume from which this backup was created.
	VolumeID string `json:"volume_id"`
	// SnapshotID is the ID of the snapshot from which this backup was created.
	SnapshotID string `json:"snapshot_id"`
	// Status is the status of the backup.
	Status string `json:"status"`
	// Size is the size of the backup, in GB.
	Size int `json:"size"`
	// Object Count is the number of objects in the backup.
	ObjectCount int `json:"object_count"`
	// Container is the container where the backup is stored.
	Container string `json:"container"`
	// HasDependentBackups is whether there are other backups
	// depending on this backup.
	HasDependentBackups bool `json:"has_dependent_backups"`
	// FailReason has the reason for the backup failure.
	FailReason string `json:"fail_reason"`
	// IsIncremental is whether this is an incremental backup.
	IsIncremental bool `json:"is_incremental"`
	// DataTimestamp is the time when the data on the volume was first saved.
	DataTimestamp *utils.Time `json:"data_timestamp" cq-type:"timestamp"`
	// ProjectID is the ID of the project that owns the backup. This is
	// an admin-only field.
	ProjectID string `json:"os-backup-project-attr:project_id"`
	// Metadata is metadata about the backup.
	// This requires microversion 3.43 or later.
	Metadata *map[string]string `json:"metadata"`
	// AvailabilityZone is the Availability Zone of the backup.
	// This requires microversion 3.51 or later.
	AvailabilityZone *string `json:"availability_zone"`
}
