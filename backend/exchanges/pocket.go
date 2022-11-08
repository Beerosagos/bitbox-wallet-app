package exchanges

import (
	"fmt"

	"github.com/digitalbitbox/bitbox-wallet-app/backend/accounts"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/coins/coin"
)

const (
	// pocketAPITestURL is the url of the pocket widget in test environment.
	//pocketAPITestURL = "http://widget.staging.pocketbitcoin.com/widget_mjxWDmSUkMvdQdXDCeHrjC"

	// pocketAPILiveURL is the url of the pocket widget in production environment.
	pocketAPILiveURL = "http://widget.pocketbitcoin.com/widget_vqx25E6kzvGBYGjN2QoXVH"
)

// BuyPocket verifies if the passed account is enabled for Pocket and returns the
// url needed to incorporate the widget in the frontend.
func BuyPocket(acct accounts.Interface) (string, error) {
	if acct.Coin().Code() != coin.CodeBTC {
		return "", fmt.Errorf("unsupported cryptocurrency code %q", acct.Coin().Code())
	}

	// Address signing is only available for mainnet accounts on firmware, we can't use test URL at the moment.
	apiURL := pocketAPILiveURL
	return apiURL, nil
}

// IsPocketSupported is true if coin.Code is supported by Pocket.
func IsPocketSupported(acct accounts.Interface) bool {
	// Pocket would also support tbtc, but at the moment testnet address signing is disabled on firmware.
	return acct.Coin().Code() == coin.CodeBTC
}
