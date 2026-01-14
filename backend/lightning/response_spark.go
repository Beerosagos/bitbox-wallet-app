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
	Description          *string                `json:"Description,omitempty"`
	Preimage             *string                `json:"Preimage,omitempty"`
	Invoice              string                 `json:"Invoice,omitempty"`
	PaymentHash          string                 `json:"PaymentHash,omitempty"`
	DestinationPubkey    string                 `json:"DestinationPubkey,omitempty"`
	LnurlPayInfo         *lnurlPayInfoDto       `json:"LnurlPayInfo,omitempty"`
	LnurlWithdrawInfo    *lnurlWithdrawInfoDto  `json:"LnurlWithdrawInfo,omitempty"`
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
	PaymentHash string                         `json:"PaymentHash,omitempty"`
	Preimage    *string                        `json:"Preimage,omitempty"`
	ExpiryTime  uint64                         `json:"ExpiryTime,omitempty"`
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
