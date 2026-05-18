// SPDX-License-Identifier: Apache-2.0

import { ReactNode, createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  type TPaymentInputType,
  TPaymentInputTypeVariant,
  type TPreparePaymentResponse,
  TSdkError,
  getParsePaymentInput,
  postPreparePayment,
  postSendPayment,
} from '@/api/lightning';
import { useDebounce } from '@/hooks/debounce';
import { useMountedRef } from '@/hooks/mount';

type TSendStep = 'select-invoice' | 'edit-invoice' | 'preparing' | 'confirm' | 'sending' | 'success';

type TCustomPrepareState =
  | { status: 'idle' }
  | { status: 'loading'; amountSat: number }
  | { status: 'success'; amountSat: number; quote: TPreparePaymentResponse }
  | { status: 'error'; amountSat: number; error: string; quote?: TPreparePaymentResponse };

type TLightningSendContext = {
  customAmount?: number;
  customPrepareState: TCustomPrepareState;
  inputError?: string;
  paymentDetails?: TPaymentInputType;
  paymentQuote?: TPreparePaymentResponse;
  resetPayment: () => void;
  sendError?: string;
  sendPayment: () => Promise<void>;
  setCustomAmount: (amount?: number) => void;
  step: TSendStep;
  parsePaymentInput: (rawInput: string) => Promise<boolean>;
};

const isValidCustomAmount = (amount?: number): amount is number => (
  typeof amount === 'number' && Number.isFinite(amount) && Number.isInteger(amount) && amount > 0
);

const isPreparePaymentResponse = (data: unknown): data is TPreparePaymentResponse => {
  if (typeof data !== 'object' || data === null) {
    return false;
  }
  const response = data as Partial<TPreparePaymentResponse>;
  return (
    typeof response.amountSat === 'number'
    && typeof response.feeSat === 'number'
    && typeof response.totalDebitSat === 'number'
  );
};

const LightningSendContext = createContext<TLightningSendContext | null>(null);

type TProps = {
  children: ReactNode;
};

export const LightningSendProvider = ({ children }: TProps) => {
  const { t } = useTranslation();
  const mounted = useMountedRef();
  const customPrepareRequestID = useRef(0);
  const [step, setStep] = useState<TSendStep>('select-invoice');
  const [paymentDetails, setPaymentDetails] = useState<TPaymentInputType>();
  const [paymentQuote, setPaymentQuote] = useState<TPreparePaymentResponse>();
  const [customAmount, setCustomAmount] = useState<number>();
  const debouncedCustomAmount = useDebounce(customAmount, 300);
  const [customPrepareState, setCustomPrepareState] = useState<TCustomPrepareState>({ status: 'idle' });
  const [inputError, setInputError] = useState<string>();
  const [sendError, setSendError] = useState<string>();

  const toErrorMessage = useCallback((error: unknown): string => {
    if (error instanceof TSdkError) {
      if (error.code) {
        return t(`error.${error.code}`);
      }
      return error.message;
    }
    return String(error);
  }, [t]);

  const resetPayment = useCallback(() => {
    customPrepareRequestID.current += 1;
    setStep('select-invoice');
    setPaymentDetails(undefined);
    setPaymentQuote(undefined);
    setCustomAmount(undefined);
    setCustomPrepareState({ status: 'idle' });
    setInputError(undefined);
    setSendError(undefined);
  }, []);

  const updateCustomAmount = useCallback((amount?: number) => {
    customPrepareRequestID.current += 1;
    setCustomAmount(amount);
    setPaymentQuote(undefined);
    setCustomPrepareState({ status: 'idle' });
    setSendError(undefined);
  }, []);

  const prepareCustomPayment = useCallback(async (
    currentPaymentDetails: TPaymentInputType | undefined,
    amountSat: number | undefined,
    prepareErrorMessage?: string,
  ) => {
    if (
      currentPaymentDetails?.type !== TPaymentInputTypeVariant.BOLT11
      || currentPaymentDetails.invoice.amountSat
      || !isValidCustomAmount(amountSat)
    ) {
      return;
    }

    const requestID = ++customPrepareRequestID.current;
    setStep('edit-invoice');
    setCustomPrepareState({ status: 'loading', amountSat });
    setSendError(prepareErrorMessage);

    try {
      const quote = await postPreparePayment({
        bolt11: currentPaymentDetails.invoice.bolt11,
        amountSat,
      });
      if (!mounted.current || requestID !== customPrepareRequestID.current) {
        return;
      }
      setCustomPrepareState({ status: 'success', amountSat, quote });
    } catch (error) {
      if (!mounted.current || requestID !== customPrepareRequestID.current) {
        return;
      }
      const quote = error instanceof TSdkError && isPreparePaymentResponse(error.data) ? error.data : undefined;
      setCustomPrepareState(quote === undefined
        ? { status: 'error', amountSat, error: toErrorMessage(error) }
        : { status: 'error', amountSat, error: toErrorMessage(error), quote });
      setSendError(undefined);
    }
  }, [mounted, toErrorMessage]);

  const preparePayment = useCallback(async (
    nextPaymentDetails?: TPaymentInputType,
    prepareErrorMessage?: string,
  ) => {
    const currentPaymentDetails = nextPaymentDetails || paymentDetails;
    if (currentPaymentDetails?.type !== TPaymentInputTypeVariant.BOLT11) {
      return;
    }
    if (!currentPaymentDetails.invoice.amountSat) {
      return;
    }

    setStep('preparing');
    setSendError(prepareErrorMessage);

    try {
      const quote = await postPreparePayment({
        bolt11: currentPaymentDetails.invoice.bolt11,
      });
      setPaymentQuote(quote);
      setStep('confirm');
    } catch (error) {
      setStep('select-invoice');
      setPaymentQuote(undefined);
      setSendError(toErrorMessage(error));
    }
  }, [paymentDetails, toErrorMessage]);

  useEffect(() => {
    if (paymentDetails?.type !== TPaymentInputTypeVariant.BOLT11 || paymentDetails.invoice.amountSat) {
      return;
    }

    if (debouncedCustomAmount !== customAmount) {
      return;
    }

    if (!isValidCustomAmount(debouncedCustomAmount)) {
      customPrepareRequestID.current += 1;
      setPaymentQuote(undefined);
      setCustomPrepareState({ status: 'idle' });
      return;
    }

    prepareCustomPayment(paymentDetails, debouncedCustomAmount);
  }, [customAmount, debouncedCustomAmount, paymentDetails, prepareCustomPayment]);

  const parsePaymentInput = useCallback(async (rawInput: string) => {
    customPrepareRequestID.current += 1;
    setInputError(undefined);
    setSendError(undefined);
    setPaymentQuote(undefined);
    setCustomPrepareState({ status: 'idle' });

    try {
      const result = await getParsePaymentInput({ s: rawInput });
      setPaymentDetails(result);

      if (result.type === TPaymentInputTypeVariant.BOLT11 && !result.invoice.amountSat) {
        setCustomAmount(undefined);
        setStep('edit-invoice');
        return true;
      }

      setCustomAmount(undefined);
      preparePayment(result);
      return true;
    } catch (error) {
      setInputError(toErrorMessage(error));
      return false;
    }
  }, [preparePayment, toErrorMessage]);

  const sendPayment = useCallback(async () => {
    if (paymentDetails?.type !== TPaymentInputTypeVariant.BOLT11) {
      return;
    }

    const activePaymentQuote = paymentDetails.invoice.amountSat
      ? paymentQuote
      : customPrepareState.status === 'success' && customPrepareState.amountSat === customAmount
        ? customPrepareState.quote
        : undefined;

    if (!activePaymentQuote) {
      return;
    }

    if (!paymentDetails.invoice.amountSat && !isValidCustomAmount(customAmount)) {
      setSendError(t('send.error.invalidAmount'));
      return;
    }

    setStep('sending');
    setSendError(undefined);

    try {
      await postSendPayment({
        bolt11: paymentDetails.invoice.bolt11,
        amountSat: paymentDetails.invoice.amountSat ? undefined : customAmount,
        approvedFeeSat: activePaymentQuote.feeSat,
      });
      setStep('success');
    } catch (error) {
      const errorMessage = toErrorMessage(error);
      if (error instanceof TSdkError && error.code === 'lightningInvoiceAlreadyUsed') {
        customPrepareRequestID.current += 1;
        setStep('select-invoice');
        setPaymentDetails(undefined);
        setPaymentQuote(undefined);
        setCustomAmount(undefined);
        setCustomPrepareState({ status: 'idle' });
        setInputError(errorMessage);
        setSendError(undefined);
        return;
      }
      if (
        error instanceof TSdkError
        && !paymentDetails.invoice.amountSat
        && (error.code === 'paymentApprovalRequired' || error.code === 'lightningInsufficientFunds')
      ) {
        prepareCustomPayment(paymentDetails, customAmount, errorMessage);
        return;
      }
      if (error instanceof TSdkError && error.code === 'paymentApprovalRequired') {
        preparePayment(paymentDetails, errorMessage);
        return;
      }
      setStep(paymentDetails.invoice.amountSat ? 'confirm' : 'edit-invoice');
      setSendError(errorMessage);
    }
  }, [customAmount, customPrepareState, paymentDetails, paymentQuote, prepareCustomPayment, preparePayment, toErrorMessage, t]);

  return (
    <LightningSendContext.Provider value={{
      customAmount,
      customPrepareState,
      inputError,
      paymentDetails,
      paymentQuote,
      resetPayment,
      sendError,
      sendPayment,
      setCustomAmount: updateCustomAmount,
      step,
      parsePaymentInput,
    }}>
      {children}
    </LightningSendContext.Provider>
  );
};

export const useLightningSendContext = () => {
  const context = useContext(LightningSendContext);
  if (context === null) {
    throw new Error('useLightningSendContext must be used within LightningSendProvider');
  }
  return context;
};
