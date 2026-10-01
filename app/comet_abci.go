package app

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
)

type abciApplication interface {
	Info(*abci.RequestInfo) (*abci.ResponseInfo, error)
	Query(context.Context, *abci.RequestQuery) (*abci.ResponseQuery, error)
	CheckTx(*abci.RequestCheckTx) (*abci.ResponseCheckTx, error)
	InitChain(*abci.RequestInitChain) (*abci.ResponseInitChain, error)
	PrepareProposal(*abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error)
	ProcessProposal(*abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error)
	FinalizeBlock(*abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error)
	ExtendVote(context.Context, *abci.RequestExtendVote) (*abci.ResponseExtendVote, error)
	VerifyVoteExtension(*abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error)
	Commit() (*abci.ResponseCommit, error)
	ListSnapshots(*abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error)
	OfferSnapshot(*abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error)
	LoadSnapshotChunk(*abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error)
	ApplySnapshotChunk(*abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error)
}

type cometABCIAdapter struct {
	app abciApplication
}

func newCometABCIAdapter(app abciApplication) abci.Application {
	return cometABCIAdapter{app: app}
}

func (adapter cometABCIAdapter) Info(_ context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	return adapter.app.Info(req)
}

func (adapter cometABCIAdapter) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	return adapter.app.Query(ctx, req)
}

func (adapter cometABCIAdapter) CheckTx(_ context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return adapter.app.CheckTx(req)
}

func (adapter cometABCIAdapter) InitChain(_ context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	return adapter.app.InitChain(req)
}

func (adapter cometABCIAdapter) PrepareProposal(_ context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	return adapter.app.PrepareProposal(req)
}

func (adapter cometABCIAdapter) ProcessProposal(_ context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	return adapter.app.ProcessProposal(req)
}

func (adapter cometABCIAdapter) FinalizeBlock(_ context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	return adapter.app.FinalizeBlock(req)
}

func (adapter cometABCIAdapter) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	return adapter.app.ExtendVote(ctx, req)
}

func (adapter cometABCIAdapter) VerifyVoteExtension(_ context.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	return adapter.app.VerifyVoteExtension(req)
}

func (adapter cometABCIAdapter) Commit(_ context.Context, _ *abci.RequestCommit) (*abci.ResponseCommit, error) {
	return adapter.app.Commit()
}

func (adapter cometABCIAdapter) ListSnapshots(_ context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return adapter.app.ListSnapshots(req)
}

func (adapter cometABCIAdapter) OfferSnapshot(_ context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return adapter.app.OfferSnapshot(req)
}

func (adapter cometABCIAdapter) LoadSnapshotChunk(_ context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return adapter.app.LoadSnapshotChunk(req)
}

func (adapter cometABCIAdapter) ApplySnapshotChunk(_ context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return adapter.app.ApplySnapshotChunk(req)
}
