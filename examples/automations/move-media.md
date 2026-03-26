# move-media (Example Automation)

This is an example automation script that demonstrates how to use HandyMKV's automations feature. It is meant to be adapted to your specific setup.

The script changes the group ownership of encoded files and moves them to a media directory. It is useful when HandyMKV runs under a different user than the media server (e.g., Plex, Jellyfin) and the files need to be accessible to a shared group.

## What it does

For each file in the encoded output directory:

1. Strips the trailing `_t##` identifier from the filename before the extension (e.g. `My_Movie_t00.mkv` → `My_Movie.mkv`). This artifact is commonly added by MakeMKV during disc ripping.
2. Changes the file's group to the specified group name via `chgrp`
3. Sets file permissions to `0774` (owner and group can read/write/execute, others can read)
4. Moves the file to the destination media directory with the cleaned filename

## Parameters

| Name | Source | Description |
|------|--------|-------------|
| `encoded_dir` | `hmkv_output` (key: `hb_output_dir`) | The directory containing encoded files. Populated automatically by HandyMKV after encoding. |
| `media_dir` | `prompt` | The destination directory to move files into. You will be asked for this before the run starts. |
| `group_name` | `static` | The group to assign to each file. |

## Setup

1. Create the automation:

   ```shell
   handymkv automations create
   ```

2. Follow the prompts:

   - **Name**: `move-media`
   - **Command**: `/path/to/examples/automations/move-media.sh`
   - **Param 1**: name `encoded_dir`, source `hmkv_output`, key `hb_output_dir`
   - **Param 2**: name `media_dir`, source `prompt`
   - **Param 3**: name `group_name`, source `static`, value `media` (or your group name)

3. Run HandyMKV with the `-a` flag:

   ```shell
   handymkv -a move-media
   ```

## Requirements

- The user running HandyMKV must have permission to change file group ownership (either as root or as a member of the target group).
- The target group must exist on the system. Create it with `sudo groupadd media` if needed.
