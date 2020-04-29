# Raft Snapshot Agent

Raft Snapshot Agent is a Go binary that is meant to run alongside every member of a Vault cluster and will take periodic snapshots of the Raft database and write it to the desired location.  It's configuration is meant to somewhat parallel that of the [Consul Snapshot Agent](https://www.consul.io/docs/commands/snapshot/agent.html) so many of the same configuration properties you see there will be present here.

## "High Availability" explained
It works in an "HA" way as follows:
1) Each running daemon checks the IP address of the machine its running on.
2) If this IP address matches that of the leader node, it will be responsible for performing snapshotting.
3) The other binaries simply continue checking, on each snapshot interval, to see if they have become the leader.

In this way, the daemon will always run on the leader Raft node.

Another way to do this, which would allow us to run the snapshot agent anywhere, is to simply have the daemons form their own Raft cluster, but this approach seemed much more cumbersome.

## Configuration

`addr` The address of the Vault cluster.  This is used to check the Vault cluster leader IP, as well as generate snapshots.

`retain` The number of backups to retain.  Currently implemented for all storage types, but only tested on AWS and Local storage.

`timeout` How often to run the snapshot agent.  Examples: `30s`, `1h`.  See https://golang.org/pkg/time/#ParseDuration for a full list of valid time units.

`token` Specify the token used to call the Vault API.  This can also be specified via the env variable `SNAPSHOT_TOKEN`.

### Storage options

Note that if you specify more than one storage option, *all* options will be written to.  For example, specifying `local_storage` and `aws_storage` will write to both locations.

`local_storage` - Object for writing to a file on disk.

`aws_storage` - Object for writing to an S3 bucket.

`google_storage` - Object for writing to GCS.

`azure_storage` - Object for writing to Azure.

#### Local Storage

`path` - Fully qualified path, not including file name, for where the snapshot should be written.  i.e. /etc/raft/snapshots

#### AWS Storage

`access_key_id` - Recommended to use the standard `AWS_ACCESS_KEY_ID` env var, but its possible to specify this in the config

`secret_access_key` - Recommended to use the standard `SECRET_ACCESS_KEY` env var, but its possible to specify this in the config

`s3_region` - S3 region as is required for programmatic interaction with AWS

`s3_bucket` - bucket to store snapshots in (required for AWS writes to work)

`s3_key_prefix` - Prefix to store s3 snapshots in.  Defaults to `raft_snapshots`

`s3_server_side_encryption` -  Encryption is **off** by default.  Set to true to turn on AWS' AES256 encryption.  Support for AWS KMS keys is not currently supported.

`s3_static_snapshot_name` - Use a single, static key for s3 snapshots as opposed to autogenerated timestamped-based ones.  Unless S3 versioning is used, this means there will only ever be a single point-in-time snapshot stored in S3.

#### Google Storage

`bucket` - The Google Storage Bucket to write to.

#### Azure Storage

`account_name` - The account name of the storage account

`account_key` - The account key of the storage account

`container_name` The name of the blob container to write to