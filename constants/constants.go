package constants

// NOTICE: do not reference any other packages in this file, otherwise it will cause import cycle error

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// App

const (
	ApplicationName = "kairoschain" // renamed from "kairoschain"
	ApplicationBinaryName = "kairosd" // renamed from "kairosd"
	ApplicationHome = ".kairoschain" // renamed from ".kairoschain"

	GitHubRepo = "https://github.com/HarryBin2002/kairoschain" // renamed from "https://github.com/HarryBin2002/kairoschain"
)

// Denom

const (
	// BaseDenom defines the default coin denomination used on this chain in:
	//
	// - Staking parameters: denomination used as stake in the dPoS chain
	// - Mint parameters: denomination minted due to fee distribution rewards
	// - Governance parameters: denomination used for spam prevention in proposal deposits
	// - Crisis parameters: constant fee denomination used for spam prevention to check broken invariant
	// - EVM parameters: denomination used for running EVM state transitions in this chain.
	//
	// During code, we will use the term 'native coin' for this denomination
	// so other developers can take advantage of renaming constants when forking this project.
	BaseDenom = "ukai" // renamed from "ukai"
	DisplayDenom = "kai" // renamed from "ether"
	SymbolDenom = "KAI" // renamed from "ETH"
	BaseDenomExponent = 18
)

// Bech32

const (
	// Bech32Prefix is the HRP (human-readable part) of the Bech32 encoded address of this chain
	Bech32Prefix = "kairos"

	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = Bech32Prefix

	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = Bech32Prefix + sdk.PrefixPublic

	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator

	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator + sdk.PrefixPublic

	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus

	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus + sdk.PrefixPublic
)

// Chain ID

const (
	ChainIdPrefix = "kairoschain"

	// MainnetChainID defines the Cosmos-style EIP155 chain ID for mainnet
	MainnetChainID = ChainIdPrefix + "_123567"
	// TestnetChainID defines the Cosmos-style EIP155 chain ID for testnet
	TestnetChainID = ChainIdPrefix + "_5678"
	// DevnetChainID defines the Cosmos-style EIP155 chain ID for devnet
	DevnetChainID = ChainIdPrefix + "_1234"

	MainnetFullChainId = MainnetChainID + "-1"
	TestnetFullChainId = TestnetChainID + "-1"
	DevnetFullChainId  = DevnetChainID + "-1"

	MainnetEIP155ChainId = 123567
	TestnetEIP155ChainId = 5678
	DevnetEIP155ChainId  = 1234
)

