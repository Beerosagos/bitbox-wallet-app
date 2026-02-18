/**
 * Copyright 2018 Shift Devices AG
 * Copyright 2022-2024 Shift Crypto AG
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useCallback, useContext, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import * as accountApi from '../../api/account';
import {
  Payment as IPayment,
  PaymentStatus,
  PaymentType,
  getLightningBalance,
  getListPayments,
  subscribeListPayments,
  subscribeNodeState,
  getBoardingAddress,
} from '../../api/lightning';
import { Balance } from '../../components/balance/balance';
import { ContentWrapper } from '@/components/contentwrapper/contentwrapper';
import { View, ViewContent, ViewHeader } from '../../components/view/view';
import { GuideWrapper, GuidedContent, Header, Main } from '../../components/layout';
import { Spinner } from '../../components/spinner/Spinner';
import { ActionButtons } from './components/action-buttons';
import { LightningGuide } from './guide';
import { unsubscribe } from '../../utils/subscriptions';
import { GlobalBanners } from '@/components/banners';
import { Status } from '../../components/status/status';
import { HideAmountsButton } from '../../components/hideamountsbutton/hideamountsbutton';
import { Transaction } from '@/components/transactions/transaction';
import { PaymentDetails } from './components/payment-details';
import styles from './lightning.module.css';
import { RatesContext } from '@/contexts/RatesContext';
import { useLoad } from '@/hooks/api';

export const Lightning = () => {
  const { t } = useTranslation();
  const { btcUnit } = useContext(RatesContext);
  const [balance, setBalance] = useState<accountApi.IBalance>();
  const [syncedAddressesCount] = useState<number>();
  const [payments, setPayments] = useState<IPayment[]>();
  const [error, setError] = useState<string>();
  const [detailID, setDetailID] = useState<accountApi.ITransaction['internalID'] | null>(null);
  const boardingAddress = useLoad(getBoardingAddress);

  const onStateChange = useCallback(async () => {
    try {
      setError(undefined);
      setBalance(await getLightningBalance());
      setPayments(await getListPayments({}));
    } catch (err: any) {
      const errorMessage = err?.errorMessage || err;
      setError(errorMessage);
    }
  }, []);

  useEffect(() => {
    onStateChange();

    const subscriptions = [subscribeNodeState(onStateChange), subscribeListPayments(onStateChange)];
    return () => unsubscribe(subscriptions);
  }, [onStateChange, btcUnit]);

  const hasDataLoaded = balance !== undefined;

  if (error) {
    return (
      <View textCenter verticallyCentered>
        <ViewHeader title={t('unknownError', { errorMessage: error })} />
      </View>
    );
  }

  if (!balance) {
    // Wait for the nodeState to become available
    return <Spinner />;
  }

  const canSend = balance && balance.hasAvailable;

  const initializingSpinnerText =
    syncedAddressesCount !== undefined && syncedAddressesCount > 1
      ? '\n' +
        t('account.syncedAddressesCount', {
          count: syncedAddressesCount.toString(),
          defaultValue: 0
        } as any)
      : '';

  const offlineErrorTextLines: string[] = [];
  const toTxStatus = (status: PaymentStatus): accountApi.TTransactionStatus => {
    switch (status) {
    case PaymentStatus.COMPLETED:
      return 'complete';
    case PaymentStatus.FAILED:
      return 'failed';
    default:
      return 'pending';
    }
  };

  const toTxType = (paymentType: PaymentType): accountApi.TTransactionType =>
    paymentType === PaymentType.RECEIVE ? 'receive' : 'send';

  return (
    <GuideWrapper>
      <GuidedContent>
        <Main>
          <ContentWrapper>
            <Status
              dismissible="lightning-alpha-warning"
              type="warning">
              This is an alpha release intended for preview and testing. Only use lightning with a small amount of funds!
            </Status>
            <GlobalBanners />
          </ContentWrapper>
          <Header
            title={
              <h2>
                <span>{t('lightning.accountLabel')}</span>
              </h2>
            }
          >
            <HideAmountsButton />
          </Header>
          <View>
            <ViewHeader>
              <div className={styles.header}>
                <Balance balance={balance} />
                <ActionButtons canSend={canSend} />
              </div>
            </ViewHeader>
            <ViewContent fullWidth>
              { <>Boarding address: {boardingAddress?.address ?? ''}</> }
              {offlineErrorTextLines.length || !hasDataLoaded ? (
                <Spinner text={initializingSpinnerText} />
              ) : (
                payments && payments.length > 0 ? (
                  payments
                    .map(payment => ({ // TODO: giant hack start
                      internalID: payment.id,
                      addresses: [],
                      amountAtTime: {
                        amount: payment.amountSat.toString(),
                        conversions: {}, // TODO: add conversions
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      deductedAmountAtTime: {
                        amount: payment.paymentType === PaymentType.SEND ? (payment.amountSat + payment.feesSat).toString() : '',
                        conversions: {}, // TODO: add conversions
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      amount: {
                        amount: payment.amountSat.toString(),
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      fee: {
                        amount: payment.feesSat.toString(),
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      feeRatePerKb: {
                        amount: '',
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      type: toTxType(payment.paymentType), // TODO: add payment.paymentType 'closedChannel'
                      txID: payment.id,
                      note: payment.description || '',
                      status: toTxStatus(payment.status),
                      time: payment.timestamp ? new Date(payment.timestamp * 1000).toString() : null, // TODO: remove hack?
                      // most of these are not for lightning
                      gas: 0,
                      nonce: null,
                      numConfirmationsComplete: payment.status === PaymentStatus.PENDING ? 1 : 0,
                      size: 0,
                      numConfirmations: 0,
                      vsize: 0,
                      weight: 0
                    })) // TODO: giant hack end
                    .map((payment) => (
                      <Transaction
                        key={payment.internalID}
                        hideFiat
                        onShowDetail={(internalID: string) => {
                          setDetailID(internalID);
                        }}
                        {...payment}
                      />
                    ))
                ) : (
                  <div className={`flex flex-row flex-center ${styles.empty}`}>
                    <p>{t('lightning.payments.placeholder')}</p>
                  </div>
                )
              )}

              <PaymentDetails
                id={detailID}
                payment={payments?.find(payment => payment.id === detailID)}
                onClose={() => setDetailID(null)}
              />
            </ViewContent>
          </View>
        </Main>
      </GuidedContent>
      <LightningGuide />
    </GuideWrapper>
  );
};
