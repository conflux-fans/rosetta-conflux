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
  <a href="https://coveralls.io/github/coinbase/rosetta-conflux"><img src="https://coveralls.io/repos/github/coinbase/rosetta-conflux/badge.svg" /></a>
  <a href="https://goreportcard.com/report/github.com/coinbase/rosetta-conflux"><img src="https://goreportcard.com/badge/github.com/coinbase/rosetta-conflux" /></a>
  <a href="https://github.com/coinbase/rosetta-conflux/blob/master/LICENSE.txt"><img src="https://img.shields.io/github/license/coinbase/rosetta-conflux.svg" /></a>
  <a href="https://pkg.go.dev/github.com/coinbase/rosetta-conflux?tab=overview"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=shield" /></a> -->
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
[`online` or `offline` mode](https://www.rosetta-api.org/docs/node_deployment.html#multiple-modes).

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
a data directory at `<working directory>/ethereum-data` and the Rosetta API accessible
at port `8080`.

#### Configuration Environment Variables
* `MODE` (required) - Determines if Rosetta can make outbound connections. Options: `ONLINE` or `OFFLINE`.
* `NETWORK` (required) - Ethereum network to launch and/or communicate with. Options: `MAINNET` or `TESTNET` (which defaults to `TESTNET` for backwards compatibility).
* `PORT`(required) - Which port to use for Rosetta.
* `CFXNODE` (optional) - Point to a remote `conflux-rust` node instead of initializing one
<!-- * `SKIP_GETH_ADMIN` (optional, default: `FALSE`) - Instruct Rosetta to not use the `geth` `admin` RPC calls. This is typically disabled by hosted blockchain node services. -->
<!-- 
#### Mainnet:Online
```text
docker run -d --rm --ulimit "nofile=100000:100000" -v "$(pwd)/ethereum-data:/data" -e "MODE=ONLINE" -e "NETWORK=MAINNET" -e "PORT=8080" -p 8080:8080 -p 30303:30303 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-mainnet-online`._ -->

#### Mainnet:Online (Remote)
```text
docker run -d --rm --ulimit "nofile=100000:100000" -e "MODE=ONLINE" -e "NETWORK=MAINNET" -e "PORT=8080" -e "GETH=<NODE URL>" -p 8080:8080 -p 30303:30303 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-mainnet-remote geth=<NODE URL>`._
<!-- 
#### Mainnet:Offline
```text
docker run -d --rm -e "MODE=OFFLINE" -e "NETWORK=MAINNET" -e "PORT=8081" -p 8081:8081 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-mainnet-offline`._

#### Testnet:Online
```text
docker run -d --rm --ulimit "nofile=100000:100000" -v "$(pwd)/ethereum-data:/data" -e "MODE=ONLINE" -e "NETWORK=TESTNET" -e "PORT=8080" -p 8080:8080 -p 30303:30303 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-testnet-online`._ -->

#### Testnet:Online (Remote)
```text
docker run -d --rm --ulimit "nofile=100000:100000" -e "MODE=ONLINE" -e "NETWORK=TESTNET" -e "PORT=8080" -e "GETH=<NODE URL>" -p 8080:8080 -p 30303:30303 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-testnet-remote geth=<NODE URL>`._
<!-- 
#### Testnet:Offline
```text
docker run -d --rm -e "MODE=OFFLINE" -e "NETWORK=TESTNET" -e "PORT=8081" -p 8081:8081 rosetta-conflux:latest
```
_If you cloned the repository, you can run `make run-testnet-offline`._ -->

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

## Testing with rosetta-cli
To validate `rosetta-conflux`, [install `rosetta-cli`](https://github.com/coinbase/rosetta-cli#install)
and run one of the following commands:
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/testnet/config.json`
* `rosetta-cli check:construction --configuration-file rosetta-cli-conf/testnet/config.json`
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/mainnet/config.json`

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

© 2021 Coinbase
