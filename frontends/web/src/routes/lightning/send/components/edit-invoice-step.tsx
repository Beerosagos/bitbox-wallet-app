// SPDX-License-Identifier: Apache-2.0

import { ChangeEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { TPaymentInputTypeVariant } from '@/api/lightning';
import { Button, Input } from '@/components/forms';
import { Column, Grid } from '@/components/layout';
import { Status } from '@/components/status/status';
import { View, ViewButtons, ViewContent } from '@/components/view/view';
import { useLightningSendContext } from '../lightning-send-context';
import { PaymentFeeDetails } from './invoice-details';

export const EditInvoiceStep = () => {
  const { t } = useTranslation();
  const {
    customAmount,
    customPrepareState,
    paymentDetails,
    resetPayment,
    sendPayment,
    setCustomAmount,
  } = useLightningSendContext();

  if (paymentDetails?.type !== TPaymentInputTypeVariant.BOLT11) {
    return null;
  }

  const currentQuote = customPrepareState.status === 'success' && customPrepareState.amountSat === customAmount
    ? customPrepareState.quote
    : undefined;
  const errorQuote = customPrepareState.status === 'error' && customPrepareState.amountSat === customAmount
    ? customPrepareState.quote
    : undefined;
  const isPreparing = customPrepareState.status === 'loading' && customPrepareState.amountSat === customAmount;
  const prepareError = customPrepareState.status === 'error' && customPrepareState.amountSat === customAmount
    ? customPrepareState.error
    : undefined;
  const displayedQuote = currentQuote || errorQuote;
  const showQuote = isPreparing || displayedQuote;

  return (
    <View fitContent minHeight="100%">
      <ViewContent>
        <Grid col="1">
          <Column>
            <Input
              type="number"
              min="0"
              label={t('lightning.receive.amountSats.label')}
              placeholder={t('lightning.receive.amountSats.placeholder')}
              id="amountSatsInput"
              onInput={(event: ChangeEvent<HTMLInputElement>) => {
                const amount = event.target.valueAsNumber;
                setCustomAmount(Number.isNaN(amount) ? undefined : amount);
              }}
              value={customAmount ? `${customAmount}` : ''}
              autoFocus
            />
            <Input
              type="text"
              label={t('lightning.receive.description.label')}
              placeholder={t('lightning.receive.description.placeholder')}
              id="descriptionInput"
              readOnly
              disabled
              value={paymentDetails.invoice.description || ''}
            />
            <Status dismissibleKey="" type="error" hidden={!prepareError}>
              {prepareError}
            </Status>
            {showQuote && <PaymentFeeDetails quote={displayedQuote} />}
          </Column>
        </Grid>
      </ViewContent>
      <ViewButtons>
        <Button
          primary
          onClick={sendPayment}
          disabled={!currentQuote}>
          {t('generic.send')}
        </Button>
        <Button secondary onClick={resetPayment}>
          {t('button.back')}
        </Button>
      </ViewButtons>
    </View>
  );
};
