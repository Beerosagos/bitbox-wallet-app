// Copyright 2018 Shift Devices AG
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

package lightning

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ArkLabsHQ/fulmine/pkg/swap"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
	"github.com/sirupsen/logrus"
)

// PostLightningActivateNode handles the POST request to activate the lightning node.
func (lightning *Lightning) PostLightningActivateNode(r *http.Request) interface{} {
	if err := lightning.Activate(); err != nil {
		lightning.log.Error(err)
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	return responseDto{Success: true}
}

// PostLightningDeactivateNode handles the POST request to deactivate the lightning node.
func (lightning *Lightning) PostLightningDeactivateNode(r *http.Request) interface{} {
	if err := lightning.Deactivate(); err != nil {
		lightning.log.Error(err)
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	return responseDto{Success: true}
}

// GetNodeInfo handles the GET request to retrieve the node info.
func (lightning *Lightning) GetNodeInfo(_ *http.Request) interface{} {
	return responseDto{Success: false, ErrorMessage: "Lightning node info is not available without the Breez SDK"}
}

func (lightning *Lightning) GetBoardingAddress(_ *http.Request) interface{} {
	address, err := lightning.BoardingAddress()
	if err != nil {
		lightning.log.Error(err.Error())
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}
	return responseDto{Success: true, Data: address}
}

// GetBalance handles the GET request to retrieve the node balance and its fiat conversions.
func (lightning *Lightning) GetBalance(_ *http.Request) interface{} {
	balance, err := lightning.Balance()
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	btcCoin := lightning.btcCoin

	formattedAvailableAmount := coin.FormattedAmount{
		Amount:      btcCoin.FormatAmount(balance.Available(), false),
		Unit:        btcCoin.GetFormatUnit(false),
		Conversions: coin.Conversions(balance.Available(), btcCoin, false, lightning.ratesUpdater),
	}

	return responseDto{
		Success: true,
		Data: accounts.FormattedAccountBalance{
			HasAvailable: balance.Available().BigInt().Sign() > 0,
			Available:    formattedAvailableAmount,
			HasIncoming:  false,
			Incoming:     coin.FormattedAmount{},
		}}
}

type ArkTx struct {
	Amount uint64 `json:"amount"`
	Type   string `json:"type"`
	Date   string `json:"date"`
}

// GetListPayments handles the GET request to list payments.
func (lightning *Lightning) GetListPayments(r *http.Request) interface{} {
	if lightning.arkClient == nil {
		return responseDto{Success: false, ErrorMessage: "Ark client not initialized"}
	}
	client := *lightning.arkClient
	txs, err := client.GetTransactionHistory(context.Background())
	if err != nil {
		lightning.log.Error(err.Error())
		return responseDto{Success: false, ErrorMessage: "Failed to get Ark Transactions history"}
	}

	txData := []ArkTx{}

	for _, tx := range txs {
		txData = append(txData, ArkTx{
			Amount: tx.Amount,
			Type:   string(tx.Type),
			Date:   tx.CreatedAt.Format(time.RFC3339),
		})
	}

	// if lightning.sdkService == nil {
	// 	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	// }

	// getParams, err := toListPaymentsRequestDto(r.URL.Query())
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// listPaymentsRequest, err := toListPaymentsRequest(getParams)
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// paymentsResponse, err := lightning.sdkService.ListPayments(listPaymentsRequest)
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// payments, err := toPaymentsDto(paymentsResponse)
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }
	// return responseDto{Success: true, Data: payments}
	return responseDto{Success: true, Data: txData}
}

// GetOpenChannelFee handles the GET request fetch the open channel fees.
func (lightning *Lightning) GetOpenChannelFee(r *http.Request) interface{} {
	return responseDto{Success: false, ErrorMessage: "Open channel fees are not available without the Breez SDK"}
}

// GetParseInput handles the GET request to parse a text input.
func (lightning *Lightning) GetParseInput(r *http.Request) interface{} {
	paymentDto, err := parseInputToDto(r.URL.Query().Get("s"))
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	return responseDto{Success: true, Data: paymentDto}
}

// PostReceivePayment handles the POST request to receive a payment.
func (lightning *Lightning) PostReceivePayment(r *http.Request) interface{} {
	// if lightning.sdkService == nil {
	// 	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	// }

	if lightning.arkClient == nil || lightning.boltzApi == nil {
		return responseDto{Success: false, ErrorMessage: "LN not initialized"}
	}
	var jsonBody receivePaymentRequestDto
	if err := json.NewDecoder(r.Body).Decode(&jsonBody); err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	postSwap := func(swapData swap.Swap) error {
		logrus.Info("Ark payment received!!!!")
		return nil
	}
	swapDetails, err := lightning.swapHandler.GetInvoice(context.Background(), jsonBody.AmountMsat/1000, postSwap)

	// mnemonic := lightning.backendConfig.LightningConfig().Accounts[0].Mnemonic
	// entropy, err := bip39.EntropyFromMnemonic(mnemonic)
	// if err != nil {
	// 	return err
	// }
	// prvKey, _ := btcec.PrivKeyFromBytes(entropy)
	// preimage := make([]byte, 32)
	// if _, err := rand.Read(preimage); err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// buf := sha256.Sum256(preimage)

	// swap, err := lightning.boltzApi.CreateReverseSwap(boltz.CreateReverseSwapRequest{
	// 	From:           boltz.CurrencyBtc,
	// 	To:             boltz.CurrencyArk,
	// 	InvoiceAmount:  jsonBody.AmountMsat / 1000,
	// 	ClaimPublicKey: hex.EncodeToString(prvKey.PubKey().SerializeCompressed()),
	// 	PreimageHash:   hex.EncodeToString(buf[:]),
	// })
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	return responseDto{Success: true, Data: receivePaymentResponseDto{LnInvoice: lnInvoiceDto{Bolt11: swapDetails.Invoice}}}

	// receivePaymentResponse, err := lightning.sdkService.ReceivePayment(toReceivePaymentRequest(jsonBody))
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// return responseDto{Success: true, Data: toReceivePaymentResponseDto(receivePaymentResponse)}
}

func (lightning *Lightning) PostSettle(r *http.Request) interface{} {
	if lightning.arkClient == nil || lightning.boltzApi == nil {
		return responseDto{Success: false, ErrorMessage: "Ark not initialized"}
	}
	err := lightning.Settle()
	if err != nil {
		lightning.log.Error(err.Error())
		return responseDto{Success: false, ErrorMessage: "Settlment failed"}
	}

	return responseDto{Success: true}
}

func (lightning *Lightning) PostSendPayment(r *http.Request) interface{} {
	if lightning.arkClient == nil || lightning.boltzApi == nil {
		return responseDto{Success: false, ErrorMessage: "Ark not initialized"}
	}
	var jsonBody sendPaymentRequestDto
	if err := json.NewDecoder(r.Body).Decode(&jsonBody); err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}
	unilateralRefund := func(swapData swap.Swap) error { /* process unilaterl Refund */ return nil }

	swapDetails, err := lightning.swapHandler.PayInvoice(context.Background(), jsonBody.Bolt11, unilateralRefund)
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}
	lightning.log.Infof("LN invoice payment succeded! %v", swapDetails)
	return responseDto{Success: true, Data: nil}

	// if lightning.sdkService == nil {
	// 	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	// }

	// var jsonBody sendPaymentRequestDto
	// if err := json.NewDecoder(r.Body).Decode(&jsonBody); err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// invoice, err := breez_sdk.ParseInvoice(jsonBody.Bolt11)
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// nodeState, err := lightning.sdkService.NodeInfo()
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// amount := invoice.AmountMsat
	// if jsonBody.AmountMsat != nil {
	// 	amount = jsonBody.AmountMsat
	// }

	// if amount == nil {
	// 	return responseDto{Success: false, ErrorMessage: "No amount specified."}
	// }

	// if *amount > nodeState.ChannelsBalanceMsat {
	// 	return responseDto{Success: false, ErrorMessage: "The available funds are not enough to pay this invoice."}
	// }

	// sendPaymentResponse, err := lightning.sdkService.SendPayment(toSendPaymentRequest(jsonBody))
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// dto, err := toSendPaymentResponseDto(sendPaymentResponse)
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// return responseDto{Success: true, Data: dto}
}

// GetDiagnosticData handles the GET request to retrieve the SDK diagnostic data.
func (lightning *Lightning) GetDiagnosticData(_ *http.Request) interface{} {
	return responseDto{Success: false, ErrorMessage: "Diagnostic data is not available without the Breez SDK"}
}

// PostReportPaymentFailure handles the POST request to report a payment failure.
func (lightning *Lightning) PostReportPaymentFailure(r *http.Request) interface{} {
	return responseDto{Success: false, ErrorMessage: "Reporting payment failures is not available without the Breez SDK"}
}

// GetServiceHealthCheck handles the GET request to retrieve the SDK service health check.
func (lightning *Lightning) GetServiceHealthCheck(_ *http.Request) interface{} {
	return responseDto{Success: false, ErrorMessage: "Service health checks are not available without the Breez SDK"}
}
