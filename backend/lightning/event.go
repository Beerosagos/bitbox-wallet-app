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
	"github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
	"github.com/sirupsen/logrus"
)

type sdkListener struct {
	log *logrus.Entry
}

func (l sdkListener) OnEvent(e breez_sdk_spark.SdkEvent) {
	switch event := e.(type) {
	case breez_sdk_spark.SdkEventSynced:
		// Wallet has been synchronized with the network
		l.log.Infof("Spark: Wallet has been synchronized with the network. Event: %v", e)
	case breez_sdk_spark.SdkEventUnclaimedDeposits:
		// SDK was unable to claim some deposits automatically
		unclaimedDeposits := event.UnclaimedDeposits
		_ = unclaimedDeposits
		l.log.Infof("Spark: unable to claim some deposit automatically. Event: %v", e)
	case breez_sdk_spark.SdkEventClaimedDeposits:
		// Deposits were successfully claimed
		claimedDeposits := event.ClaimedDeposits
		_ = claimedDeposits
		l.log.Infof("Spark: deposit successfully claimed. Event: %v", e)
	case breez_sdk_spark.SdkEventPaymentSucceeded:
		// A payment completed successfully
		payment := event.Payment
		_ = payment

		l.log.Infof("Spark: payment completed successfully. Event: %v", e)
	case breez_sdk_spark.SdkEventPaymentPending:
		// A payment is pending (waiting for confirmation)
		pendingPayment := event.Payment
		_ = pendingPayment
		l.log.Infof("Spark: payment waiting for confirmation. Event: %v", e)
	case breez_sdk_spark.SdkEventPaymentFailed:
		// A payment failed
		failedPayment := event.Payment
		_ = failedPayment
		l.log.Infof("Spark: payment failed. Event: %v", e)
	default:
		// Handle any future event types
		l.log.Infof("Spark event: %v", e)
	}
}

// // OnEvent receives an event from the sdk and handles it.
// // Implementation of breez_sdk.EventListener.
// func (lightning *Lightning) OnEvent(breezEvent breez_sdk.BreezEvent) {
// 	lightning.log.Infof("BreezSDK: %#v", breezEvent)

// 	switch breezEvent.(type) {
// 	case breez_sdk.BreezEventInvoicePaid, breez_sdk.BreezEventPaymentFailed, breez_sdk.BreezEventPaymentSucceed:
// 		lightning.Notify(observable.Event{
// 			Subject: "lightning/list-payments",
// 			Action:  action.Reload,
// 		})
// 	case breez_sdk.BreezEventSynced:
// 		lightning.Notify(observable.Event{
// 			Subject: "lightning/node-info",
// 			Action:  action.Reload,
// 		})
// 	}
// }
