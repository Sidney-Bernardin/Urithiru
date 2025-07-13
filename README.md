# Urithiru

# Urithiru

**Contents**
1. [Overview](#overview)
1. [Usage](#usage)

## Overview
This is a layer-4 reverse proxy that can be easily configured with a .toml file.

For more on how this project works, visit my [portfolio](https://sidney-bernardin.github.io/project/?id=urithiru).

## Usage

### Build

#### From source with Go.
```bash
git clone https://github.com/Sidney-Bernardin/Urithiru.git
cd Urithiru
go install ./cmd/urithiru/.
```

#### Pre-built from Docker Hub.
```bash
docker pull sidneybernardin/urithiru:latest
```

### Run
Run the executable with the following command line arguments.
```bash
Usage of urithiru:
  -config string
        Path to configuration file. (default "/etc/urithiru/config.toml")
  -pprof_addr string
        Address for the PPROF server. (default ":6060")
```

### Configuration guide
```toml
# All default values

pingTimeout = "10s" # Timeout for a backend ping message.
pingInterval = "1s" # Frequency of backend ping messages.

[[ proxies ]]
name = ""
addr = "" # Address to listen on.
buffer_size = 1024 # Buffer size for each TCP connection.
tls_cert = "" # TLS certificate.
tls_key = "" # TLS key.
pingTimeout = "<defaults to parent's value>"
pingInterval = "<defaults to parent's value>"

[[ proxies.backends ]]
addr = "" # Address of a backend server.
pingTimeout = "<defaults to parent's value>"
pingInterval = "<defaults to parent's value>"

# Configure as many as you need.
# [[ proxies ]]
# [[ proxies.backends ]]
# [[ proxies.backends ]]
# ...

# [[ proxies ]]
# [[ proxies.backends ]]
# ...
```
