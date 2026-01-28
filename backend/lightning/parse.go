package lightning

import (
	"strings"

	"github.com/BitBoxSwiss/bitbox-wallet-app/util/errp"
	decodepay "github.com/nbd-wtf/ln-decodepay"
)

func parseInputToDto(input string) (interface{}, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, errp.New("Input missing")
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "lightning:") {
		trimmed = trimmed[len("lightning:"):]
		lower = strings.ToLower(strings.TrimSpace(trimmed))
	}

	if strings.HasPrefix(lower, "lnurl") {
		return nil, errp.New("LNURL is not supported")
	}
	if !strings.HasPrefix(lower, "ln") {
		return nil, errp.New("Unsupported input")
	}

	decoded, err := decodepay.Decodepay(trimmed)
	if err != nil {
		return nil, err
	}

	invoice, err := toLnInvoiceDtoFromBolt11(trimmed, decoded)
	if err != nil {
		return nil, err
	}

	type inputTypeBolt11Dto struct {
		Type    string       `json:"type"`
		Invoice lnInvoiceDto `json:"invoice"`
	}

	return inputTypeBolt11Dto{
		Type:    "bolt11",
		Invoice: invoice,
	}, nil
}

func toLnInvoiceDtoFromBolt11(bolt11 string, decoded decodepay.Bolt11) (lnInvoiceDto, error) {
	description := optionalString(decoded.Description)
	descriptionHash := optionalString(decoded.DescriptionHash)

	var amountMsat *uint64
	if decoded.MSatoshi > 0 {
		msat := uint64(decoded.MSatoshi)
		amountMsat = &msat
	}

	routingHints, err := toRouteHintsDtoFromDecodepay(decoded.Route)
	if err != nil {
		return lnInvoiceDto{}, err
	}

	return lnInvoiceDto{
		Bolt11:          bolt11,
		PayeePubkey:     decoded.Payee,
		PaymentHash:     decoded.PaymentHash,
		Description:     description,
		DescriptionHash: descriptionHash,
		AmountMsat:      amountMsat,
		Timestamp:       uint64(decoded.CreatedAt),
		Expiry:          uint64(decoded.Expiry),
		RoutingHints:    routingHints,
		PaymentSecret:   nil,
	}, nil
}

func toRouteHintsDtoFromDecodepay(routes [][]decodepay.Hop) ([]routeHintDto, error) {
	if len(routes) == 0 {
		return []routeHintDto{}, nil
	}

	list := make([]routeHintDto, 0, len(routes))
	for _, route := range routes {
		hops := make([]routeHintHopDto, 0, len(route))
		for _, hop := range route {
			if hop.FeeBaseMsat < 0 || hop.FeeProportionalMillionths < 0 || hop.CLTVExpiryDelta < 0 {
				return nil, errp.New("Invalid route hint")
			}
			hops = append(hops, routeHintHopDto{
				SrcNodeId:                  hop.PubKey,
				ShortChannelId:             hop.ShortChannelId,
				FeesBaseMsat:               uint32(hop.FeeBaseMsat),
				FeesProportionalMillionths: uint32(hop.FeeProportionalMillionths),
				CltvExpiryDelta:            uint64(hop.CLTVExpiryDelta),
				HtlcMinimumMsat:            nil,
				HtlcMaximumMsat:            nil,
			})
		}
		list = append(list, routeHintDto{Hops: hops})
	}

	return list, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	copied := value
	return &copied
}
