package swapkit

import (
	"testing"

	coinpkg "github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
	"github.com/stretchr/testify/require"
)

func TestNewQuoteRequestFromCoinCodesRejectsEqualAssets(t *testing.T) {
	quoteRequest, quoteError := newQuoteRequestFromCoinCodes(
		coinpkg.Code("btc"),
		coinpkg.Code("tbtc"),
		"1",
		[]string{"NEAR"},
	)

	require.Nil(t, quoteRequest)
	require.Equal(t, &QuoteError{
		ErrorCode: ErrInvalidRequest,
		Message:   "Sell and buy coins must differ.",
	}, quoteError)
}
