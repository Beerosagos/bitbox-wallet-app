// SPDX-License-Identifier: Apache-2.0

import { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { TAmountWithConversions } from '@/api/account';
import {
  TInputType,
  TInputTypeVariant,
  TLightningInvoice,
  TPreparePaymentResponse,
  TSdkError,
  getParsePaymentInput,
  postPreparePayment,
  postSendPayment,
} from '@/api/lightning';
import { getBtcSatsAmount } from '@/api/coins';
import { AmountWithUnit } from '@/components/amount/amount-with-unit';
import { Button, Input } from '@/components/forms';
import { Column, Grid, GuideWrapper, GuidedContent, Header, Main } from '@/components/layout';
import { Skeleton } from '@/components/skeleton/skeleton';
import { Spinner } from '@/components/spinner/Spinner';
import { Status } from '@/components/status/status';
import { View, ViewButtons, ViewContent, ViewHeader } from '@/components/view/view';
import { runningInAndroid, runningInIOS } from '@/utils/env';
import { SimpleMarkup } from '@/utils/markup';
import { useNavigate } from 'react-router-dom';
import { ScanQRVideo } from '../../account/send/components/inputs/scan-qr-video';
import styles from './send.module.css';

type TStep = 'select-invoice' | 'edit-invoice' | 'preparing' | 'confirm' | 'sending' | 'success';

const SendingSpinner = () => {
  const { t } = useTranslation();
  const [message, setStep] = useState<string>(t('lightning.send.sending.connecting'));

  setTimeout(() => {
    setStep(t('lightning.send.sending.message'));
  }, 1000);

  return <Spinner text={message} />;
};

const PreparingSpinner = () => {
  const { t } = useTranslation();
  return <Spinner text={t('loading')} />;
};

type TAmountValueProps = {
  sats: number;
  showFiat?: boolean;
};

const AmountValue = ({ sats, showFiat = false }: TAmountValueProps) => {
  const [amount, setAmount] = useState<TAmountWithConversions>();

  useEffect(() => {
    getBtcSatsAmount(`${sats}`).then((response) => {
      if (response.success) {
        setAmount(response.amount);
      }
    });
  }, [sats]);

  if (!amount) {
    return <Skeleton />;
  }

  return (
    <span className={styles.amountLine}>
      <AmountWithUnit amount={amount} alwaysShowAmounts />
      {showFiat && (
        <>
          {' / '}
          <AmountWithUnit amount={amount} alwaysShowAmounts convertToFiat />
        </>
      )}
    </span>
  );
};

type TInvoiceConfirmProps = {
  invoice: TLightningInvoice;
  quote: TPreparePaymentResponse;
};

const InvoiceConfirm = ({ invoice, quote }: TInvoiceConfirmProps) => {
  const { t } = useTranslation();

  return (
    <>
      <h1 className={styles.title}>{t('lightning.send.confirm.title')}</h1>
      <div className={styles.info}>
        <h2 className={styles.label}>{t('lightning.send.confirm.amount')}</h2>
        <AmountValue sats={quote.amountSat} showFiat />
      </div>
      {invoice.description && (
        <div className={styles.info}>
          <h2 className={styles.label}>{t('lightning.send.confirm.memo')}</h2>
          {invoice.description}
        </div>
      )}
      <div className={styles.info}>
        <h2 className={styles.label}>{t('send.fee.label')}</h2>
        <AmountValue sats={quote.feeSat} />
      </div>
      <div className={styles.info}>
        <h2 className={styles.label}>{t('send.confirm.total')}</h2>
        <AmountValue sats={quote.totalDebitSat} showFiat />
      </div>
    </>
  );
};

type TPaymentConfirmProps = {
  input: TInputType;
  quote: TPreparePaymentResponse;
};

const PaymentConfirm = ({ input, quote }: TPaymentConfirmProps) => {
  switch (input.type) {
  case TInputTypeVariant.BOLT11:
    return <InvoiceConfirm invoice={input.invoice} quote={quote} />;
  }
};

type TSendWorkflowProps = {
  customAmount?: number;
  inputError?: string;
  parsedInput?: TInputType;
  quote?: TPreparePaymentResponse;
  step: TStep;
  onBack: () => void;
  onCustomAmount: (input: number) => void;
  onInvoiceInput: (input: string) => void;
  onPrepare: () => void;
  onSend: () => void;
};

const SendWorkflow = ({
  customAmount,
  inputError,
  parsedInput,
  quote,
  step,
  onBack,
  onCustomAmount,
  onInvoiceInput,
  onPrepare,
  onSend,
}: TSendWorkflowProps) => {
  const { t } = useTranslation();
  const [lnInvoice, setLnInvoice] = useState('');

  const memoizedScanQRVideo = useMemo(() => (
    <ScanQRVideo onResult={onInvoiceInput} />
  ), [onInvoiceInput]);

  switch (step) {
  case 'select-invoice':
    return (
      <View textCenter width="660px">
        <ViewHeader title="Scan lightning invoice" />
        <ViewContent textAlign="center">
          <Grid col="1">
            <Column className={styles.camera}>
              <div className={styles.error}>
                {inputError && <Status dismissible="" type="warning">{inputError}</Status>}
              </div>
              {memoizedScanQRVideo}
              <Input
                placeholder={t('lightning.send.invoice.input')}
                onInput={(e: ChangeEvent<HTMLInputElement>) => setLnInvoice(e.target.value)}
                value={lnInvoice}
                autoFocus={!runningInAndroid() && !runningInIOS()}
              />
            </Column>
          </Grid>
        </ViewContent>
        <ViewButtons>
          <Button
            disabled={!lnInvoice}
            primary
            onClick={() => {
              onInvoiceInput(lnInvoice);
              setLnInvoice('');
            }}>
            {t('generic.send')}
          </Button>
          <Button secondary onClick={onBack}>
            {t('button.back')}
          </Button>
        </ViewButtons>
      </View>
    );
  case 'edit-invoice':
    if (parsedInput?.type !== TInputTypeVariant.BOLT11) {
      return (
        <View fitContent minHeight="100%">
          Invoices without amount are currently only supported for BOLT11 type invoices
        </View>
      );
    }
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
                onInput={e => onCustomAmount(e.target.valueAsNumber)}
                value={customAmount ? `${customAmount}` : ''}
                autoFocus
              />
              <Input
                type="text"
                label={t('lightning.receive.description.label')}
                placeholder="This invoice has no description"
                id="descriptionInput"
                readOnly
                disabled
                value={parsedInput.invoice.description}
              />
            </Column>
          </Grid>
        </ViewContent>
        <ViewButtons>
          <Button
            primary
            onClick={onPrepare}
            disabled={!customAmount}>
            {t('button.continue')}
          </Button>
          <Button secondary onClick={onBack}>
            {t('button.back')}
          </Button>
        </ViewButtons>
      </View>
    );
  case 'preparing':
    return <PreparingSpinner />;
  case 'confirm':
    if (!parsedInput || !quote) {
      return 'no invoice found';
    }
    return (
      <View fitContent minHeight="100%">
        <ViewContent>
          <Grid col="1">
            <Column>
              <PaymentConfirm input={parsedInput} quote={quote} />
            </Column>
          </Grid>
        </ViewContent>
        <ViewButtons>
          <Button primary onClick={onSend}>
            {t('generic.send')}
          </Button>
          <Button secondary onClick={onBack}>
            {t('button.back')}
          </Button>
        </ViewButtons>
      </View>
    );
  case 'sending':
    return <SendingSpinner />;
  case 'success':
    return (
      <View fitContent textCenter verticallyCentered>
        <ViewContent withIcon="success">
          <SimpleMarkup className={styles.successMessage} markup={t('lightning.send.success.message')} tagName="p" />
        </ViewContent>
      </View>
    );
  }
};

export const Send = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [step, setStep] = useState<TStep>('select-invoice');
  const [paymentInput, setPaymentInput] = useState<TInputType>();
  const [paymentQuote, setPaymentQuote] = useState<TPreparePaymentResponse>();
  const [customAmount, setCustomAmount] = useState<number>();
  const [rawInputError, setRawInputError] = useState<string>();
  const [sendError, setSendError] = useState<string>();

  const resetFlow = useCallback(() => {
    setStep('select-invoice');
    setSendError(undefined);
    setRawInputError(undefined);
    setPaymentInput(undefined);
    setPaymentQuote(undefined);
    setCustomAmount(undefined);
  }, []);

  const back = () => {
    switch (step) {
    case 'select-invoice':
      navigate('/lightning');
      break;
    case 'edit-invoice':
      resetFlow();
      break;
    case 'confirm':
      if (!paymentInput?.invoice.amountSat) {
        setPaymentQuote(undefined);
        setSendError(undefined);
        setStep('edit-invoice');
        break;
      }
      navigate('/lightning');
      break;
    case 'success':
      resetFlow();
      break;
    }
  };

  const preparePayment = useCallback(async (
    input: TInputType,
    maybeCustomAmount?: number,
    errorMessage?: string,
  ) => {
    setStep('preparing');
    setSendError(errorMessage);
    try {
      switch (input.type) {
      case TInputTypeVariant.BOLT11: {
        const quote = await postPreparePayment({
          bolt11: input.invoice.bolt11,
          amountSat: maybeCustomAmount || undefined,
        });
        setPaymentQuote(quote);
        setStep('confirm');
        break;
      }
      }
    } catch (e) {
      setStep('select-invoice');
      setPaymentQuote(undefined);
      if (e instanceof TSdkError) {
        setSendError(e.message);
      } else {
        setSendError(String(e));
      }
    }
  }, []);

  const parsePaymentInput = useCallback(async (rawInput: string) => {
    setRawInputError(undefined);
    setSendError(undefined);
    setPaymentQuote(undefined);
    try {
      const result = await getParsePaymentInput({ s: rawInput });
      switch (result.type) {
      case TInputTypeVariant.BOLT11:
        setPaymentInput(result);
        if (!result.invoice.amountSat) {
          setCustomAmount(0);
          setStep('edit-invoice');
          break;
        }
        setCustomAmount(undefined);
        void preparePayment(result);
        break;
      default:
        setRawInputError('Invalid input');
      }
    } catch (e) {
      if (e instanceof TSdkError) {
        setRawInputError(e.message);
      } else {
        setRawInputError(String(e));
      }
    }
  }, [preparePayment]);

  const sendPayment = async () => {
    if (!paymentInput || !paymentQuote) {
      return;
    }
    setStep('sending');
    setSendError(undefined);
    try {
      switch (paymentInput.type) {
      case TInputTypeVariant.BOLT11:
        await postSendPayment({
          bolt11: paymentInput.invoice.bolt11,
          amountSat: customAmount || undefined,
          approvedFeeSat: paymentQuote.feeSat,
        });
        setStep('success');
        setTimeout(() => navigate('/lightning'), 1000);
        break;
      }
    } catch (e) {
      if (e instanceof TSdkError && e.code === 'paymentApprovalRequired') {
        void preparePayment(paymentInput, customAmount, e.message);
        return;
      }
      setStep('select-invoice');
      setPaymentQuote(undefined);
      if (e instanceof TSdkError) {
        setSendError(e.message);
      } else {
        setSendError(String(e));
      }
    }
  };

  return (
    <GuideWrapper>
      <GuidedContent>
        <Main>
          <Status dismissible="" type="warning" hidden={!sendError}>
            {sendError}
          </Status>
          <Header title={<h2>{t('lightning.send.title')}</h2>} />
          <SendWorkflow
            customAmount={customAmount}
            inputError={rawInputError}
            parsedInput={paymentInput}
            quote={paymentQuote}
            step={step}
            onBack={back}
            onCustomAmount={setCustomAmount}
            onInvoiceInput={parsePaymentInput}
            onPrepare={() => paymentInput && void preparePayment(paymentInput, customAmount)}
            onSend={sendPayment}
          />
        </Main>
      </GuidedContent>
    </GuideWrapper>
  );
};
