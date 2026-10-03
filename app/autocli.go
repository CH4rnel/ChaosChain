package app

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"cosmossdk.io/client/v2/autocli"
	"github.com/cosmos/cosmos-sdk/client"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authmodule "github.com/cosmos/cosmos-sdk/x/auth"
	bankmodule "github.com/cosmos/cosmos-sdk/x/bank"
	distributionmodule "github.com/cosmos/cosmos-sdk/x/distribution"
	govmodule "github.com/cosmos/cosmos-sdk/x/gov"
	stakingmodule "github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/spf13/cobra"
)

func addModuleCLICommands(rootCmd *cobra.Command, clientCtx client.Context) error {
	options := autocli.AppOptions{
		ModuleOptions: map[string]*autocliv1.ModuleOptions{
			"auth":         (authmodule.AppModule{}).AutoCLIOptions(),
			"bank":         (bankmodule.AppModule{}).AutoCLIOptions(),
			"distribution": (distributionmodule.AppModule{}).AutoCLIOptions(),
			"gov":          (govmodule.AppModule{}).AutoCLIOptions(),
			"staking":      (stakingmodule.AppModule{}).AutoCLIOptions(),
		},
		AddressCodec:          addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
		ClientCtx:             clientCtx,
	}
	return options.EnhanceRootCommand(rootCmd)
}
