# RClone Backup

This is a script I used to take backups of my disk and google drive on Windows. It utilizes [rclone](https://rclone.org/) to do so.

**NOTE:** I do not take any responsibility for this script potentially causing data loss or any unwanted consequences what so ever.

## Setup

You will need to use [PowerShell 7](https://learn.microsoft.com/en-us/powershell/scripting/install/installing-powershell-on-windows?view=powershell-7.4) to use this script.

Sample config files also exist in the `./configs` directory which you can update if needed or just point the script to custom files in other locations.

Refer to the [Usage section](#usage) on more info regarding how to pass in configurations to the script and more usage information.

### Executable

You will be required to [download](https://rclone.org/downloads/) the rclone executable for Windows and point the script to it for it to work.

### Remote Configuration

You are expected to setup an `rclone config` to access `remote` locations like [Google Drive](https://rclone.org/drive/) or otherwise, if applicable. Refer to [rclone documentation](https://rclone.org/docs/) to find how to set this up.

For the case of Google Drive, it is recommended to get your own custom `client_id` and `client_secret` to avoid being rate limited on the default ones. You can find more information on how to do this when going through the `rclone config` workflow which will link you to their [documentation](https://rclone.org/drive/#making-your-own-client-id). A config like below will be created once you follow the process. You are recommended to look at official rclone documentation to see what other configs are needed for your usecases or in the case where this data is stale.

```ini
[<NAME_OF_THE_REMOTE>]
type = drive
client_id = <YOUR_CLIENT_ID>
client_secret = <YOUR_CLIENT_SECRET>
scope = drive.readonly
token = <SOME_TOKEN_INFO>
team_drive = 
```

### Script Configuration

The script's config can be found in `./configs/config.json`. Populate the file with array items representing backups you may wish to take. A sample of this is shown below where a disk directory is being backed up and the entire google-drive (this is the name of the `remote` you select during it's setup) remote location is being backed up.

```json
{
    "Items": [
        {
            "Name": "my-disk-data",             // Any name for you to identify this config item
            "SourceDir": "C:/Windows",          // Source directory on disk
            "DestDir": "B:/My Windows Backup"   // Destination directory on disk
        },
        {
            "Name": "my-google-drive",
            "SourceDir": "google-drive:",   // Name of the remote you chose when configuring it
                                            // You can add absolute paths to access files on this remote
                                            // location as well after the colon ":" (the colon is important)
            "DestDir": "B:/My Google Drive Backup"
        }
    ]
}
```


## Usage

Always run quick dry runs first 

Run a comprehensive check first and then a quick-check for verification since comprehensive checks can be much slower.

Perhaps the best strategy is
- Use [comprehensive checks](#comprehensive-check) for
    - first transfer
    - quarterly checks
- Use [quick checks](#quick-check) for
    - monthly checks
- Use [dry runs](#dry-run) to test things

Flags can be used in conjunction with one another.

### Help

To get usage information

```powershell
./backup.ps1 -Help
```

### Comprehensive Check

Files are checked with also computing checksums, this obviously takes longer because every file will have its checksum computed (on source and destination). It can be 100 times slower than the [quick check](#quick-check).

```powershell
./backup.ps1 -RcloneBinary ./rclone.exe -RcloneConfig ./configs/rclone.conf -BackupConfigJson ./configs/config.json -ComputeChecksums
```

### Quick Check

Files are checked without computing checksums

```powershell
./backup.ps1 -RcloneBinary ./rclone.exe -RcloneConfig ./configs/rclone.conf -BackupConfigJson ./configs/config.json
```

### Dry-run

No mutations take place with this flag.

#### Quick Dry run:

```powershell
./backup.ps1 -RcloneBinary ./rclone.exe -RcloneConfig ./configs/rclone.conf -BackupConfigJson ./configs/config.json -DryRun
```

#### Comprehensive Dry run:

```powershell
./backup.ps1 -RcloneBinary ./rclone.exe -RcloneConfig ./configs/rclone.conf -BackupConfigJson ./configs/config.json -ComputeChecksums -DryRun
```
