// Copyright 2023 Shift Crypto AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bitsurance

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/digitalbitbox/bitbox-wallet-app/backend/accounts"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/accounts/types"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/signing"
	"github.com/digitalbitbox/bitbox-wallet-app/backend/util"
	"github.com/digitalbitbox/bitbox-wallet-app/util/errp"
)

const (
	apiKey        = "265bfd773038c9ad9da7047e107babba0269bc3d31952172d9b10335e8a9d8e9"
	xpubSalt      = "bitsurance"
	apiURL        = "https://api.bitsurance.eu/api/"
	widgetVersion = "1"
	widgetURL     = "https://get.bitsurance.eu/?wallet=bitbox&version=" + widgetVersion + "&lang="
	widgetTestURL = "https://test.bitsurance.eu/?wallet=bitbox&version=" + widgetVersion + "&lang="
)

type DetailStatus string

const (
	ActiveStatus      DetailStatus = "active"
	ProcessingStatus  DetailStatus = "processing"
	RefusedStatus     DetailStatus = "refused"
	WaitPaymentStatus DetailStatus = "waitpayment"
	InactiveStatus    DetailStatus = "inactive"
	CanceledStatus    DetailStatus = "canceled"
)

type AccountDetails struct {
	AccountCode types.Code   `json:"code"`
	Status      DetailStatus `json:"status"`
	Details     struct {
		MaxCoverage int    `json:"maxCoverage"`
		Currency    string `json:"currency"`
		Support     string `json:"support"`
	} `json:"details"`
}

// bitsuranceCheckId fetches and returns the account details of the passed accountId from the
// Bitsurance server.
// If an accountId is not retrieved at all, the endpoint return a 404 http code.
func bitsuranceCheckId(httpClient *http.Client, accountId string) (AccountDetails, error) {
	endpoint := apiURL + "accountDetails/" + accountId
	details := AccountDetails{}
	code, err := util.APIGet(httpClient, endpoint, apiKey, 1024, &details)
	if err != nil && code != http.StatusNotFound {
		return details, err
	}
	return details, nil
}

// BitsuranceGetId returns the BitsuranceId of a given account.
// The id is computed hashing with sha256 the P2WPKH xpub of the account, concatenated with a fixed salt.
// If a P2WPKH xpub can't be found in the account, empty string is returned.
func BitsuranceGetId(account accounts.Interface) (string, error) {
	for _, signingConf := range account.Info().SigningConfigurations {
		bitcoinScriptType := signingConf.BitcoinSimple
		if bitcoinScriptType != nil && bitcoinScriptType.ScriptType == signing.ScriptTypeP2WPKH {
			hash := sha256.Sum256([]byte(bitcoinScriptType.KeyInfo.ExtendedPublicKey.String() + xpubSalt))
			return hex.EncodeToString(hash[:]), nil
		}
	}
	return "", errp.New("Unable to retrieve a bitsurance Id for the account: " + string(account.Config().Config.Code))

}

// BitsuranceURL returns the url for the Bitsurance widget for a given locale.
func BitsuranceURL(devServer bool, lang string) string {
	if devServer {
		return widgetTestURL + lang
	}
	return widgetURL + lang
}

// BitsuranceAccountsLookup takes in input a slice of accounts. For each account, it interrogates the
// Bitsurance server and returns a map with the given accounts' codes as keys and the insurance details as value.
func BitsuranceAccountsLookup(accounts []accounts.Interface, httpClient *http.Client) ([]AccountDetails, error) {
	insuredAccounts := []AccountDetails{}

	for _, account := range accounts {
		bitsuranceId, err := BitsuranceGetId(account)
		if err != nil {
			return nil, err
		}

		bitsuranceAccount, err := bitsuranceCheckId(httpClient, bitsuranceId)
		if err != nil {
			return nil, err
		}

		bitsuranceAccount.AccountCode = account.Config().Config.Code
		insuredAccounts = append(insuredAccounts, bitsuranceAccount)
	}

	return insuredAccounts, nil
}
