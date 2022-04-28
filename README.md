<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Rosetta" src="https://www.rosetta-api.org/img/rosetta_header.png">
  </a>
</p>
<h3 align="center">
   Rosetta Conflux
</h3>
<p align="center">
  <!-- <a href="https://circleci.com/gh/coinbase/rosetta-conflux/tree/master"><img src="https://circleci.com/gh/coinbase/rosetta-conflux/tree/master.svg?style=shield" /></a>
  <a href="https://coveralls.io/github/coinbase/rosetta-conflux"><img src="https://coveralls.io/repos/github/coinbase/rosetta-conflux/badge.svg" /></a> -->
  <a href="https://goreportcard.com/report/github.com/conflux-fans/rosetta-conflux"><img src="https://goreportcard.com/badge/github.com/conflux-fans/rosetta-conflux" /></a>
  <a href="https://github.com/conflux-fans/rosetta-conflux/blob/master/LICENSE.txt"><img src="https://img.shields.io/github/license/conflux-fans/rosetta-conflux.svg" /></a>
  <img alt="GitHub go.mod Go version" src="https://img.shields.io/github/go-mod/go-version/conflux-fans/rosetta-conflux">
</p>

<p align="center"><b>
ROSETTA-CONFLUX IS CONSIDERED <a href="https://en.wikipedia.org/wiki/Software_release_life_cycle#Alpha">ALPHA SOFTWARE</a>.
USE AT YOUR OWN RISK! CONFLUX ASSUMES NO RESPONSIBILITY NOR LIABILITY IF THERE IS A BUG IN THIS IMPLEMENTATION.
</b></p>

## Overview
`rosetta-conflux` provides a reference implementation of the Rosetta API for
Conflux in Golang. If you haven't heard of the Rosetta API, you can find more
information [here](https://rosetta-api.org).

## Features
* Comprehensive tracking of all CFX balance changes
* Stateless, offline, curve-based transaction construction (with address checksum validation)
* Atomic balance lookups using conflux-rust Endpoint
* Idempotent access to all transaction traces and receipts

## Usage
As specified in the [Rosetta API Principles](https://www.rosetta-api.org/docs/automated_deployment.html),
all Rosetta implementations must be deployable via Docker and support running via either an
[`online` mode](https://www.rosetta-api.org/docs/node_deployment.html#multiple-modes).

**YOU MUST INSTALL DOCKER FOR THE FOLLOWING INSTRUCTIONS TO WORK. YOU CAN DOWNLOAD
DOCKER [HERE](https://www.docker.com/get-started).**

### Install
Running the following commands will create a Docker image called `rosetta-conflux:latest`.

#### From GitHub
To download the pre-built Docker image from the latest release, run:
```text
curl -sSfL https://raw.githubusercontent.com/coinbase/rosetta-conflux/master/install.sh | sh -s
```

#### From Source
After cloning this repository, run:
```text
make build-local
```

### Run
Running the following commands will start a Docker container in
[detached mode](https://docs.docker.com/engine/reference/run/#detached--d) with
a data directory at `<working directory>/conflux-data` and the Rosetta API accessible
at port `8080`.

#### Configuration Environment Variables
* `MODE` (required) - Determines if Rosetta can make outbound connections. Options: `ONLINE` or `OFFLINE`.
* `NETWORK` (required) - Conflux network to launch and/or communicate with. Options: `MAINNET` or `TESTNET` (which defaults to `TESTNET` for backwards compatibility).
* `PORT`(required) - Which port to use for Rosetta.
* `CFXNODE` (optional) - Point to a remote `conflux-rust` node instead of initializing one

#### Mainnet:Online (Remote)
```text
docker run -d --rm --ulimit "nofile=100000:100000" -e "MODE=ONLINE" -e "NETWORK=MAINNET" -e "PORT=8080" -e "GETH=<NODE URL>" -p 8080:8080 -p 30303:30303 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-mainnet-remote cfxnode=<NODE URL>`._

#### Testnet:Online (Remote)
```text
docker run -d --rm --ulimit "nofile=100000:100000" -e "MODE=ONLINE" -e "NETWORK=TESTNET" -e "PORT=8080" -e "GETH=<NODE URL>" -p 8080:8080 -p 30303:30303 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-testnet-remote cfxnode=<NODE URL>`._

## System Requirements
`rosetta-conflux` has been tested on an [AWS c5.2xlarge instance](https://aws.amazon.com/ec2/instance-types/c5).
This instance type has 8 vCPU and 16 GB of RAM. If you use a computer with less than 16 GB of RAM,
it is possible that `rosetta-conflux` will exit with an OOM error.

### Recommended OS Settings
To increase the load `rosetta-conflux` can handle, it is recommended to tune your OS
settings to allow for more connections. On a linux-based OS, you can run the following
commands ([source](http://www.tweaked.io/guide/kernel)):
```text
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
sysctl -w net.ipv4.tcp_max_syn_backlog=10000
sysctl -w net.core.somaxconn=10000
sysctl -p (when done)
```
_We have not tested `rosetta-conflux` with `net.ipv4.tcp_tw_recycle` and do not recommend
enabling it._

You should also modify your open file settings to `100000`. This can be done on a linux-based OS
with the command: `ulimit -n 100000`.

### Conflux-Rust Settings

Conflux-Rust is conflux node program, we need run a conflux-rust node be rosetta-conflux backend RPC server. 

Please find [Run Conflux Node](https://developer.confluxnetwork.org/conflux-doc/docs/get_started) to setup conflux full node

You should set conflux full node config file as bellow for rosetta-conflux could access all RPC and more data it needs. The meanings of these could be find from [here](https://github.com/Conflux-Chain/conflux-rust/blob/master/run/hydra.toml)

```
node_type = "archive"
persist_block_number_index = true
persist_tx_index = true
additional_maintained_snapshot_count = 90
executive_trace = true
public_rpc_apis = "all"

dev_pos_private_key_encryption_password = ""

jsonrpc_ws_eth_port=8546
jsonrpc_ws_max_payload_bytes=209715200
jsonrpc_http_keep_alive=true

pow_problem_window_size=10
```

The mininal query available time of conflux node is 22 hours, imporve available time by config `additional_maintained_snapshot_count`, per 1 will increase 2000 seconds, so if you need query data avaialble time for 3 days, please set it to 90. But the larger value need more storage for stroe snapshot, about 11G per snapshot, so please set it according to your usage scenario.

The `dev_pos_private_key_encryption_password` is used for PoS mining and nothing to do with be a RPC server , so set it be empty


## Testing with rosetta-cli
To validate `rosetta-conflux`, [install `rosetta-cli`](https://github.com/coinbase/rosetta-cli#install)

Firstly [run a corresponding conflux node](#conflux-rust-settings) and set the online_url in config to the conflux node url. Then run one of the following commands:
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/testnet/config.json` - This command validates that the Data API implementation is correct using the conflux testnet node. It also ensures that the implementation does not miss any balance-changing operations.
* `rosetta-cli check:construction --configuration-file rosetta-cli-conf/testnet/config.json` - This command validates the Construction API implementation. It also verifies transaction construction, signing, and submissions to the testnet network.
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/mainnet/config.json` - This command validates that the Data API implementation is correct using the conflux mainnet node. It also ensures that the implementation does not miss any balance-changing operations.

## Future Work
* Add ERC-20 Rosetta Module to enable reading ERC-20 token transfers and transaction construction
* [Rosetta API `/mempool/*`](https://www.rosetta-api.org/docs/MempoolApi.html) implementation
<!-- * Add more methods to the `/call` endpoint (currently only supports `eth_getBlockByNumber` and `eth_getTransactionReceipt`) -->
* Add CI test using `rosetta-cli` to run on each PR (likely on a regtest network)

_Please reach out on our [community](https://community.rosetta-api.org) if you want to tackle anything on this list!_

## Development
* `make deps` to install dependencies
* `make test` to run tests
* `make lint` to lint the source code
* `make salus` to check for security concerns
* `make build-local` to build a Docker image from the local context
* `make coverage-local` to generate a coverage report

## License
This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).

© 2022 Conflux
