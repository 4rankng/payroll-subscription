import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { FilterPill } from './FilterPill';

const options = [
  { value: 'active', label: 'Đang hoạt động' },
  { value: 'paused', label: 'Tạm dừng' },
];

describe('FilterPill', () => {
  it('không khóa con trỏ ngoài menu khi đang mở (modal=false)', () => {
    render(
      <>
        <FilterPill
          value="all"
          onChange={vi.fn()}
          placeholder="Vai trò"
          options={options}
        />
        <button type="button">Thêm người dùng</button>
      </>,
    );

    const trigger = screen.getByRole('button', { name: 'Vai trò' });
    // Radix DropdownMenu opens on pointerdown/Enter, not synthetic click.
    fireEvent.keyDown(trigger, { key: 'Enter' });
    expect(screen.getByRole('menu')).toBeInTheDocument();

    // Radix's modal layers set `pointer-events: none` on <body> while open,
    // which is what made the next outside click (e.g. "+ Thêm người dùng")
    // vanish into the dismissal. A non-modal pill must leave the pointer live.
    expect(document.body.style.pointerEvents).not.toBe('none');

    // Selecting an option still routes the change and closes the menu.
    fireEvent.click(screen.getByText('Đang hoạt động'));
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('hiển thị nhãn đang chọn khi giá trị khác mặc định', () => {
    render(
      <FilterPill
        value="active"
        onChange={vi.fn()}
        placeholder="Vai trò"
        options={options}
      />,
    );
    expect(screen.getByRole('button', { name: 'Vai trò' })).toHaveTextContent(
      'Đang hoạt động',
    );
  });
});
