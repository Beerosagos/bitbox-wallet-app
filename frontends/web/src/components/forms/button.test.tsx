import { MemoryRouter } from 'react-router-dom';
import { render, screen } from '@testing-library/react';
import { ButtonLink, Button } from './button';

describe('components/forms/button', () => {
  describe('ButtonLink', () => {
    it('renders as button when disabled with disabled attribute', () => {
      render(<ButtonLink to="/settings" disabled>A ButtonLink</ButtonLink>);
      expect(screen.getByRole('button', { name: /A ButtonLink/i })).toHaveAttribute('disabled');
    });

    it('renders as link when not disabled', () => {
      render(<ButtonLink transparent to={'/settings'}>A ButtonLink</ButtonLink>, { wrapper: MemoryRouter });
      expect(screen.getByRole('link', { name: /A ButtonLink/i })).toBeInTheDocument();
    });
  });

  describe('Button', () => {
    it('renders a button with proper styling', () => {
      render(<Button primary>A Button</Button>);
      expect(screen.getByRole('button', { name: /A Button/i })).toHaveClass('primary');
    });
  });
});