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
	"encoding/json"
	"net/http"

	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/accounts"
	"github.com/BitBoxSwiss/bitbox-wallet-app/backend/coins/coin"
)

type responseDto struct {
	Success      bool        `json:"success"`
	Data         interface{} `json:"data"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
	ErrorCode    string      `json:"errorCode,omitempty"`
}

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
	// if lightning.sdkService == nil {
	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	// }

	// nodeState, err := lightning.sdkService.NodeInfo()
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// return responseDto{Success: true, Data: toNodeStateDto(nodeState)}
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

// GetListPayments handles the GET request to list payments.
func (lightning *Lightning) GetListPayments(r *http.Request) interface{} {
	if lightning.sdkService == nil {
		return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	}

	request, err := toSparkListPaymentsRequest(r.URL.Query())
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	payments, err := lightning.ListPayments(request)
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	paymentDtos, err := toSparkPaymentsDto(payments)
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	return responseDto{Success: true, Data: paymentDtos}
}

// GetOpenChannelFee handles the GET request fetch the open channel fees.
func (lightning *Lightning) GetOpenChannelFee(r *http.Request) interface{} {
	// if lightning.sdkService == nil {
	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	// }

	// getParams, err := toOpenChannelFeeRequestDto(r.URL.Query())
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// openChannelFeeResponse, err := lightning.sdkService.OpenChannelFee(toOpenChannelFeeRequest(getParams))
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }
	// return responseDto{Success: true, Data: toOpenChannelFeeResponseDto(openChannelFeeResponse)}
}

// GetParseInput handles the GET request to parse a text input.
func (lightning *Lightning) GetParseInput(r *http.Request) interface{} {
	if lightning.sdkService == nil {
		return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	}
	input, err := lightning.sdkService.Parse(r.URL.Query().Get("s"))
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	paymentDto, err := toInputTypeDto(input)
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	return responseDto{Success: true, Data: paymentDto}
}

type ReceivePaymentResponse struct {
	Invoice string `json:"invoice"`
	// Fee to pay to receive the payment
	// Denominated in sats or token base units
	Fee uint64 `json:"fee"`
}

// PostReceivePayment handles the POST request to receive a payment.
func (lightning *Lightning) PostReceivePayment(r *http.Request) interface{} {
	if lightning.sdkService == nil {
		return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	}

	var jsonBody receivePaymentRequestDto
	if err := json.NewDecoder(r.Body).Decode(&jsonBody); err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	receivePaymentResponse, err := lightning.ReceivePayment(jsonBody.AmountMsat/1000, jsonBody.Description)
	if err != nil {
		return responseDto{Success: false, ErrorMessage: err.Error()}
	}

	return responseDto{Success: true, Data: ReceivePaymentResponse{Invoice: receivePaymentResponse.PaymentRequest, Fee: receivePaymentResponse.Fee.Uint64()}}
}

// PostSendPayment handles the POST request to send a payment.
func (lightning *Lightning) PostSendPayment(r *http.Request) interface{} {
	// if lightning.sdkService == nil {
	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
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
	// if lightning.sdkService == nil {
	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	// }

	// diagnosticData, err := lightning.sdkService.GenerateDiagnosticData()
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// return responseDto{Success: true, Data: toDiagnosticDataDto(diagnosticData)}
}

// PostReportPaymentFailure handles the POST request to report a payment failure.
func (lightning *Lightning) PostReportPaymentFailure(r *http.Request) interface{} {
	// if lightning.sdkService == nil {
	return responseDto{Success: false, ErrorMessage: "BreezServices not initialized"}
	// }

	// var jsonBody reportPaymentFailureRequestDto
	// if err := json.NewDecoder(r.Body).Decode(&jsonBody); err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// err := lightning.sdkService.ReportIssue(toReportIssueRequest(jsonBody))
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// return responseDto{Success: true}
}

// GetServiceHealthCheck handles the GET request to retrieve the SDK service health check.
func (lightning *Lightning) GetServiceHealthCheck(_ *http.Request) interface{} {
	// breezApiKey, err := lightning.getBreezApiKey()
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// response, err := breez_sdk.ServiceHealthCheck(*breezApiKey)
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// dto, err := toServiceHealthCheckResponseDto(response)
	// if err != nil {
	// 	return responseDto{Success: false, ErrorMessage: err.Error()}
	// }

	// return responseDto{Success: true, Data: dto}
	return responseDto{Success: false, ErrorMessage: "health check not available"}
}
