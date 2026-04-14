// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { vi, describe, expect, beforeEach, it } from 'vitest';
import type { Fiat, TAccountBase } from '@/api/account';
import { RatesContext } from '@/contexts/RatesContext';
import { GroupedAccountSelector } from './groupedaccountselector';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('@/utils/config', () => ({
  getConfig: vi.fn().mockResolvedValue({ backend: { locale: 'en', proxy: { useProxy: false } } }),
  setConfig: vi.fn(),
}));

vi.mock('@/api/nativelocale', () => ({
  getNativeLocale: vi.fn().mockResolvedValue('en'),
}));

vi.mock('./services', async () => {
  const actual = await vi.importActual<typeof import('./services')>('./services');
  return {
    ...actual,
    getBalancesForGroupedAccountSelector: vi.fn().mockImplementation(async (groupedOptions) => groupedOptions),
  };
});

const makeMatchMedia = (matches: boolean) => vi.fn().mockImplementation(() => ({
  matches,
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
}));

const ratesContextValue = {
  defaultCurrency: 'USD' as const,
  activeCurrencies: ['USD'] as Fiat[],
  rotateDefaultCurrency: vi.fn(),
  rotateBtcUnit: vi.fn(),
  addToActiveCurrencies: vi.fn(),
  updateDefaultCurrency: vi.fn(),
  removeFromActiveCurrencies: vi.fn(),
};

const accounts: TAccountBase[] = [
  {
    keystore: {
      connected: true,
      lastConnected: '',
      name: 'BitBox',
      rootFingerprint: 'abcd1234',
      watchonly: false,
    },
    active: true,
    coinCode: 'btc',
    coinUnit: 'BTC',
    code: 'btc-account',
    isToken: false,
    name: 'Bitcoin account',
  },
  {
    keystore: {
      connected: true,
      lastConnected: '',
      name: 'BitBox',
      rootFingerprint: 'abcd1234',
      watchonly: false,
    },
    active: true,
    coinCode: 'eth',
    coinUnit: 'ETH',
    code: 'eth-account',
    isToken: false,
    name: 'Ethereum account',
  },
];

describe('components/groupedaccountselector/groupedaccountselector', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: makeMatchMedia(false),
    });
  });

  it('marks predicate-matched accounts as disabled', async () => {
    render(
      <RatesContext.Provider value={ratesContextValue}>
        <GroupedAccountSelector
          accounts={accounts}
          isAccountDisabled={({ coinCode }) => coinCode === 'btc'}
          onChange={vi.fn()}
          selected="eth-account"
        />
      </RatesContext.Provider>
    );

    fireEvent.keyDown(screen.getByRole('combobox'), { code: 'ArrowDown', key: 'ArrowDown' });

    const disabledOption = (await screen.findAllByText('Bitcoin account'))[0]?.closest('[aria-disabled]');
    const enabledOption = (await screen.findAllByText('Ethereum account'))[1]?.closest('[aria-disabled]');

    expect(disabledOption).toHaveAttribute('aria-disabled', 'true');
    expect(enabledOption).not.toHaveAttribute('aria-disabled', 'true');
  });
});
