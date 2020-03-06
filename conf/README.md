# Usage

The default configuration file is `./config.json`.  
Run program with `-C <path/to/config/file>` to specify another one.

The default configuration could be found in file `./sample-config.json`.

The simplest configuration with other items configured to default value is like:

```json
{
    "network": {
        "p2p": {
            "seeds": "192.168.1.100,192.168.1.200,..."
        }
    }
}
```