package swapkit

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"

	coinpkg "github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
)

var swapkitAssetByCoinCode = map[coinpkg.Code]string{
	coinpkg.CodeBTC:    "BTC.BTC",
	coinpkg.CodeTBTC:   "BTC.BTC",
	coinpkg.CodeRBTC:   "BTC.BTC",
	coinpkg.CodeETH:    "ETH.ETH",
	coinpkg.CodeSEPETH: "ETH.ETH",

	coinpkg.Code("eth-erc20-usdt"):      "ETH.USDT-0xdac17f958d2ee523a2206206994597c13d831ec7",
	coinpkg.Code("eth-erc20-usdc"):      "ETH.USDC-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	coinpkg.Code("eth-erc20-link"):      "ETH.LINK-0x514910771af9ca656af840dff83e8264ecf986ca",
	coinpkg.Code("eth-erc20-bat"):       "ETH.BAT-0x0d8775f648430679a709e98d2b0cb6250d2887ef",
	coinpkg.Code("eth-erc20-mkr"):       "ETH.MKR-0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
	coinpkg.Code("eth-erc20-zrx"):       "ETH.ZRX-0xe41d2489571d322189246dafa5ebde1f4699f498",
	coinpkg.Code("eth-erc20-wbtc"):      "ETH.WBTC-0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
	coinpkg.Code("eth-erc20-paxg"):      "ETH.PAXG-0x45804880De22913dAFE09f4980848ECE6EcbAf78",
	coinpkg.Code("eth-erc20-dai0x6b17"): "ETH.DAI-0x6b175474e89094c44da98b954eedeac495271d0f",
}

const apiKey = "0722e09f-9d3f-4817-a870-069848d03ee9"

// ErrInvalidRequest is returned when the quote request is invalid, for example due to missing or invalid fields.
const ErrInvalidRequest errp.ErrorCode = "invalidRequest"

func newQuoteRequestFromCoinCodes(
	sellCoinCode, buyCoinCode coinpkg.Code,
	sellAmount string,
	providers []string,
) (*QuoteRequest, *QuoteError) {
	sellAmount = strings.TrimSpace(sellAmount)

	if sellCoinCode == "" {
		return nil, &QuoteError{
			ErrorCode: ErrInvalidRequest,
			Message:   "Missing sellCoinCode.",
		}
	}
	if buyCoinCode == "" {
		return nil, &QuoteError{
			ErrorCode: ErrInvalidRequest,
			Message:   "Missing buyCoinCode.",
		}
	}
	parsedSellAmount, ok := new(big.Rat).SetString(sellAmount)
	if sellAmount == "" {
		return nil, &QuoteError{
			ErrorCode: ErrInvalidRequest,
			Message:   "Missing sellAmount.",
		}
	}
	if !ok || parsedSellAmount.Sign() <= 0 {
		return nil, &QuoteError{
			ErrorCode: ErrInvalidRequest,
			Message:   "Invalid sellAmount.",
		}
	}
	sellAsset, ok := swapkitAssetByCoinCode[sellCoinCode]
	if !ok {
		return nil, &QuoteError{
			ErrorCode: ErrInvalidRequest,
			Message:   "Unsupported sell asset.",
		}
	}
	buyAsset, ok := swapkitAssetByCoinCode[buyCoinCode]
	if !ok {
		return nil, &QuoteError{
			ErrorCode: ErrInvalidRequest,
			Message:   "Unsupported buy asset.",
		}
	}
	if sellAsset == buyAsset {
		return nil, &QuoteError{
			ErrorCode: ErrInvalidRequest,
			Message:   "Sell and buy coins must differ.",
		}
	}
	return &QuoteRequest{
		SellAsset:  sellAsset,
		BuyAsset:   buyAsset,
		SellAmount: sellAmount,
		Providers:  providers,
	}, nil
}

// GetQuoteRoutes validates the provided coin codes, fetches a quote, and returns the route
// summaries needed by the frontend.
func GetQuoteRoutes(
	sellCoinCode, buyCoinCode coinpkg.Code,
	sellAmount string,
) ([]QuoteRouteSummary, *QuoteError) {
	quoteRequest, quoteError := newQuoteRequestFromCoinCodes(
		sellCoinCode,
		buyCoinCode,
		sellAmount,
		[]string{"NEAR"},
	)
	if quoteError != nil {
		return nil, quoteError
	}

	quoteResponse, err := NewClient(apiKey).Quote(context.Background(), quoteRequest)
	if err != nil {
		if quoteError, ok := quoteErrorFromError(err); ok {
			return nil, quoteError
		}
		return nil, &QuoteError{
			ErrorCode: errp.ErrorCode("unexpectedError"),
			Message:   err.Error(),
		}
	}
	return quoteRouteSummariesFromResponse(quoteResponse), nil
}

func quoteRouteSummariesFromResponse(quoteResponse *QuoteResponse) []QuoteRouteSummary {
	if quoteResponse == nil {
		return nil
	}

	routes := make([]QuoteRouteSummary, 0, len(quoteResponse.Routes))
	for i := range quoteResponse.Routes {
		route := &quoteResponse.Routes[i]
		routes = append(routes, QuoteRouteSummary{
			RouteID:           route.RouteID,
			ExpectedBuyAmount: route.ExpectedBuyAmount,
		})
	}

	return routes
}

// quoteErrorFromError extracts a structured quote error from a SwapKit client error.
// It parses the json payload from the error message and returns a QuoteError if successful.
// These are the valid errors returned:
// https://docs.swapkit.dev/swapkit-api/v3-quote-request-a-swap-quote#quote-error-schema
func quoteErrorFromError(err error) (*QuoteError, bool) {
	raw := err.Error()
	jsonStart := strings.Index(raw, "{")
	if jsonStart == -1 {
		return nil, false
	}
	var payload struct {
		Provider  string         `json:"provider"`
		ErrorCode errp.ErrorCode `json:"error"` // While the docs mention a field named "errorCode", the actual error response uses "error".
		Message   string         `json:"message"`
	}
	if unmarshalErr := json.Unmarshal([]byte(raw[jsonStart:]), &payload); unmarshalErr != nil {
		return nil, false
	}
	if payload.ErrorCode == "" || payload.Message == "" {
		return nil, false
	}
	return &QuoteError{
		Provider:  payload.Provider,
		ErrorCode: payload.ErrorCode,
		Message:   payload.Message,
	}, true
}
