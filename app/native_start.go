package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"cosmossdk.io/log/v2"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

func runNativeCometNode(serverCtx *server.Context, appCreator servertypes.AppCreator, opts server.StartCmdOptions) error {
	appConfig, err := serverconfig.ParseConfig(serverCtx.Viper)
	if err != nil {
		return fmt.Errorf("parse application server configuration: %w", err)
	}
	if appConfig.API.Enable || appConfig.GRPC.Enable || appConfig.GRPCWeb.Enable {
		return fmt.Errorf("start-native currently supports CometBFT RPC only; disable API, gRPC, and gRPC-Web in app.toml")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	home := serverCtx.Config.RootDir
	if home == "" {
		home = DefaultNodeHome
	}
	serverCtx.Config.SetRoot(home)
	cmtcfg.EnsureRoot(home)

	appDB, err := opts.DBOpener(home, server.GetAppDBBackend(serverCtx.Viper))
	if err != nil {
		return fmt.Errorf("open application database: %w", err)
	}
	defer appDB.Close()

	chaosApp := appCreator(serverCtx.Logger, appDB, serverCtx.Viper)
	defer func() {
		if err := chaosApp.Close(); err != nil {
			serverCtx.Logger.Error("close application", "error", err)
		}
	}()

	nodeKey, err := p2p.LoadOrGenNodeKey(serverCtx.Config.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("load or create node key: %w", err)
	}
	genesisDocProvider := func() (*cmttypes.GenesisDoc, error) {
		appGenesis, err := genutiltypes.AppGenesisFromFile(serverCtx.Config.GenesisFile())
		if err != nil {
			return nil, err
		}
		return appGenesis.ToGenesisDoc()
	}

	tmNode, err := node.NewNodeWithContext(
		ctx,
		serverCtx.Config,
		privval.LoadOrGenFilePV(serverCtx.Config.PrivValidatorKeyFile(), serverCtx.Config.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(newCometABCIAdapter(chaosApp)),
		genesisDocProvider,
		cmtcfg.DefaultDBProvider,
		node.DefaultMetricsProvider(serverCtx.Config.Instrumentation),
		servercmtlog.CometLoggerWrapper{Logger: serverCtx.Logger.With("module", "cometbft")},
	)
	if err != nil {
		return fmt.Errorf("construct CometBFT node: %w", err)
	}
	if err := tmNode.Start(); err != nil {
		return fmt.Errorf("start CometBFT node: %w", err)
	}
	<-ctx.Done()
	if err := tmNode.Stop(); err != nil {
		return fmt.Errorf("stop CometBFT node: %w", err)
	}
	return nil
}

var _ cmtlog.Logger = servercmtlog.CometLoggerWrapper{Logger: log.NewNopLogger()}
