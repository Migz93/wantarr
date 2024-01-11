# wantarr

A simple CLI tool that can be used to search for wanted media in:

- Sonarr
- Radarr

Once an item has been searched, it will not be searched again until the retry days setting has been reached.

## Configuration

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
    url: https://radarr.domain.com
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
```


## Examples

- `wantarr missing radarr -v -m 20`
- `wantarr cutoff radarr4k -v -m 20`

## Help
```
Available Commands:
  cutoff      Search for cutoff unmet media files
  help        Help about any command
  missing     Search for missing media files

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

## Notes

Supported Sonarr Version(s):

- 3
- 4

Supported Radarr Version(s):

- 2
- 3
- 4 (Untested)
- 5