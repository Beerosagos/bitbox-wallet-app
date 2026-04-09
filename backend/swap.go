// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"context"
	"math/big"
	"slices"
	"sort"
	"strings"

	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts"
	accountsTypes "github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts/types"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/btc"
	btcaddresses "github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/btc/addresses"
	coinpkg "github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/eth"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/config"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/market/swapkit"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/paymentrequest"
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
	SellAccounts           []SwapAccount
	BuyAccounts            []SwapAccount
	DefaultSellAccountCode *accountsTypes.Code
	DefaultBuyAccountCode  *accountsTypes.Code
}

// SwapSignTxInput mirrors the existing frontend tx proposal input shape.
type SwapSignTxInput struct {
	Address        string                 `json:"address"`
	Amount         string                 `json:"amount"`
	UseHighestFee  bool                   `json:"useHighestFee"`
	SendAll        string                 `json:"sendAll"`
	SelectedUTXOS  []string               `json:"selectedUTXOS"`
	PaymentRequest *paymentrequest.Slip24 `json:"paymentRequest"`
}

// SwapPreparation contains everything the frontend needs to reuse the regular BTC send flow.
type SwapPreparation struct {
	ExpectedBuyAmount string          `json:"expectedBuyAmount"`
	SwapID            string          `json:"swapId"`
	TxInput           SwapSignTxInput `json:"txInput"`
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
	defaultSellAccount, defaultSellAccountCode := backend.swapDefaultSellAccount(sellAccounts)
	defaultBuyAccountCode := swapDefaultBuyAccount(buyAccounts, defaultSellAccount)
	return SwapAccounts{
		SellAccounts:           sellAccounts,
		BuyAccounts:            buyAccounts,
		DefaultSellAccountCode: defaultSellAccountCode,
		DefaultBuyAccountCode:  defaultBuyAccountCode,
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

func (backend *Backend) swapDefaultSellAccount(sellAccounts []SwapAccount) (*SwapAccount, *accountsTypes.Code) {
	for _, account := range sellAccounts {
		if account.AccountCoin.Code() != coinpkg.CodeETH {
			continue
		}
		if backend.accountHasNonZeroBalance(account.AccountConfig.Code) {
			return &account, &account.AccountConfig.Code
		}
	}
	for _, account := range sellAccounts {
		if account.AccountCoin.Code() == coinpkg.CodeBTC {
			continue
		}
		if backend.accountHasNonZeroBalance(account.AccountConfig.Code) {
			return &account, &account.AccountConfig.Code
		}
	}
	for _, account := range sellAccounts {
		if account.AccountCoin.Code() != coinpkg.CodeBTC {
			continue
		}
		if backend.accountHasNonZeroBalance(account.AccountConfig.Code) {
			return &account, &account.AccountConfig.Code
		}
	}
	if len(sellAccounts) == 0 {
		return nil, nil
	}
	return &sellAccounts[0], &sellAccounts[0].AccountConfig.Code
}

func swapDefaultBuyAccount(
	buyAccounts []SwapAccount,
	defaultSellAccount *SwapAccount,
) *accountsTypes.Code {
	if defaultSellAccount == nil {
		return nil
	}
	preferredBuyCoinCode := coinpkg.CodeBTC
	if defaultSellAccount.AccountCoin.Code() == coinpkg.CodeBTC {
		preferredBuyCoinCode = coinpkg.CodeETH
	}
	for _, account := range buyAccounts {
		if account.AccountCoin.Code() == preferredBuyCoinCode {
			return &account.AccountConfig.Code
		}
	}
	for _, account := range buyAccounts {
		if account.AccountConfig.Code == defaultSellAccount.AccountConfig.Code {
			continue
		}
		return &account.AccountConfig.Code
	}
	return nil
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

// PrepareSwap prepares a real SwapKit swap and returns a tx input that can be proposed and sent
// through the existing BTC payment-request flow.
func (backend *Backend) PrepareSwap(
	buyAccountCode, sellAccountCode accountsTypes.Code,
	routeID, sellAmount string,
) (*SwapPreparation, error) {
	if err := backend.activateSwapDestinationAccount(buyAccountCode); err != nil {
		return nil, err
	}

	sellAccount, err := backend.GetAccountFromCode(sellAccountCode)
	if err != nil {
		return nil, err
	}
	specificSellAccount, ok := sellAccount.(*btc.Account)
	if !ok {
		return nil, errp.New("Only BTC sell accounts are currently supported")
	}
	buyAccount, err := backend.GetAccountFromCode(buyAccountCode)
	if err != nil {
		return nil, err
	}
	specificBuyAccount, ok := buyAccount.(*eth.Account)
	if !ok {
		return nil, errp.New("Only ETH/ERC20 receive accounts are currently supported")
	}

	sourceAddress, err := swapSourceAddress(specificSellAccount)
	if err != nil {
		return nil, err
	}
	destinationAddress, destinationDerivation, _, err := swapDestinationAddress(specificBuyAccount)
	if err != nil {
		return nil, err
	}
	swapSellAmount, err := swapkit.FormatAmount(specificSellAccount.Coin(), sellAmount)
	if err != nil {
		return nil, err
	}

	swapResponse, swapError := swapkit.NewSwap(
		context.Background(),
		backend.httpClient,
		string(specificSellAccount.Coin().Code()),
		string(specificBuyAccount.Coin().Code()),
		swapSellAmount,
		routeID,
		sourceAddress,
		destinationAddress,
	)
	if swapError != nil {
		return nil, errp.New(swapError.Message)
	}
	if strings.TrimSpace(swapResponse.Memo) != "" {
		return nil, errp.New("Swap transaction memo is currently unsupported")
	}
	paymentRequest := swapResponse.PaymentRequest()
	if paymentRequest == nil {
		return nil, errp.New("Missing payment request")
	}
	if len(paymentRequest.Outputs) != 1 {
		return nil, errp.New("Missing or multiple payment request output unsupported")
	}
	if !slip24HasCoinPurchase(paymentRequest) {
		return nil, errp.New("Missing coinPurchase payment request memo")
	}
	txInput, err := swapSignTxInput(paymentRequest, specificSellAccount.Coin(), destinationDerivation)
	if err != nil {
		return nil, err
	}
	return &SwapPreparation{
		ExpectedBuyAmount: swapResponse.ExpectedBuyAmount,
		SwapID:            swapResponse.SwapID,
		TxInput:           txInput,
	}, nil
}

// SignSwap prepares the selected destination before the real swap signing flow is implemented.
func (backend *Backend) SignSwap(buyAccountCode, sellAccountCode accountsTypes.Code, routeID, sellAmount string) error {
	_ = sellAccountCode
	_ = routeID
	_ = sellAmount
	return backend.activateSwapDestinationAccount(buyAccountCode)
}

func (backend *Backend) activateSwapDestinationAccount(buyAccountCode accountsTypes.Code) error {
	account, err := backend.swapDestinationAccount(buyAccountCode)
	if err != nil {
		return err
	}
	if account.ParentAccountCode != nil {
		if parentAccount := backend.config.AccountsConfig().Lookup(*account.ParentAccountCode); parentAccount != nil && parentAccount.Inactive {
			if err := backend.SetAccountActive(*account.ParentAccountCode, true); err != nil {
				return err
			}
		}
		if account.AccountConfig.Inactive {
			if err := backend.SetTokenActive(*account.ParentAccountCode, string(account.AccountCoin.Code()), true); err != nil {
				return err
			}
		}
		return nil
	}
	if account.AccountConfig.Inactive {
		return backend.SetAccountActive(account.AccountConfig.Code, true)
	}
	return nil
}

func (backend *Backend) swapDestinationAccount(accountCode accountsTypes.Code) (*SwapAccount, error) {
	swapAccounts, err := backend.SwapBuyAccounts()
	if err != nil {
		return nil, err
	}
	for _, account := range swapAccounts {
		if account.AccountConfig.Code == accountCode {
			return &account, nil
		}
	}
	return nil, errp.Newf("Could not find swap destination account %s", accountCode)
}

func (backend *Backend) accountHasNonZeroBalance(accountCode accountsTypes.Code) bool {
	account := backend.Accounts().lookup(accountCode)
	if account == nil {
		return false
	}
	balance, err := account.Balance()
	if err != nil {
		backend.log.WithField("code", accountCode).WithError(err).Error("could not get account balance")
		return false
	}
	if balance == nil {
		return false
	}
	return balance.Available().BigInt().Sign() > 0
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

func slip24HasCoinPurchase(paymentRequest *paymentrequest.Slip24) bool {
	if paymentRequest == nil {
		return false
	}
	for _, memo := range paymentRequest.Memos {
		if memo.CoinPurchase != nil {
			return true
		}
	}
	return false
}

func frontendPaymentRequest(
	paymentRequest *paymentrequest.Slip24,
	destinationDerivation *paymentrequest.Slip24AddressDerivation,
) *paymentrequest.Slip24 {
	if paymentRequest == nil {
		return nil
	}
	memos := make([]paymentrequest.Slip24Memo, 0, len(paymentRequest.Memos))
	for _, memo := range paymentRequest.Memos {
		switch memo.Type {
		case "text":
			memos = append(memos, paymentrequest.Slip24Memo{
				Type: "text",
				Text: memo.Text,
			})
		case "coinPurchase":
			if memo.CoinPurchase == nil {
				continue
			}
			mappedMemo := paymentrequest.Slip24Memo{
				Type: "coinPurchase",
				CoinPurchase: &paymentrequest.Slip24CoinPurchase{
					CoinType: memo.CoinPurchase.CoinType,
					Amount:   memo.CoinPurchase.Amount,
					Address:  memo.CoinPurchase.Address,
				},
			}
			if destinationDerivation != nil {
				mappedMemo.CoinPurchase.AddressDerivation = destinationDerivation
			}
			memos = append(memos, mappedMemo)
		}
	}
	return &paymentrequest.Slip24{
		RecipientName: paymentRequest.RecipientName,
		Nonce:         paymentRequest.Nonce,
		Memos:         memos,
		Outputs:       paymentRequest.Outputs,
		Signature:     paymentRequest.Signature,
	}
}

func swapSignTxInput(
	paymentRequest *paymentrequest.Slip24,
	sellCoin coinpkg.Coin,
	destinationDerivation *paymentrequest.Slip24AddressDerivation,
) (SwapSignTxInput, error) {
	if paymentRequest == nil {
		return SwapSignTxInput{}, errp.New("Missing payment request")
	}
	if len(paymentRequest.Outputs) != 1 {
		return SwapSignTxInput{}, errp.New("Missing or multiple payment request output unsupported")
	}
	output := paymentRequest.Outputs[0]
	if strings.TrimSpace(output.Address) == "" {
		return SwapSignTxInput{}, errp.New("Missing target address")
	}
	amount := sellCoin.FormatAmount(coinpkg.NewAmount(new(big.Int).SetUint64(output.Amount)), false)
	return SwapSignTxInput{
		Address:        output.Address,
		Amount:         amount,
		UseHighestFee:  true,
		SendAll:        "no",
		SelectedUTXOS:  []string{},
		PaymentRequest: frontendPaymentRequest(paymentRequest, destinationDerivation),
	}, nil
}

func firstUnusedAddress(account accounts.Interface) (accounts.Address, error) {
	addressLists, err := account.GetUnusedReceiveAddresses()
	if err != nil {
		return nil, err
	}
	for _, addressList := range addressLists {
		if len(addressList.Addresses) == 0 {
			continue
		}
		return addressList.Addresses[0], nil
	}
	return nil, errp.New("Could not find an unused receive address")
}

func swapDestinationAddress(account *eth.Account) (string, *paymentrequest.Slip24AddressDerivation, accounts.Address, error) {
	address, err := firstUnusedAddress(account)
	if err != nil {
		return "", nil, nil, err
	}
	typedAddress, ok := address.(eth.Address)
	if !ok {
		return "", nil, nil, errp.New("Unsupported swap destination address type")
	}
	return typedAddress.EncodeForHumans(), &paymentrequest.Slip24AddressDerivation{
		Eth: &paymentrequest.Slip24EthAddressDerivation{
			Keypath: typedAddress.AbsoluteKeypath().ToUInt32(),
		},
	}, typedAddress, nil
}

func swapSourceAddress(account *btc.Account) (string, error) {
	spendableOutputs, err := account.SpendableOutputs()
	if err != nil {
		return "", err
	}
	for _, output := range spendableOutputs {
		if output.Address != nil {
			return output.Address.EncodeForHumans(), nil
		}
	}
	address, err := firstUnusedAddress(account)
	if err != nil {
		return "", err
	}
	switch typedAddress := address.(type) {
	case *btcaddresses.AccountAddress:
		return typedAddress.EncodeForHumans(), nil
	default:
		return "", errp.New("Unsupported swap source address type")
	}
}
