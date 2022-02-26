package bridge

import (
	"context"
	"fmt"
	"math/big"

	"bridge-oracle/internal/avalanche"
	"bridge-oracle/internal/tezos"
	"go.uber.org/zap"
)

type Event interface {
	User() string
	Amount() *big.Int
	Destination() string
}

type Bridge struct {
	avalanche *avalanche.Avalanche
	tezos     *tezos.Tezos

	logger *zap.SugaredLogger
}

func New(avalanche *avalanche.Avalanche, tezos *tezos.Tezos, logger *zap.SugaredLogger) *Bridge {
	return &Bridge{
		avalanche: avalanche,
		tezos:     tezos,
		logger:    logger,
	}
}

func (b *Bridge) Run(ctx context.Context) error {
	avaSub, err := b.avalanche.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("subscribe avalanche: %w", err)
	}

	tzsSub, err := b.tezos.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("subscribe tezos: %w", err)
	}

	b.loop(ctx, avaSub, tzsSub)

	return nil
}

func (b *Bridge) loop(ctx context.Context, avaSub *avalanche.Subscription, tzsSub *tezos.Subscription) {
	for {
		select {
		// Break loop on interruption
		case <-ctx.Done():
			return

		// Handle events from chains and call another chain
		case event := <-avaSub.OnAVAXLocked():
			b.mintWAVAX(ctx, event)
		case event := <-avaSub.OnUSDCLocked():
			b.mintWUSDC(ctx, event)
		case event := <-tzsSub.OnWAVAXBurned():
			b.unlockAVAX(ctx, event)
		case event := <-tzsSub.OnWUSDCBurned():
			b.unlockUSDC(ctx, event)

		// Handle errors occurred during chains subscriptions
		case err := <-avaSub.Err():
			b.logger.Errorf("avalanche subscribtion error: %s", err)
		case err := <-tzsSub.Err():
			b.logger.Errorf("tezos subscribtion error: %s", err)
		}
	}
}

func (b *Bridge) mintWAVAX(ctx context.Context, event Event) {
	hash, fee, err := b.tezos.MintWAVAX(event.Destination(), event.Amount())
	if err != nil {
		b.logger.Errorf("mint wavax: %s", err)

		return
	}

	b.logger.With(
		zap.String("user", event.User()),
		zap.Int64("amount", event.Amount().Int64()),
		zap.String("destination", event.Destination()),
		zap.String("tx_hash", hash),
		zap.Int64("fee", fee.Int64()),
	).Info("wavax minted")
}

func (b *Bridge) mintWUSDC(ctx context.Context, event Event) {
	hash, fee, err := b.tezos.MintWUSDC(event.Destination(), event.Amount())
	if err != nil {
		b.logger.Errorf("mint wusdc: %s", err)

		return
	}

	b.logger.With(
		zap.String("user", event.User()),
		zap.Int64("amount", event.Amount().Int64()),
		zap.String("destination", event.Destination()),
		zap.String("tx_hash", hash),
		zap.Int64("fee", fee.Int64()),
	).Info("wusdc minted")
}

func (b *Bridge) unlockAVAX(ctx context.Context, event Event) {
	hash, fee, err := b.avalanche.UnlockAVAX(ctx, event.Destination(), event.Amount())
	if err != nil {
		b.logger.Errorf("unlock avax: %s", err)

		return
	}

	b.logger.With(
		zap.String("user", event.User()),
		zap.Int64("amount", event.Amount().Int64()),
		zap.String("destination", event.Destination()),
		zap.String("tx_hash", hash),
		zap.Int64("fee", fee.Int64()),
	).Info("avax unlocked")
}

func (b *Bridge) unlockUSDC(ctx context.Context, event Event) {
	hash, fee, err := b.avalanche.UnlockUSDC(ctx, event.Destination(), event.Amount())
	if err != nil {
		b.logger.Errorf("unlock usdc: %s", err)

		return
	}

	b.logger.With(
		zap.String("user", event.User()),
		zap.Int64("amount", event.Amount().Int64()),
		zap.String("destination", event.Destination()),
		zap.String("tx_hash", hash),
		zap.Int64("fee", fee.Int64()),
	).Info("usdc unlocked")
}
