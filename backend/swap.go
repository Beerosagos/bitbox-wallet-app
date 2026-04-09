// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"slices"
	"sort"

	accountsTypes "github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts/types"
	coinpkg "github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/config"
	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
)

// SwapAccount contains the backend-native data needed to serialize swap accounts.
type SwapAccount struct {
	Keystore          config.Keystore
	KeystoreConnected bool
	AccountConfig     *config.Account
	AccountCoin       coinpkg.Coin
	ParentAccountCode *accountsTypes.Code
}

// SwapAccounts contains the sell and buy accounts needed by the swap screen.
type SwapAccounts struct {
	SellAccounts []SwapAccount
	BuyAccounts  []SwapAccount
}

// SwapAccounts returns the accounts that can be selected in the swap screen.
func (backend *Backend) SwapAccounts() (SwapAccounts, error) {
	sellAccounts, err := backend.swapSellAccounts()
	if err != nil {
		return SwapAccounts{}, err
	}
	buyAccounts, err := backend.SwapBuyAccounts()
	if err != nil {
		return SwapAccounts{}, err
	}
	return SwapAccounts{
		SellAccounts: sellAccounts,
		BuyAccounts:  buyAccounts,
	}, nil
}

// SwapBuyAccounts returns the accounts that can be selected as swap destinations.
func (backend *Backend) SwapBuyAccounts() ([]SwapAccount, error) {
	connectedKeystore, err := backend.connectedKeystoreConfig()
	if err != nil {
		return nil, err
	}
	if connectedKeystore == nil {
		return []SwapAccount{}, nil
	}

	swapAccounts := []SwapAccount{}
	persistedAccounts := backend.config.AccountsConfig()
	for _, persistedAccount := range persistedAccounts.Accounts {
		if persistedAccount.HiddenBecauseUnused {
			continue
		}
		if _, isTestnet := coinpkg.TestnetCoins[persistedAccount.CoinCode]; isTestnet != backend.Testing() {
			continue
		}

		rootFingerprint, err := persistedAccount.SigningConfigurations.RootFingerprint()
		if err != nil {
			backend.log.WithField("code", persistedAccount.Code).Error("could not identify root fingerprint")
			continue
		}
		if !slices.Equal(rootFingerprint, connectedKeystore.RootFingerprint) {
			continue
		}

		accountCoin, err := backend.Coin(persistedAccount.CoinCode)
		if err != nil {
			backend.log.WithField("code", persistedAccount.Code).WithError(err).Error("could not find coin")
			continue
		}

		swapAccounts = append(swapAccounts, SwapAccount{
			Keystore:          *connectedKeystore,
			KeystoreConnected: true,
			AccountConfig:     persistedAccount,
			AccountCoin:       accountCoin,
		})

		if persistedAccount.CoinCode != coinpkg.CodeETH {
			continue
		}
		swapAccounts = backend.appendERC20SwapDestinationAccounts(
			swapAccounts,
			*connectedKeystore,
			persistedAccount,
		)
	}

	sort.Slice(swapAccounts, func(i, j int) bool {
		return lessAccountSortOrder(
			swapAccounts[i].AccountCoin,
			swapAccounts[i].AccountConfig,
			swapAccounts[j].AccountCoin,
			swapAccounts[j].AccountConfig,
		)
	})

	return swapAccounts, nil
}

func (backend *Backend) connectedKeystoreConfig() (*config.Keystore, error) {
	persistedAccounts := backend.config.AccountsConfig()
	connectedKeystore := backend.Keystore()
	if connectedKeystore == nil {
		return nil, nil
	}
	connectedRootFingerprint, err := connectedKeystore.RootFingerprint()
	if err != nil {
		return nil, errp.Wrap(err, "could not retrieve rootFingerprint")
	}
	keystore, err := persistedAccounts.LookupKeystore(connectedRootFingerprint)
	if err != nil {
		return nil, errp.Wrap(err, "could not find connected keystore in config")
	}
	return keystore, nil
}

func (backend *Backend) swapSellAccounts() ([]SwapAccount, error) {
	connectedKeystore, err := backend.connectedKeystoreConfig()
	if err != nil {
		return nil, err
	}
	if connectedKeystore == nil {
		return []SwapAccount{}, nil
	}
	swapAccounts := []SwapAccount{}
	for _, account := range backend.Accounts() {
		accountConfig := account.Config().Config
		if accountConfig.Inactive || accountConfig.HiddenBecauseUnused {
			continue
		}

		rootFingerprint, err := accountConfig.SigningConfigurations.RootFingerprint()
		if err != nil {
			backend.log.WithField("code", accountConfig.Code).Error("could not identify root fingerprint")
			continue
		}
		if !slices.Equal(rootFingerprint, connectedKeystore.RootFingerprint) {
			continue
		}

		swapAccounts = append(swapAccounts, SwapAccount{
			Keystore:          *connectedKeystore,
			KeystoreConnected: true,
			AccountConfig:     accountConfig,
			AccountCoin:       account.Coin(),
		})
	}

	sort.Slice(swapAccounts, func(i, j int) bool {
		return lessAccountSortOrder(
			swapAccounts[i].AccountCoin,
			swapAccounts[i].AccountConfig,
			swapAccounts[j].AccountCoin,
			swapAccounts[j].AccountConfig,
		)
	})

	return swapAccounts, nil
}

// SignSwap prepares the selected destination before the real swap signing flow is implemented.
func (backend *Backend) SignSwap(buyAccountCode, sellAccountCode accountsTypes.Code, routeID, sellAmount string) error {
	_ = sellAccountCode
	_ = routeID
	_ = sellAmount
	swapAccounts, err := backend.SwapBuyAccounts()
	if err != nil {
		return err
	}
	for _, account := range swapAccounts {
		if account.AccountConfig.Code != buyAccountCode {
			continue
		}
		if account.ParentAccountCode != nil {
			return backend.SetTokenActive(*account.ParentAccountCode, string(account.AccountCoin.Code()), true)
		}
		if account.AccountConfig.Inactive {
			return backend.SetAccountActive(account.AccountConfig.Code, true)
		}

		// TODO implement swap sign logic here.
		return nil
	}
	return errp.Newf("Could not find swap destination account %s", buyAccountCode)
}

func (backend *Backend) appendERC20SwapDestinationAccounts(
	swapAccounts []SwapAccount,
	keystore config.Keystore,
	persistedAccount *config.Account,
) []SwapAccount {
	for _, token := range ERC20Tokens() {
		tokenCoin, err := backend.Coin(token.Code)
		if err != nil {
			backend.log.WithField("tokenCode", token.Code).WithError(err).Error("could not find ERC20 coin")
			continue
		}

		tokenAccountCode := Erc20AccountCode(persistedAccount.Code, string(token.Code))
		tokenName, err := configuredAccountName(tokenCoin, persistedAccount)
		if err != nil {
			backend.log.WithField("code", persistedAccount.Code).WithError(err).Error("could not get account number")
		}

		tokenConfig := &config.Account{
			Inactive:              !slices.Contains(persistedAccount.ActiveTokens, string(token.Code)),
			HiddenBecauseUnused:   persistedAccount.HiddenBecauseUnused,
			CoinCode:              token.Code,
			Name:                  tokenName,
			Code:                  tokenAccountCode,
			SigningConfigurations: persistedAccount.SigningConfigurations,
		}
		parentCode := persistedAccount.Code
		swapAccounts = append(swapAccounts, SwapAccount{
			Keystore:          keystore,
			KeystoreConnected: true,
			AccountConfig:     tokenConfig,
			AccountCoin:       tokenCoin,
			ParentAccountCode: &parentCode,
		})
	}
	return swapAccounts
}
