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
import { getListPayments, subscribeListPayments, subscribeNodeState, getLightningBalance, TArkTx, getBoardingAddress } from '../../api/lightning';
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
import styles from './lightning.module.css';
import { RatesContext } from '@/contexts/RatesContext';

export const Lightning = () => {
  const { t } = useTranslation();
  const { btcUnit } = useContext(RatesContext);
  const [balance, setBalance] = useState<accountApi.IBalance>();
  const [syncedAddressesCount] = useState<number>();
  const [transactions, setTransactions] = useState<TArkTx[]>();
  const [boardingAddress, setBoardingAddress] = useState<string>();
  const [error, setError] = useState<string>();
  // const [detailID, setDetailID] = useState<accountApi.ITransaction['internalID'] | null>(null);

  const onStateChange = useCallback(async () => {
    try {
      setError(undefined);
      setBalance(await getLightningBalance());
      setTransactions(await getListPayments({}));
      setBoardingAddress(await getBoardingAddress());
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
              <>{boardingAddress && (<div>Boarding address: {boardingAddress}</div>)}</>
              {offlineErrorTextLines.length || !hasDataLoaded ? (
                <Spinner text={initializingSpinnerText} />
              ) : (
                transactions && transactions.length > 0 ? (
                  transactions
                    .map(tx => ({ // TODO: giant hack start
                      internalID: '666',
                      addresses: [],
                      amountAtTime: {
                        amount: tx.amount.toString(),
                        conversions: {}, // TODO: add conversions
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      deductedAmountAtTime: {
                        amount: tx.type === 'SENT' ? tx.amount.toString() : '',
                        conversions: {}, // TODO: add conversions
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      amount: {
                        amount: tx.amount.toString(),
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      fee: {
                        amount: '0',
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      feeRatePerKb: {
                        amount: '',
                        unit: 'sat' as accountApi.CoinUnit,
                        estimated: false
                      },
                      type: tx.type === 'RECEIVED' ? 'receive' as accountApi.TTransactionType : 'send', // TODO: add payment.paymentType 'closedChannel'
                      txID: '666',
                      note: '',
                      status: 'complete' as accountApi.TTransactionStatus,
                      time: tx.date, // TODO: remove hack?
                      // most of these are not for lightning
                      gas: 0,
                      nonce: null,
                      numConfirmationsComplete: 0,
                      size: 0,
                      numConfirmations: 0,
                      vsize: 0,
                      weight: 0
                    })) // TODO: giant hack end
                    .map((tx) => (
                      <Transaction
                        key={tx.internalID}
                        hideFiat
                        onShowDetail={() => {
                          // setDetailID(internalID);
                        }}
                        {...tx}
                      />
                    ))
                ) : (
                  <div className={`flex flex-row flex-center ${styles.empty}`}>
                    <p>{t('lightning.payments.placeholder')}</p>
                  </div>
                )
              )}

              { /*

              <PaymentDetails
                id={detailID}
                payment={transactions?.find(payment => payment.id === detailID)}
                onClose={() => setDetailID(null)}
              />

              */}
            </ViewContent>
          </View>
        </Main>
      </GuidedContent>
      <LightningGuide />
    </GuideWrapper>
  );
};
