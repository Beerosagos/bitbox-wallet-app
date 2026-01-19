package lightning

import (
	"math/big"

	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
	"github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
)

type listPaymentsResponsePaymentDto struct {
	Id          string      `json:"Id"`
	PaymentType uint32      `json:"PaymentType"`
	Status      uint32      `json:"Status"`
	Amount      string      `json:"Amount"`
	Fees        string      `json:"Fees"`
	Timestamp   uint64      `json:"Timestamp"`
	Method      uint32      `json:"Method"`
	Details     interface{} `json:"Details,omitempty"`
}

type listPaymentsResponseLightningDetailsDto struct {
	Description          *string                  `json:"Description,omitempty"`
	Preimage             *string                  `json:"Preimage,omitempty"`
	Invoice              string                   `json:"Invoice,omitempty"`
	PaymentHash          string                   `json:"PaymentHash,omitempty"`
	DestinationPubkey    string                   `json:"DestinationPubkey,omitempty"`
	LnurlPayInfo         *lnurlPayInfoDto         `json:"LnurlPayInfo,omitempty"`
	LnurlWithdrawInfo    *lnurlWithdrawInfoDto    `json:"LnurlWithdrawInfo,omitempty"`
	LnurlReceiveMetadata *lnurlReceiveMetadataDto `json:"LnurlReceiveMetadata,omitempty"`
}

type listPaymentsResponseSparkDetailsDto struct {
	InvoiceDetails *sparkInvoicePaymentDetailsDto `json:"InvoiceDetails,omitempty"`
	HtlcDetails    *sparkHtlcDetailsDto           `json:"HtlcDetails,omitempty"`
}

type sparkInvoicePaymentDetailsDto struct {
	Description *string `json:"Description,omitempty"`
	Invoice     string  `json:"Invoice,omitempty"`
}

type sparkHtlcDetailsDto struct {
	PaymentHash string                          `json:"PaymentHash,omitempty"`
	Preimage    *string                         `json:"Preimage,omitempty"`
	ExpiryTime  uint64                          `json:"ExpiryTime,omitempty"`
	Status      breez_sdk_spark.SparkHtlcStatus `json:"Status,omitempty"`
}

type lnurlPayInfoDto struct {
	LnAddress              *string     `json:"lnAddress,omitempty"`
	Comment                *string     `json:"comment,omitempty"`
	Domain                 *string     `json:"domain,omitempty"`
	Metadata               *string     `json:"metadata,omitempty"`
	ProcessedSuccessAction interface{} `json:"processedSuccessAction,omitempty"`
}

type lnurlWithdrawInfoDto struct {
	WithdrawUrl string `json:"withdrawUrl"`
}

type lnurlReceiveMetadataDto struct {
	NostrZapRequest *string `json:"nostrZapRequest,omitempty"`
	NostrZapReceipt *string `json:"nostrZapReceipt,omitempty"`
	SenderComment   *string `json:"senderComment,omitempty"`
}

type sparkAddressDetailsDto struct {
	Address           string `json:"address"`
	IdentityPublicKey string `json:"identityPublicKey"`
	Network           string `json:"network"`
}

type sparkInvoiceDetailsDto struct {
	Invoice           string  `json:"invoice"`
	IdentityPublicKey string  `json:"identityPublicKey"`
	Network           string  `json:"network"`
	Amount            *string `json:"amount,omitempty"`
	TokenIdentifier   *string `json:"tokenIdentifier,omitempty"`
	ExpiryTime        *uint64 `json:"expiryTime,omitempty"`
	Description       *string `json:"description,omitempty"`
	SenderPublicKey   *string `json:"senderPublicKey,omitempty"`
}

type sparkBolt11InvoiceDto struct {
	Bolt11                  string         `json:"bolt11"`
	PayeePubkey             string         `json:"payeePubkey"`
	PaymentHash             string         `json:"paymentHash"`
	Description             *string        `json:"description"`
	DescriptionHash         *string        `json:"descriptionHash"`
	AmountMsat              *uint64        `json:"amountMsat"`
	Timestamp               uint64         `json:"timestamp"`
	Expiry                  uint64         `json:"expiry"`
	RoutingHints            []routeHintDto `json:"routingHints"`
	PaymentSecret           string         `json:"paymentSecret"`
	MinFinalCltvExpiryDelta uint64         `json:"minFinalCltvExpiryDelta"`
	Network                 string         `json:"network"`
}

func toSparkPaymentsDto(payments []breez_sdk_spark.Payment) ([]listPaymentsResponsePaymentDto, error) {
	list := make([]listPaymentsResponsePaymentDto, 0, len(payments))

	for _, payment := range payments {
		dto, err := toSparkPaymentDto(payment)
		if err != nil {
			return nil, err
		}
		list = append(list, dto)
	}

	return list, nil
}

func toSparkPaymentDto(payment breez_sdk_spark.Payment) (listPaymentsResponsePaymentDto, error) {
	details, err := toSparkPaymentDetailsDto(payment.Details)
	if err != nil {
		return listPaymentsResponsePaymentDto{}, err
	}

	return listPaymentsResponsePaymentDto{
		Id:          payment.Id,
		PaymentType: uint32(payment.PaymentType),
		Status:      uint32(payment.Status),
		Amount:      toBigIntString(payment.Amount),
		Fees:        toBigIntString(payment.Fees),
		Timestamp:   payment.Timestamp,
		Method:      uint32(payment.Method),
		Details:     details,
	}, nil
}

func toSparkPaymentDetailsDto(details *breez_sdk_spark.PaymentDetails) (interface{}, error) {
	if details == nil {
		return nil, nil
	}

	switch typed := (*details).(type) {
	case breez_sdk_spark.PaymentDetailsLightning:
		return toSparkLightningDetailsDto(typed)
	case breez_sdk_spark.PaymentDetailsSpark:
		return toSparkDetailsDto(typed), nil
	case breez_sdk_spark.PaymentDetailsToken:
		return typed, nil
	case breez_sdk_spark.PaymentDetailsWithdraw:
		return struct {
			TxId string `json:"TxId"`
		}{TxId: typed.TxId}, nil
	case breez_sdk_spark.PaymentDetailsDeposit:
		return struct {
			TxId string `json:"TxId"`
		}{TxId: typed.TxId}, nil
	}

	return nil, errp.New("Invalid PaymentDetails")
}

func toSparkLightningDetailsDto(details breez_sdk_spark.PaymentDetailsLightning) (listPaymentsResponseLightningDetailsDto, error) {
	lnurlPayInfo, err := toSparkLnurlPayInfoDto(details.LnurlPayInfo)
	if err != nil {
		return listPaymentsResponseLightningDetailsDto{}, err
	}

	return listPaymentsResponseLightningDetailsDto{
		Description:          details.Description,
		Preimage:             details.Preimage,
		Invoice:              details.Invoice,
		PaymentHash:          details.PaymentHash,
		DestinationPubkey:    details.DestinationPubkey,
		LnurlPayInfo:         lnurlPayInfo,
		LnurlWithdrawInfo:    toSparkLnurlWithdrawInfoDto(details.LnurlWithdrawInfo),
		LnurlReceiveMetadata: toSparkLnurlReceiveMetadataDto(details.LnurlReceiveMetadata),
	}, nil
}

func toSparkDetailsDto(details breez_sdk_spark.PaymentDetailsSpark) listPaymentsResponseSparkDetailsDto {
	var invoiceDetails *sparkInvoicePaymentDetailsDto
	if details.InvoiceDetails != nil {
		invoiceDetails = &sparkInvoicePaymentDetailsDto{
			Description: details.InvoiceDetails.Description,
			Invoice:     details.InvoiceDetails.Invoice,
		}
	}

	var htlcDetails *sparkHtlcDetailsDto
	if details.HtlcDetails != nil {
		htlcDetails = &sparkHtlcDetailsDto{
			PaymentHash: details.HtlcDetails.PaymentHash,
			Preimage:    details.HtlcDetails.Preimage,
			ExpiryTime:  details.HtlcDetails.ExpiryTime,
			Status:      details.HtlcDetails.Status,
		}
	}

	return listPaymentsResponseSparkDetailsDto{
		InvoiceDetails: invoiceDetails,
		HtlcDetails:    htlcDetails,
	}
}

func toSparkLnurlPayInfoDto(info *breez_sdk_spark.LnurlPayInfo) (*lnurlPayInfoDto, error) {
	if info == nil {
		return nil, nil
	}

	processedSuccessAction, err := toSparkSuccessActionProcessedDto(info.ProcessedSuccessAction)
	if err != nil {
		return nil, err
	}

	return &lnurlPayInfoDto{
		LnAddress:              info.LnAddress,
		Comment:                info.Comment,
		Domain:                 info.Domain,
		Metadata:               info.Metadata,
		ProcessedSuccessAction: processedSuccessAction,
	}, nil
}

func toSparkLnurlWithdrawInfoDto(info *breez_sdk_spark.LnurlWithdrawInfo) *lnurlWithdrawInfoDto {
	if info == nil {
		return nil
	}

	return &lnurlWithdrawInfoDto{
		WithdrawUrl: info.WithdrawUrl,
	}
}

func toSparkLnurlReceiveMetadataDto(info *breez_sdk_spark.LnurlReceiveMetadata) *lnurlReceiveMetadataDto {
	if info == nil {
		return nil
	}

	return &lnurlReceiveMetadataDto{
		NostrZapRequest: info.NostrZapRequest,
		NostrZapReceipt: info.NostrZapReceipt,
		SenderComment:   info.SenderComment,
	}
}

func toInputTypeDto(inputType breez_sdk_spark.InputType) (interface{}, error) {
	switch typed := inputType.(type) {
	case breez_sdk_spark.InputTypeBitcoinAddress:
		type inputTypeBitcoinAddressDto struct {
			Type    string                `json:"type"`
			Address bitcoinAddressDataDto `json:"address"`
		}
		bitcoinAddressData, err := toSparkBitcoinAddressDetailsDto(typed.Field0)
		if err != nil {
			return nil, err
		}
		return inputTypeBitcoinAddressDto{Type: "bitcoinAddress", Address: bitcoinAddressData}, nil
	case breez_sdk_spark.InputTypeBolt11Invoice:
		type inputTypeBolt11Dto struct {
			Type    string                `json:"type"`
			Invoice sparkBolt11InvoiceDto `json:"invoice"`
		}
		invoice, err := toSparkBolt11InvoiceDto(typed.Field0)
		if err != nil {
			return nil, err
		}
		return inputTypeBolt11Dto{Type: "bolt11", Invoice: invoice}, nil
	case breez_sdk_spark.InputTypeLnurlPay:
		type inputTypeLnUrlPayDto struct {
			Type string                 `json:"type"`
			Data lnUrlPayRequestDataDto `json:"data"`
		}
		return inputTypeLnUrlPayDto{Type: "lnUrlPay", Data: toSparkLnurlPayRequestDataDto(typed.Field0)}, nil
	case breez_sdk_spark.InputTypeLnurlWithdraw:
		type inputTypeLnUrlWithdrawDto struct {
			Type string                      `json:"type"`
			Data lnUrlWithdrawRequestDataDto `json:"data"`
		}
		return inputTypeLnUrlWithdrawDto{Type: "lnUrlWithdraw", Data: toSparkLnurlWithdrawRequestDataDto(typed.Field0)}, nil
	case breez_sdk_spark.InputTypeSparkAddress:
		type inputTypeSparkAddressDto struct {
			Type string                 `json:"type"`
			Data sparkAddressDetailsDto `json:"data"`
		}
		address, err := toSparkAddressDetailsDto(typed.Field0)
		if err != nil {
			return nil, err
		}
		return inputTypeSparkAddressDto{Type: "sparkAddress", Data: address}, nil
	case breez_sdk_spark.InputTypeSparkInvoice:
		type inputTypeSparkInvoiceDto struct {
			Type string                 `json:"type"`
			Data sparkInvoiceDetailsDto `json:"data"`
		}
		invoice, err := toSparkInvoiceDetailsDto(typed.Field0)
		if err != nil {
			return nil, err
		}
		return inputTypeSparkInvoiceDto{Type: "sparkInvoice", Data: invoice}, nil
	}

	return nil, errp.New("Invalid InputType")
}

func toSparkBitcoinAddressDetailsDto(details breez_sdk_spark.BitcoinAddressDetails) (bitcoinAddressDataDto, error) {
	network, err := toSparkBitcoinNetworkDto(details.Network)
	if err != nil {
		return bitcoinAddressDataDto{}, err
	}

	return bitcoinAddressDataDto{
		Address:   details.Address,
		Network:   network,
		AmountSat: nil,
		Label:     nil,
		Message:   nil,
	}, nil
}

func toSparkBolt11InvoiceDto(details breez_sdk_spark.Bolt11InvoiceDetails) (sparkBolt11InvoiceDto, error) {
	network, err := toSparkBitcoinNetworkDto(details.Network)
	if err != nil {
		return sparkBolt11InvoiceDto{}, err
	}

	return sparkBolt11InvoiceDto{
		Bolt11:                  details.Invoice.Bolt11,
		PayeePubkey:             details.PayeePubkey,
		PaymentHash:             details.PaymentHash,
		Description:             details.Description,
		DescriptionHash:         details.DescriptionHash,
		AmountMsat:              details.AmountMsat,
		Timestamp:               details.Timestamp,
		Expiry:                  details.Expiry,
		RoutingHints:            toSparkRouteHintsDto(details.RoutingHints),
		PaymentSecret:           details.PaymentSecret,
		MinFinalCltvExpiryDelta: details.MinFinalCltvExpiryDelta,
		Network:                 network,
	}, nil
}

func toSparkRouteHintsDto(routeHints []breez_sdk_spark.Bolt11RouteHint) []routeHintDto {
	list := make([]routeHintDto, 0, len(routeHints))

	for _, routeHint := range routeHints {
		list = append(list, routeHintDto{
			Hops: toSparkRouteHintHopsDto(routeHint.Hops),
		})
	}

	return list
}

func toSparkRouteHintHopsDto(routeHintHops []breez_sdk_spark.Bolt11RouteHintHop) []routeHintHopDto {
	list := make([]routeHintHopDto, 0, len(routeHintHops))

	for _, routeHintHop := range routeHintHops {
		list = append(list, routeHintHopDto{
			SrcNodeId:                  routeHintHop.SrcNodeId,
			ShortChannelId:             routeHintHop.ShortChannelId,
			FeesBaseMsat:               routeHintHop.FeesBaseMsat,
			FeesProportionalMillionths: routeHintHop.FeesProportionalMillionths,
			CltvExpiryDelta:            uint64(routeHintHop.CltvExpiryDelta),
			HtlcMinimumMsat:            routeHintHop.HtlcMinimumMsat,
			HtlcMaximumMsat:            routeHintHop.HtlcMaximumMsat,
		})
	}

	return list
}

func toSparkLnurlPayRequestDataDto(details breez_sdk_spark.LnurlPayRequestDetails) lnUrlPayRequestDataDto {
	allowsNostr := false
	if details.AllowsNostr != nil {
		allowsNostr = *details.AllowsNostr
	}

	return lnUrlPayRequestDataDto{
		Callback:       details.Callback,
		MinSendable:    details.MinSendable,
		MaxSendable:    details.MaxSendable,
		MetadataStr:    details.MetadataStr,
		CommentAllowed: details.CommentAllowed,
		Domain:         details.Domain,
		AllowsNostr:    allowsNostr,
		NostrPubkey:    details.NostrPubkey,
		LnAddress:      details.Address,
	}
}

func toSparkLnurlWithdrawRequestDataDto(details breez_sdk_spark.LnurlWithdrawRequestDetails) lnUrlWithdrawRequestDataDto {
	return lnUrlWithdrawRequestDataDto{
		Callback:           details.Callback,
		K1:                 details.K1,
		DefaultDescription: details.DefaultDescription,
		MinWithdrawable:    details.MinWithdrawable,
		MaxWithdrawable:    details.MaxWithdrawable,
	}
}

func toSparkAddressDetailsDto(details breez_sdk_spark.SparkAddressDetails) (sparkAddressDetailsDto, error) {
	network, err := toSparkBitcoinNetworkDto(details.Network)
	if err != nil {
		return sparkAddressDetailsDto{}, err
	}

	return sparkAddressDetailsDto{
		Address:           details.Address,
		IdentityPublicKey: details.IdentityPublicKey,
		Network:           network,
	}, nil
}

func toSparkInvoiceDetailsDto(details breez_sdk_spark.SparkInvoiceDetails) (sparkInvoiceDetailsDto, error) {
	network, err := toSparkBitcoinNetworkDto(details.Network)
	if err != nil {
		return sparkInvoiceDetailsDto{}, err
	}

	var amount *string
	if details.Amount != nil {
		amountValue := toBigIntString(*details.Amount)
		amount = &amountValue
	}

	return sparkInvoiceDetailsDto{
		Invoice:           details.Invoice,
		IdentityPublicKey: details.IdentityPublicKey,
		Network:           network,
		Amount:            amount,
		TokenIdentifier:   details.TokenIdentifier,
		ExpiryTime:        details.ExpiryTime,
		Description:       details.Description,
		SenderPublicKey:   details.SenderPublicKey,
	}, nil
}

func toSparkBitcoinNetworkDto(network breez_sdk_spark.BitcoinNetwork) (string, error) {
	switch network {
	case breez_sdk_spark.BitcoinNetworkBitcoin:
		return "bitcoin", nil
	case breez_sdk_spark.BitcoinNetworkTestnet3, breez_sdk_spark.BitcoinNetworkTestnet4:
		return "testnet", nil
	case breez_sdk_spark.BitcoinNetworkSignet:
		return "signet", nil
	case breez_sdk_spark.BitcoinNetworkRegtest:
		return "regtest", nil
	}
	return "", errp.New("Invalid BitcoinNetwork")
}

func toSparkSuccessActionProcessedDto(successActionProcessed *breez_sdk_spark.SuccessActionProcessed) (interface{}, error) {
	if successActionProcessed == nil {
		return nil, nil
	}

	switch successActionType := (*successActionProcessed).(type) {
	case breez_sdk_spark.SuccessActionProcessedAes:
		switch resultType := successActionType.Result.(type) {
		case breez_sdk_spark.AesSuccessActionDataResultDecrypted:
			return &aesSuccessActionResultDto{
				Type: "aes",
				Result: typeDataDto{
					Type: "decrypted",
					Data: aesSuccessActionDataDecryptedDto{
						Description: resultType.Data.Description,
						Plaintext:   resultType.Data.Plaintext,
					},
				},
			}, nil
		case breez_sdk_spark.AesSuccessActionDataResultErrorStatus:
			return &aesSuccessActionResultDto{
				Type: "aes",
				Result: aesSuccessActionResultErrorDto{
					Type:   "errorStatus",
					Reason: resultType.Reason,
				},
			}, nil
		}
	case breez_sdk_spark.SuccessActionProcessedMessage:
		return &typeDataDto{Type: "message", Data: messageSuccessActionDataDto{
			Message: successActionType.Data.Message,
		}}, nil
	case breez_sdk_spark.SuccessActionProcessedUrl:
		return &typeDataDto{Type: "url", Data: urlSuccessActionDataDecryptedDto{
			Description: successActionType.Data.Description,
			Url:         successActionType.Data.Url,
		}}, nil
	}

	return nil, errp.New("Invalid SuccessActionProcessed")
}

func toBigIntString(value *big.Int) string {
	if value == nil {
		return "0"
	}
	return value.String()
}
