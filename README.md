# wantarr

A simple CLI tool that can be used to search for wanted/cutoff unmet media in:

- Sonarr
- Radarr
- Lidarr
- Readarr

Once an item has been searched, it will not be searched again until the retry days age has been reached.

## Configuration
Name `config.yaml` and place in same directory as wantarr executable.
```yaml
pvr:
  sonarr:
    type: sonarr_v3
    url: https://sonarr.domain.com
    api_key: YOUR_API_KEY
    retry_days_age:
      missing: 90
      cutoff: 90
  radarr:
    type: radarr_v2
    url: http://192.168.1.7:7878
    api_key: YOUR_API_KEY
    retry_days_age:
      missing: 90
      cutoff: 90
  radarr4k:
    type: radarr_v3
    url: https://radarr.domain.com
    api_key: YOUR_API_KEY
    retry_days_age:
      missing: 90
      cutoff: 90
  lidarr:
    type: lidarr_v2
    url: https://lidarr.domain.com
    api_key: YOUR_API_KEY
    retry_days_age:
      missing: 90
      cutoff: 90
```


## Examples

- `wantarr missing radarr -v -m 20`
- `wantarr cutoff radarr4k -v -m 20`

## Help
```
Available Commands:
  cutoff      Search for cutoff unmet media files
  missing     Search for missing media files
  help        Help about any command

Flags:
  -h, --help              help for missing
  -m, --max-search int    Exit when this many items have been searched.
  -q, --queue-size int    Exit when queue size reached.
  -r, --refresh-cache     Refresh the locally stored cache.
  -s, --search-size int   How many items to search at once. (default 10)

Global Flags:
  -c, --config string       Config file (default "config.yaml")
      --config-dir string   Config folder (default "C:\\Users\\migue\\AppData\\Local\\Temp\\go-build1345219734\\b001\\exe")
  -d, --database string     Database file (default "vault.db")
  -l, --log string          Log file (default "activity.log")
  -v, --verbose count       Verbose level
```

## Versions

### Supported Sonarr Version(s):
 | Version | Config Type |
 | :---: | :-----------: |
 | 3 | sonarr_v3 |
 | 4 | sonarr_v4 |

### Supported Radarr Version(s):
 | Version | Config Type |
 | :---: | :-----------: |
 | 2 (Untested) | radarr_v2 |
 | 3 | radarr_v3 |
 | 4 | radarr_v4 |
 | 5 | radarr_v5 |

### Supported Lidarr Version(s):
 | Version | Config Type |
 | :---: | :-----------: |
 | 2  | lidarr_v3 |

### Supported Readarr Version(s):
Version 0 was tested specifically on `0.3.17.2406`
 | Version | Config Type |
 | :---: | :-----------: |
 | 0  | readarr_v0 |