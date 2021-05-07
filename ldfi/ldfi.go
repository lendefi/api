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

/*
	calculateTotalSupply the total supply of all LDFI tokens in wei

	ETH total minus:
	- vesting
	- project wallet balance in ETH chain
	- project wallet balance in BSC chain
*/
func calculateTotalSupply(ethLDFITotal, ethVestingTotal, ethProjectBalance, bscProjectBalance *big.Int) (totalSupply *big.Int) {
	totalSupply = ethLDFITotal.Sub(ethLDFITotal, ethVestingTotal)
	totalSupply.Sub(totalSupply, ethProjectBalance)
	totalSupply.Sub(totalSupply, bscProjectBalance)
	return
}

// Client connect to BSC and ETH chains explorers
type Client struct {
	// API client for bscscan.com
	bsc *etherscan.Client
	// API client for etherscan.com
	eth *etherscan.Client
	// address of the LDFI token on ETH
	ethToken string
	// address of the LDFI token on BSC
	bscToken string
	// address of the project wallet on both chains
	projectWallet string
	// address of vesting contract on ETH chain
	vestingContract string
	// decimals of the token
	decimals uint
}

// NewClient create new instance of Client
func NewClient(etherscanAPIkey, bscscanAPIkey string) *Client {
	return &Client{
		eth: etherscan.New(etherscan.Mainnet, etherscanAPIkey),
		bsc: etherscan.NewCustomized(etherscan.Customization{
			Timeout: time.Second * 5,
			Key:     bscscanAPIkey,
			BaseURL: "https://api.bscscan.com/api?",
			Verbose: false,
		}),
		ethToken:        "0x5479d565e549f3ecdbde4ab836d02d86e0d6a8c7",
		bscToken:        "0xae1119b918f971f232fed504d48604d5fef7277f",
		projectWallet:   "0x9200F737DE4D0BdAdCe8EF83c9f7f1A087569456",
		vestingContract: "0xcE8996314F80200974Bba394Caa19Fc1D41225F9",
		decimals:        18,
	}
}

// GetTotalSupply get from the ETH and BSC scanners the total supply
func (l *Client) GetTotalSupply() (*float64, error) {
	var (
		ethLDFI, ethVesting, ethProjectBalance, bscProjectBalance *etherscan.BigInt
		group                                                     errgroup.Group
	)

	/*
		Perform 4 API calls to BSC/ETH scanners in parallels for speed
	*/

	// get total supply of LDFI on ETH
	group.Go(func() (err error) {
		if ethLDFI, err = l.eth.TokenTotalSupply(l.ethToken); err != nil {
			return fmt.Errorf("eth.TokenTotalSupply(%q): %w", l.ethToken, err)
		}
		return nil
	})

	// get total balance of vesting contract on ETH
	group.Go(func() (err error) {
		if ethVesting, err = l.eth.TokenBalance(l.ethToken, l.vestingContract); err != nil {
			return fmt.Errorf("eth.TokenBalance(%q, %q): %w", l.ethToken, l.vestingContract, err)
		}
		return nil
	})

	// get balance of project wallet on BSC
	group.Go(func() (err error) {
		if ethProjectBalance, err = l.eth.TokenBalance(l.ethToken, l.projectWallet); err != nil {
			return fmt.Errorf("eth.TokenBalance(%q, %q): %w", l.ethToken, l.projectWallet, err)
		}
		return nil
	})

	// get balance of project wallet on BSC
	group.Go(func() (err error) {
		if bscProjectBalance, err = l.bsc.TokenBalance(l.bscToken, l.projectWallet); err != nil {
			return fmt.Errorf("bsc.TokenBalance(%q, %q): %w", l.bscToken, l.projectWallet, err)
		}
		return nil
	})

	// run all requests
	if err := group.Wait(); err != nil {
		// at least 1 of them failed
		return nil, err
	}

	// no requests failure

	weiSupply := calculateTotalSupply(
		ethLDFI.Int(),
		ethVesting.Int(),
		ethProjectBalance.Int(),
		bscProjectBalance.Int(),
	)

	// convert to tokens
	bf := new(big.Float)
	bf.SetInt(weiSupply)
	quotient := new(big.Float).SetInt(big.NewInt(int64(math.Pow10(int(l.decimals)))))
	bf.Quo(bf, quotient)

	f, _ := bf.Float64()
	return &f, nil
}

// NewClientFromEnv is like NewLDFIClient but initialize itself through environment variables.
// API_ETHERSCAN for etherscan.com and API_BSCSCAN for bscscan.com
func NewClientFromEnv() (*Client, error) {
	const (
		envKeyETH                      = "API_ETHERSCAN"
		envKeyBSC                      = "API_BSCSCAN"
		errFormatMissingEnvironmentKey = "Missing environment key %q"
	)

	var (
		ethAPIKey = os.Getenv(envKeyETH)
		bscAPIKey = os.Getenv(envKeyBSC)
	)

	if len(ethAPIKey) == 0 {
		return nil, fmt.Errorf(errFormatMissingEnvironmentKey, envKeyETH)
	}

	if len(bscAPIKey) == 0 {
		return nil, fmt.Errorf(errFormatMissingEnvironmentKey, envKeyBSC)
	}

	return NewClient(ethAPIKey, bscAPIKey), nil
}
