// SPDX-License-Identifier: Apache-2.0

import '../../../../__mocks__/i18n';
import { act, renderHook } from '@testing-library/react';
import { createElement, type ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/api/lightning', () => {
  class TSdkError extends Error {
    code?: string;
    data?: unknown;

    constructor(message: string, code?: string, data?: unknown) {
      super(message);
      this.code = code;
      this.data = data;
    }
  }

  return {
    TPaymentInputTypeVariant: {
      BOLT11: 'bolt11',
    },
    TSdkError,
    getParsePaymentInput: vi.fn(),
    postPreparePayment: vi.fn(),
    postSendPayment: vi.fn(),
  };
});

import {
  TPaymentInputTypeVariant,
  TSdkError,
  getParsePaymentInput,
  postPreparePayment,
  postSendPayment,
} from '@/api/lightning';
import { LightningSendProvider, useLightningSendContext } from './lightning-send-context';

const amountlessInvoice = {
  type: TPaymentInputTypeVariant.BOLT11,
  invoice: {
    bolt11: 'lnbc1invoice',
    description: 'invoice description',
  },
};

const quote = (amountSat: number, feeSat: number) => ({
  amountSat,
  feeSat,
  totalDebitSat: amountSat + feeSat,
});

const deferred = <T>() => {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, resolve, reject };
};

type TProps = {
  children: ReactNode;
};

const wrapper = ({ children }: TProps) => (
  createElement(LightningSendProvider, null, children)
);

const renderContext = () => renderHook(() => useLightningSendContext(), { wrapper });

const flushPromises = async () => {
  await act(async () => {
    await Promise.resolve();
  });
};

describe('lightning send context', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.resetAllMocks();
    vi.mocked(getParsePaymentInput).mockResolvedValue(amountlessInvoice);
    vi.mocked(postSendPayment).mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not prepare custom invoices until the amount is valid', async () => {
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(0);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });

    expect(postPreparePayment).not.toHaveBeenCalled();
    expect(result.current.customPrepareState.status).toBe('idle');
  });

  it('debounces custom invoice prepare requests and stores the matching quote', async () => {
    vi.mocked(postPreparePayment).mockResolvedValue(quote(123, 4));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(299);
    });

    expect(postPreparePayment).not.toHaveBeenCalled();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    await flushPromises();

    expect(postPreparePayment).toHaveBeenCalledWith({
      bolt11: 'lnbc1invoice',
      amountSat: 123,
    });
    expect(result.current.customPrepareState).toEqual({
      status: 'success',
      amountSat: 123,
      quote: quote(123, 4),
    });
  });

  it('ignores stale custom invoice prepare responses', async () => {
    const firstPrepare = deferred<ReturnType<typeof quote>>();
    const secondPrepare = deferred<ReturnType<typeof quote>>();
    vi.mocked(postPreparePayment)
      .mockReturnValueOnce(firstPrepare.promise)
      .mockReturnValueOnce(secondPrepare.promise);
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(100);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    act(() => {
      result.current.setCustomAmount(200);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });

    await act(async () => {
      firstPrepare.resolve(quote(100, 1));
    });

    expect(result.current.customPrepareState).toEqual({
      status: 'loading',
      amountSat: 200,
    });

    await act(async () => {
      secondPrepare.resolve(quote(200, 2));
    });
    await flushPromises();

    expect(result.current.customPrepareState).toEqual({
      status: 'success',
      amountSat: 200,
      quote: quote(200, 2),
    });
  });

  it('stores custom invoice prepare errors inline', async () => {
    vi.mocked(postPreparePayment).mockRejectedValue(new TSdkError(
      'insufficient funds',
      'lightningInsufficientFunds',
      quote(123, 4)
    ));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    await flushPromises();

    expect(result.current.customPrepareState).toEqual({
      status: 'error',
      amountSat: 123,
      error: 'error.lightningInsufficientFunds',
      quote: quote(123, 4),
    });
    expect(result.current.sendError).toBeUndefined();

    await act(async () => {
      await result.current.sendPayment();
    });

    expect(postSendPayment).not.toHaveBeenCalled();
  });

  it('ignores malformed quote data in custom invoice prepare errors', async () => {
    vi.mocked(postPreparePayment).mockRejectedValue(new TSdkError(
      'insufficient funds',
      'lightningInsufficientFunds',
      { feeSat: 4 }
    ));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    await flushPromises();

    expect(result.current.customPrepareState).toEqual({
      status: 'error',
      amountSat: 123,
      error: 'error.lightningInsufficientFunds',
    });
  });

  it('sends custom invoices with the visible quote fee', async () => {
    vi.mocked(postPreparePayment).mockResolvedValue(quote(123, 4));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    await flushPromises();
    expect(result.current.customPrepareState.status).toBe('success');

    await act(async () => {
      await result.current.sendPayment();
    });

    expect(postSendPayment).toHaveBeenCalledWith({
      bolt11: 'lnbc1invoice',
      amountSat: 123,
      approvedFeeSat: 4,
    });
  });

  it('refreshes the custom quote in the edit step when fee approval is stale', async () => {
    vi.mocked(postPreparePayment)
      .mockResolvedValueOnce(quote(123, 4))
      .mockResolvedValueOnce(quote(123, 5));
    vi.mocked(postSendPayment).mockRejectedValue(new TSdkError('approval required', 'paymentApprovalRequired'));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    await flushPromises();
    expect(result.current.customPrepareState.status).toBe('success');

    await act(async () => {
      await result.current.sendPayment();
    });
    await flushPromises();

    expect(postPreparePayment).toHaveBeenCalledTimes(2);
    expect(result.current.customPrepareState).toEqual({
      status: 'success',
      amountSat: 123,
      quote: quote(123, 5),
    });
    expect(result.current.step).toBe('edit-invoice');
    expect(result.current.sendError).toBe('error.paymentApprovalRequired');
  });

  it('keeps custom invoice insufficient funds errors in the edit step after send', async () => {
    vi.mocked(postPreparePayment)
      .mockResolvedValueOnce(quote(123, 4))
      .mockRejectedValueOnce(new TSdkError(
        'insufficient funds',
        'lightningInsufficientFunds',
        quote(123, 4)
      ));
    vi.mocked(postSendPayment).mockRejectedValue(new TSdkError('insufficient funds', 'lightningInsufficientFunds'));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    await flushPromises();
    expect(result.current.customPrepareState.status).toBe('success');

    await act(async () => {
      await result.current.sendPayment();
    });
    await flushPromises();

    expect(postPreparePayment).toHaveBeenCalledTimes(2);
    expect(result.current.customPrepareState).toEqual({
      status: 'error',
      amountSat: 123,
      error: 'error.lightningInsufficientFunds',
      quote: quote(123, 4),
    });
    expect(result.current.step).toBe('edit-invoice');
    expect(result.current.sendError).toBeUndefined();
  });

  it('keeps generic custom invoice send errors in the edit step', async () => {
    vi.mocked(postPreparePayment).mockResolvedValue(quote(123, 4));
    vi.mocked(postSendPayment).mockRejectedValue(new TSdkError('duplicate operation'));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    await flushPromises();
    expect(result.current.customPrepareState.status).toBe('success');

    await act(async () => {
      await result.current.sendPayment();
    });

    expect(result.current.step).toBe('edit-invoice');
    expect(result.current.sendError).toBe('duplicate operation');
    expect(result.current.customPrepareState).toEqual({
      status: 'success',
      amountSat: 123,
      quote: quote(123, 4),
    });
  });

  it('returns already used custom invoices to invoice entry', async () => {
    vi.mocked(postPreparePayment).mockResolvedValue(quote(123, 4));
    vi.mocked(postSendPayment).mockRejectedValue(new TSdkError(
      'already used',
      'lightningInvoiceAlreadyUsed'
    ));
    const { result } = renderContext();

    await act(async () => {
      await result.current.parsePaymentInput('lnbc1invoice');
    });
    act(() => {
      result.current.setCustomAmount(123);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(300);
    });
    await flushPromises();
    expect(result.current.customPrepareState.status).toBe('success');

    await act(async () => {
      await result.current.sendPayment();
    });

    expect(result.current.step).toBe('select-invoice');
    expect(result.current.inputError).toBe('error.lightningInvoiceAlreadyUsed');
    expect(result.current.paymentDetails).toBeUndefined();
    expect(result.current.customAmount).toBeUndefined();
    expect(result.current.customPrepareState).toEqual({ status: 'idle' });
    expect(result.current.sendError).toBeUndefined();
  });
});
