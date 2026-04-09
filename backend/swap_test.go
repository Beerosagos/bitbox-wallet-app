// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"slices"
	"testing"

	accountsTypes "github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts/types"
	coinpkg "github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
	"github.com/stretchr/testify/require"
)

func TestSwapDestinationAccountsRequireConnectedKeystore(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	allSwapAccounts, err := b.SwapAccounts()
	require.NoError(t, err)
	require.Len(t, allSwapAccounts.SellAccounts, 0)
	require.Len(t, allSwapAccounts.BuyAccounts, 0)
}

func TestSwapDestinationAccountsExcludeDisconnectedWatchonlyAccounts(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	ks := makeBitBox02Multi()
	rootFingerprint, err := ks.RootFingerprint()
	require.NoError(t, err)

	b.registerKeystore(ks)
	swapAccounts, err := b.SwapBuyAccounts()
	require.NoError(t, err)
	require.NotEmpty(t, swapAccounts)

	require.NoError(t, b.SetWatchonly(rootFingerprint, true))
	b.DeregisterKeystore()

	allSwapAccounts, err := b.SwapAccounts()
	require.NoError(t, err)
	require.Len(t, allSwapAccounts.SellAccounts, 0)
	require.Len(t, allSwapAccounts.BuyAccounts, 0)
}

func TestSwapAccountsSellAccountsExcludeInactiveAccounts(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	ks := makeBitBox02Multi()
	ks.RootFingerprintFunc = func() ([]byte, error) {
		return rootFingerprint1, nil
	}
	b.registerKeystore(ks)

	btcAccountCode := accountsTypes.Code("v0-55555555-btc-0")
	require.NoError(t, b.SetAccountActive(btcAccountCode, false))

	swapAccounts, err := b.SwapAccounts()
	require.NoError(t, err)
	sellAccountCodes := make([]accountsTypes.Code, 0, len(swapAccounts.SellAccounts))
	for _, account := range swapAccounts.SellAccounts {
		sellAccountCodes = append(sellAccountCodes, account.AccountConfig.Code)
	}
	require.NotContains(t, sellAccountCodes, btcAccountCode)

	buyAccountCodes := make([]accountsTypes.Code, 0, len(swapAccounts.BuyAccounts))
	for _, account := range swapAccounts.BuyAccounts {
		buyAccountCodes = append(buyAccountCodes, account.AccountConfig.Code)
	}
	require.Contains(t, buyAccountCodes, btcAccountCode)
}

func TestSwapBuyAccountsExcludeHiddenUnusedAccounts(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	ks := makeBitBox02Multi()
	ks.RootFingerprintFunc = func() ([]byte, error) {
		return rootFingerprint1, nil
	}
	b.registerKeystore(ks)

	btcAccountCode := accountsTypes.Code("v0-55555555-btc-0")
	cfg := b.Config().AccountsConfig().Lookup(btcAccountCode)
	cfg.HiddenBecauseUnused = true

	swapAccounts, err := b.SwapBuyAccounts()
	require.NoError(t, err)
	buyAccountCodes := make([]accountsTypes.Code, 0, len(swapAccounts))
	for _, account := range swapAccounts {
		buyAccountCodes = append(buyAccountCodes, account.AccountConfig.Code)
	}
	require.NotContains(t, buyAccountCodes, btcAccountCode)
}

func TestSwapDestinationAccountsSortOrder(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	ks := makeBitBox02Multi()
	ks.RootFingerprintFunc = func() ([]byte, error) {
		return rootFingerprint1, nil
	}

	b.registerKeystore(ks)

	btcAccount2Code, err := b.CreateAndPersistAccountConfig(coinpkg.CodeBTC, "Bitcoin account name 2", ks)
	require.NoError(t, err)

	ltcAccount2Code, err := b.CreateAndPersistAccountConfig(coinpkg.CodeLTC, "Litecoin account name 2", ks)
	require.NoError(t, err)

	ethAccount2Code, err := b.CreateAndPersistAccountConfig(coinpkg.CodeETH, "Ethereum account name 2", ks)
	require.NoError(t, err)

	tokenCodes := make([]string, 0, len(ERC20Tokens()))
	for _, token := range ERC20Tokens() {
		tokenCodes = append(tokenCodes, string(token.Code))
	}
	slices.Sort(tokenCodes)

	ethAccount1Code := accountsTypes.Code("v0-55555555-eth-0")
	expectedCodes := []accountsTypes.Code{
		"v0-55555555-btc-0",
		btcAccount2Code,
		"v0-55555555-ltc-0",
		ltcAccount2Code,
		ethAccount1Code,
	}
	for _, tokenCode := range tokenCodes {
		expectedCodes = append(expectedCodes, Erc20AccountCode(ethAccount1Code, tokenCode))
	}
	expectedCodes = append(expectedCodes, ethAccount2Code)
	for _, tokenCode := range tokenCodes {
		expectedCodes = append(expectedCodes, Erc20AccountCode(ethAccount2Code, tokenCode))
	}

	swapAccounts, err := b.SwapBuyAccounts()
	require.NoError(t, err)
	actualCodes := make([]accountsTypes.Code, 0, len(swapAccounts))
	for _, account := range swapAccounts {
		actualCodes = append(actualCodes, account.AccountConfig.Code)
	}

	require.Equal(t, expectedCodes, actualCodes)
}

func TestSignSwapActivatesInactiveAccount(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	ks := makeBitBox02Multi()
	ks.RootFingerprintFunc = func() ([]byte, error) {
		return rootFingerprint1, nil
	}
	b.registerKeystore(ks)

	btcAccountCode := accountsTypes.Code("v0-55555555-btc-0")
	require.NoError(t, b.SetAccountActive(btcAccountCode, false))
	require.True(t, b.Config().AccountsConfig().Lookup(btcAccountCode).Inactive)

	require.NoError(t, b.SignSwap(btcAccountCode, "", "", ""))
	require.False(t, b.Config().AccountsConfig().Lookup(btcAccountCode).Inactive)
}

func TestSignSwapActivatesParentOfTokenDestination(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	ks := makeBitBox02Multi()
	ks.RootFingerprintFunc = func() ([]byte, error) {
		return rootFingerprint1, nil
	}
	b.registerKeystore(ks)

	ethAccountCode := accountsTypes.Code("v0-55555555-eth-0")
	tokenCode := "eth-erc20-bat"
	tokenAccountCode := Erc20AccountCode(ethAccountCode, tokenCode)

	require.NoError(t, b.SetTokenActive(ethAccountCode, tokenCode, true))
	require.NoError(t, b.SetAccountActive(ethAccountCode, false))
	require.True(t, b.Config().AccountsConfig().Lookup(ethAccountCode).Inactive)
	require.Contains(t, b.Config().AccountsConfig().Lookup(ethAccountCode).ActiveTokens, tokenCode)

	require.NoError(t, b.SignSwap(tokenAccountCode, "", "", ""))
	require.False(t, b.Config().AccountsConfig().Lookup(ethAccountCode).Inactive)
	require.Contains(t, b.Config().AccountsConfig().Lookup(ethAccountCode).ActiveTokens, tokenCode)
}

func TestSignSwapReturnsErrorForUnknownAccount(t *testing.T) {
	b := newBackend(t, testnetDisabled, regtestDisabled)
	defer b.Close()

	err := b.SignSwap(accountsTypes.Code("missing-account"), "", "", "")
	require.Error(t, err)
}
