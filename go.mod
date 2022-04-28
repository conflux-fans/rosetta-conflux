module github.com/conflux-fans/rosetta-conflux

go 1.16

require (
	github.com/Conflux-Chain/go-conflux-sdk v1.1.4
	github.com/coinbase/rosetta-sdk-go v0.7.1
	github.com/ethereum/go-ethereum v1.10.15
	github.com/fatih/color v1.10.0
	github.com/neilotoole/errgroup v0.1.6
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v1.1.1
)

// replace github.com/Conflux-Chain/go-conflux-sdk v1.0.19 => ../go-conflux-sdk
replace github.com/Conflux-Chain/go-conflux-sdk v1.0.19 => github.com/wangdayong228/go-conflux-sdk v1.0.15-0.20220106052702-43ad3e602ea3
