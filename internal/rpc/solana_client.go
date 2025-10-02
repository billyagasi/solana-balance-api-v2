package rpc

import (
	"context"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type Client struct{ rpc *rpc.Client }

func NewSolanaClient(url string) *Client { return &Client{rpc: rpc.New(url)} }

func (c *Client) GetBalance(ctx context.Context, addr string) (uint64, error) {
	pk, err := solana.PublicKeyFromBase58(addr)
	if err != nil {
		return 0, err
	}
	ctx2, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	res, err := c.rpc.GetBalance(ctx2, pk, rpc.CommitmentConfirmed)
	if err != nil {
		return 0, err
	}
	return uint64(res.Value), nil
}

