# bdpatch

A command-line tool that patches Blu-ray backups for playback on Oppo Blu-ray players.

## Why

Oppo players can be picky about the metadata in Blu-ray backups. `bdpatch` fixes the index file so the player recognizes and plays the disc correctly. There was an application called `bdmv_modify` written by an unknown (to me, at least) author that does this patching. Unforutnately, there was no source code and the application is Windows only, so I figured out what it does and re-produced that here in this cross-platform Go application. Additionally, my version restructures the backup folder in to a `AVCHD` directory (that Oppo players require). It will not do this restructuring if it is already in `AVCHD` or if the `--no-restructure` flag is given.   

## Usage

Point bdpatch at the root folder of your Blu-ray backup (the one containing the `BDMV/` directory):

```sh
bdpatch /path/to/MyMovie
```

### Options

| Flag               | Description                                                    |
|--------------------|----------------------------------------------------------------|
| `--force-2d`       | Force a 3D disc to play as 2D                                  |
| `--no-restructure` | Skip reorganizing the folder layout into an `AVCHD/` directory |

### Examples

```sh
# Patch a disc
bdpatch /mnt/bluray/MyMovie

# Patch and convert 3D to 2D
bdpatch --force-2d /mnt/bluray/MyMovie

# Patch without reorganizing the folder structure
bdpatch --no-restructure /mnt/bluray/MyMovie
```

## What it changes

1. **Index file** — Adds extension data that Oppo players require and add the extension data pointer. Already-patched discs are detected and skipped.
2. **3D flag** (with `--force-2d`) — Clears the 3D flag so the disc plays in 2D.
3. **MovieObject timestamp** — Syncs the modification time of `MovieObject.bdmv` to match the patched index, which some players check.
4. **Folder layout** (unless `--no-restructure`) — Moves `BDMV/`, `STREAM/`, and other top-level folders into an `AVCHD/` parent directory.
