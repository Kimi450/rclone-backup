#Requires -RunAsAdministrator

# https://rclone.org/

param (
    [Parameter(mandatory=$false)]
    [switch] $Help,
    [Parameter(mandatory=$false)]
    [switch] $DryRun,
    [Parameter(mandatory=$false)]
    [switch] $ComputeChecksums,
    [Parameter(mandatory=$false)]
    [string] $LogBundleBaseDir = "$PWD",
    [Parameter(mandatory=$true)]
    [string] $RcloneBinary,
    [Parameter(mandatory=$true)]
    [string] $RcloneConfig,
    [Parameter(mandatory=$true)]
    [string] $BackupConfigJson
)

# Get Date Time in a format used for files
function Get-DateTimeForFile {
    return (Get-Date).ToString("yyyyMMdd-HHmmss")
}

# Log to a file
function Write-Log {
    param (
        [Parameter(mandatory=$true)]
        [string] $LogFile,
        [Parameter(mandatory=$true)]
        [string] $Message
    )
    Write-Output "[$(Get-Date)] $Message" | Tee-Object -FilePath $LogFile -Append
}

function Usage {
    param (
        [Parameter(mandatory=$true)]
        [string] $LogFile
    )

    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "Usage: .\backup.ps1 -RcloneConfig <PATH> -BackupConfigJson <PATH> [-RcloneBinary <PATH>] [-DryRun]"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "       -LogBundleBaseDir string    Path to the directory where the log bundles should be generated"
    Write-Log -LogFile LogFile -Message "                                   (default: `$PWD)"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "       -RcloneBinary string        Path to the rclone executable"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "       -RcloneConfig string        Path to the rclone configuration file"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "                                   When using a remote source you are expected to have set it up already"
    Write-Log -LogFile LogFile -Message "                                   using '.\rclone.exe config'. This remote's name is to be used in the"
    Write-Log -LogFile LogFile -Message "                                   the -BackupConfigJson file as the SourceDir. Example: 'google-drive:'"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "                                   Refer to these pages to setup drive and the recommended"
    Write-Log -LogFile LogFile -Message "                                   client-id and client-secret required for this setup:"
    Write-Log -LogFile LogFile -Message "                                       - https://rclone.org/drive/"
    Write-Log -LogFile LogFile -Message "                                       - https://rclone.org/drive/#making-your-own-client-id"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "       -BackupConfigJson string    Path to the script's backup config json file"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "       -DryRun switch              Flag to enable --dry-run mode for the sync operation"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "       -ComputeChecksums switch    Flag to enable --checksum checks for the sync operation"
    Write-Log -LogFile LogFile -Message "   "
    Write-Log -LogFile LogFile -Message "       -Help switch                To see this page"
    Write-Log -LogFile LogFile -Message "   "
}

function Exit-OnError {
    param (
        [Parameter(mandatory=$false)]
        [string] $Message = "Non-Zero exit code, refer to logs",
        [Parameter(mandatory=$true)]
        [string] $LogFile,
        [Parameter(mandatory=$true)]
        [string] $ExitCode
    )

    if ($ExitCode -ne 0) {
        Write-Log -LogFile $LogFile -Message "($ExitCode) $Message"
        exit $ExitCode
    }
}

function Assert-ValidConfig {
    param (
        [Parameter(mandatory=$true)]
        [string] $LogFile
    )

    $Files = @($RcloneBinary, $RcloneConfig, $BackupConfigJson)

    foreach ($File in $Files) {
        if (! (Test-Path -Path $File)) {
            Exit-OnError -LogFile $LogFile -ExitCode 1 -Message "Required file does not exist: [$File]"
            Usage -LogFile $LogFile
        }
    }
}

# Update the rclone binary logging the version before and after the update
function Update-Rclone {
    param (
        [Parameter(mandatory=$true)]
        [string] $LogFile
    )
    Write-Log -LogFile $LogFile -Message "Getting pre-update version"
    .$RcloneBinary --config $RcloneConfig version | Tee-Object -FilePath $LogFile -Append
    Exit-OnError -LogFile $LogFile -ExitCode $LASTEXITCODE
    
    Write-Log -LogFile $LogFile -Message "Running self update"
    .$RcloneBinary --config $RcloneConfig selfupdate | Tee-Object -FilePath $LogFile -Append
    Exit-OnError -LogFile $LogFile -ExitCode $LASTEXITCODE
    
    Write-Log -LogFile $LogFile -Message "Getting post-update version"
    .$RcloneBinary --config $RcloneConfig version | Tee-Object -FilePath $LogFile -Append
    Exit-OnError -LogFile $LogFile -ExitCode $LASTEXITCODE
}

function Get-ReportSummary {
    param (
        [Parameter(mandatory=$true)]
        [string] $LogFile,
        [Parameter(mandatory=$true)]
        [string] $ReportFile
    )
    Write-Log -LogFile $LogFile -Message "Refer to logs for report summary of non-equal items at [$LogFile]"
    Write-Log -LogFile $LogFile -Message "  + is for new files"
    Write-Log -LogFile $LogFile -Message "  - is for deleted files"
    Write-Log -LogFile $LogFile -Message "  ! is for unknown"
    Get-Content $ReportFile | Select-String -Pattern "^[^=]" | ForEach-Object {$_.ToString()} *>> $LogFile
}

function Sync-Source-And-Destination {
    param (
        [Parameter(mandatory=$true)]
        [string] $LogFile,
        [Parameter(mandatory=$true)]
        [hashtable] $ConfigItem,
        [Parameter(mandatory=$true)]
        [bool] $DryRun,
        [Parameter(mandatory=$true)]
        [bool] $ComputeChecksums
    )
    # Config extraction
    $Name = $ConfigItem.Name
    $SourceDir = $ConfigItem.SourceDir
    $DestDir = $ConfigItem.DestDir
    
    Write-Log -LogFile $LogFile -Message "Processing configured backup item '$Name' to sync source [$SourceDir] with destination [$DestDir]"
    
    $ExtraArgs = $null
    if ($DryRun -eq $true) {
        Write-Log -LogFile $LogFile -Message "Dry-run is enabled!"
        $ExtraArgs = "--dry-run"
    }
    if ($ComputeChecksums -eq $true) {
        Write-Log -LogFile $LogFile -Message "Compute checksums is enabled! This will cause a bit of a slowdown"
        $ExtraArgs = "--checksum"
    }

    $DateTime = Get-DateTimeForFile

    # Log files for this set of source-destination combo
    $LogFileSourceFiles = "$LOG_BUNDLE_DIR\$DateTime-$Name-source-files.json"
    $LogFileDestFilesBeforeSync = "$LOG_BUNDLE_DIR\$DateTime-$Name-dest-files-before-sync.json"
    $LogFileDestFilesAfterSync = "$LOG_BUNDLE_DIR\$DateTime-$Name-dest-files-after-sync.json"
    $LogFileSync = "$LOG_BUNDLE_DIR\$DateTime-$Name-sync-logs.json"
    $CombinedReportFileSync = "$LOG_BUNDLE_DIR\$DateTime-$Name-sync-report.txt"
    

    Write-Log -LogFile $LogFile -Message "Getting ls data in json format before sync for source [$SourceDir] and logging in [$LogFileSourceFiles]"
    .$RcloneBinary --config $RcloneConfig lsjson -R $SourceDir *>> $LogFileSourceFiles
    Exit-OnError -LogFile $LogFile -ExitCode $LASTEXITCODE
    
    if (Test-Path -Path $DestDir) {
        Write-Log -LogFile $LogFile -Message "Getting ls data in json format before sync for dest [$DestDir] and logging in [$LogFileDestFilesBeforeSync]"
        .$RcloneBinary --config $RcloneConfig lsjson -R $DestDir *>> $LogFileDestFilesBeforeSync
        Exit-OnError -LogFile $LogFile -ExitCode $LASTEXITCODE
    } else {
        Write-Log -LogFile $LogFile -Message "WARN: Destination dir does not exist: [$DestDir]. Not getting ls data for destination before sync."
    }
    
    Write-Log -LogFile $LogFile -Message "Syncing source [$SourceDir] with destination [$DestDir]"
    Write-Log -LogFile $LogFile -Message "Follow sync logs at: $LogFileSync"
    Write-Log -LogFile $LogFile -Message "Follow report at: $CombinedReportFileSync"#
    .$RcloneBinary --config $RcloneConfig sync $SourceDir $DestDir `
        $ExtraArgs `
        --use-json-log `
        --log-level DEBUG `
        --log-file $LogFileSync `
        --combined $CombinedReportFileSync `
        --check-first `
        --metadata | Tee-Object -FilePath $LogFile -Append
    Exit-OnError -LogFile $LogFile -ExitCode $LASTEXITCODE

    Get-ReportSummary -LogFile $LogFile -ReportFile $CombinedReportFileSync

    Write-Log -LogFile $LogFile -Message "Getting ls data in json format after sync for dest [$DestDir] and logging in [$LogFileDestFilesAfterSync]"
    .$RcloneBinary --config $RcloneConfig lsjson -R $DestDir *>> $LogFileDestFilesAfterSync
    Exit-OnError -LogFile $LogFile -ExitCode $LASTEXITCODE
    
    Write-Log -LogFile $LogFile -Message "Done!"
    
}

## Main script

$LogDateTime = Get-DateTimeForFile
$LOG_BUNDLE_DIR = "$LogBundleBaseDir\$LogDateTime-backup-log-bundle"
$ScriptLogFile = "$LOG_BUNDLE_DIR\$LogDateTime-script-logs.txt"

if ($Help -eq $true) {
    Usage -LogFile $ScriptLogFile
    exit 0
}

# create log bundle dir
New-Item -ItemType Directory -Path $LOG_BUNDLE_DIR -ErrorAction SilentlyContinue

# Validate configuration
Assert-ValidConfig -LogFile $ScriptLogFile

# Update Rclone binary
Update-Rclone -LogFile $ScriptLogFile

# sync every config item
$CONFIG=Get-Content $BackupConfigJson | Out-String | ConvertFrom-Json -AsHashtable
foreach ($Item in $CONFIG.Items) {
    Sync-Source-And-Destination -LogFile $ScriptLogFile -ConfigItem $Item -DryRun $DryRun -ComputeChecksums $ComputeChecksums
}
