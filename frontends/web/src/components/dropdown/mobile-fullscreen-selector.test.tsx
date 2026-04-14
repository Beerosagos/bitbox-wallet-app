// SPDX-License-Identifier: Apache-2.0

import { fireEvent, render, screen } from '@testing-library/react';
import { vi, describe, expect, it } from 'vitest';
import { MobileFullscreenSelector } from './mobile-fullscreen-selector';

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

vi.mock('@/hooks/backbutton', () => ({
  UseBackButton: () => null,
}));

describe('components/dropdown/mobile-fullscreen-selector', () => {
  it('does not select disabled options', () => {
    const onSelect = vi.fn();

    render(
      <MobileFullscreenSelector
        title="Accounts"
        options={[
          {
            connected: true,
            label: 'BitBox',
            options: [
              { disabled: true, label: 'Bitcoin account', value: 'btc-account' },
              { label: 'Ethereum account', value: 'eth-account' },
            ],
          },
        ]}
        renderOptions={(option) => <span>{option.label}</span>}
        renderGroupHeader={(group) => <span>{group.label}</span>}
        value={{ label: 'Ethereum account', value: 'eth-account' }}
        onSelect={onSelect}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Ethereum account' }));

    const disabledOption = screen.getByRole('button', { name: 'Bitcoin account' });
    expect(disabledOption).toBeDisabled();

    fireEvent.click(disabledOption);
    expect(onSelect).not.toHaveBeenCalled();
  });
});
