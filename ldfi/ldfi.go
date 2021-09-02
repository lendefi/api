package ldfi

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/nanmu42/etherscan-api"
	"golang.org/x/sync/errgroup"
)

type multiAccountBalanceError struct {
	client *Client
	err    error
}

func (m *multiAccountBalanceError) Error() string {
	return fmt.Sprintf(
		"eth.MultiAccountBalance(%q, %q, %q, %q): %q",
		m.client.token,
		m.client.vestingContract,
		m.client.projectWallet,
		m.client.burnAddress,
		m.err.Error(),
	)
}

// Client connect to BSC and ETH chains explorers
type Client struct {
	// API client for bscscan.com
	client *etherscan.Client
	// address of the LDFI token
	token string
	// address of the project wallet
	projectWallet string
	// address of vesting contract on ETH chain
	vestingContract string
	// address where token goes to die
	burnAddress string
	// decimals of the token
	decimals uint
	// max supply
	maxSupply etherscan.BigInt
}

// NewClient create new instance of Client
func NewClient(bscscanAPIkey string) *Client {
	return &Client{
		client: etherscan.NewCustomized(etherscan.Customization{
			Timeout: time.Second * 5,
			Key:     bscscanAPIkey,
			BaseURL: "https://api.bscscan.com/api?",
			Verbose: false,
		}),
		token:           "0x8f1e60d84182db487ac235acc65825e50b5477a1",
		projectWallet:   "0x30DD781D2143fE32C36E894a049898f268b82092",
		vestingContract: "0xc598d81c62f6391b2412d02a78fa3f3affe58b52",
		burnAddress:     "0x000000000000000000000000000000000000dead",
		decimals:        18,
	}
}

func (l *Client) weiToFloat(i *big.Int) float64 {
	bf := new(big.Float)
	bf.SetInt(i)
	quotient := new(big.Float).SetInt(big.NewInt(int64(math.Pow10(int(l.decimals)))))
	bf.Quo(bf, quotient)
	output, _ := bf.Float64()
	return output
}

type Supplies struct {
	Total       float64
	Circulating float64
	Max         float64
}

func copyBigInt(i *big.Int) (o *big.Int) {
	o = new(big.Int)
	o = o.Set(i)
	return
}

// GetSupplies get BSC scanners total and circulating supplies
func (l *Client) GetSupplies() (*Supplies, error) {
	l.client.EtherTotalSupply()
	var (
		maxSupply, burnBalance, vesting, projectBalance *etherscan.BigInt
		group                                           errgroup.Group
	)

	/*
		Perform 4 API calls to BSC/ETH scanners in parallels for speed
	*/

	// get total supply of LDFI on ETH
	group.Go(func() (err error) {
		if maxSupply, err = l.client.TokenTotalSupply(l.token); err != nil {
			return fmt.Errorf("client.TokenTotalSupply(%q): %w", l.token, err)
		}
		return nil
	})

	// get balances
	group.Go(func() (err error) {
		balances, err := l.client.MultiAccountBalance(l.token, l.vestingContract, l.projectWallet, l.burnAddress)
		if err != nil {
			return &multiAccountBalanceError{client: l, err: err}
		}

		lenBalances := len(balances)
		if lenBalances != 4 {
			return &multiAccountBalanceError{client: l, err: fmt.Errorf("Invalid number of balances %d, expected 4", lenBalances)}
		}

		vesting = balances[0].Balance
		projectBalance = balances[1].Balance
		burnBalance = balances[2].Balance

		return nil
	})

	// run all requests
	if err := group.Wait(); err != nil {
		// at least 1 of them failed
		return nil, err
	}

	// no requests failure

	// Total Supply = Max Supply minus Burnt Tokens
	totalSupply := copyBigInt(maxSupply.Int())
	totalSupply = totalSupply.Sub(totalSupply, burnBalance.Int())

	// Circulating Supply = Total Supply minus Vesting Contract minus Project Wallet
	calculateTotalSupply := copyBigInt(maxSupply.Int())
	calculateTotalSupply = calculateTotalSupply.Sub(calculateTotalSupply, vesting.Int())
	calculateTotalSupply = calculateTotalSupply.Sub(calculateTotalSupply, projectBalance.Int())

	return &Supplies{
		Total:       l.weiToFloat(totalSupply),
		Circulating: l.weiToFloat(calculateTotalSupply),
		Max:         l.weiToFloat(maxSupply.Int()),
	}, nil
}

// NewClientFromEnv is like NewLDFIClient but initialize itself through environment variables.
// API_ETHERSCAN for etherscan.com and API_BSCSCAN for bscscan.com
func NewClientFromEnv() (*Client, error) {
	const (
		envKeyBSC                      = "API_BSCSCAN"
		errFormatMissingEnvironmentKey = "Missing environment key %q"
	)

	bscAPIKey := os.Getenv(envKeyBSC)

	if len(bscAPIKey) == 0 {
		return nil, fmt.Errorf(errFormatMissingEnvironmentKey, envKeyBSC)
	}

	return NewClient(bscAPIKey), nil
}
