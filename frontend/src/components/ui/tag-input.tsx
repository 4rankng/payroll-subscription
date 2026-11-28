import * as React from 'react';
import { X } from 'lucide-react';

import { cn } from '@/lib/utils';
import { Input } from '@/components/ui/input';

export interface TagInputProps {
  value: string[];
  onChange: (value: string[]) => void;
  placeholder?: string;
  id?: string;
  disabled?: boolean;
  /** Return false to reject a token. Defaults to accepting any non-empty token. */
  validateToken?: (token: string) => boolean;
  /** Message shown when a token is rejected. */
  invalidTokenMessage?: (token: string) => string;
  className?: string;
}

/** Split pasted/typed text into candidate tokens (comma, semicolon, whitespace). */
const splitTokens = (raw: string): string[] =>
  raw
    .split(/[,;\s]+/)
    .map((token) => token.trim())
    .filter((token) => token.length > 0);

/**
 * Chip/tag input for short string lists (emails To/CC). Tokens commit on
 * Enter/comma/semicolon/blur; paste accepts several at once; Backspace on an
 * empty field removes the last chip. Rejected tokens surface one error line
 * and are not added.
 */
export function TagInput({
  value,
  onChange,
  placeholder,
  id,
  disabled = false,
  validateToken,
  invalidTokenMessage,
  className,
}: TagInputProps) {
  const [draft, setDraft] = React.useState('');
  const [error, setError] = React.useState<string | null>(null);

  const commit = React.useCallback(
    (raw: string) => {
      const tokens = splitTokens(raw);
      if (tokens.length === 0) {
        setDraft('');
        return;
      }
      const accepted: string[] = [];
      let rejected: string | null = null;
      for (const token of tokens) {
        const isValid = validateToken ? validateToken(token) : true;
        const isDuplicate = value.some((existing) => existing === token) || accepted.includes(token);
        if (!isValid) {
          rejected ??= token;
          continue;
        }
        if (isDuplicate) continue;
        accepted.push(token);
      }
      setError(
        rejected !== null
          ? (invalidTokenMessage?.(rejected) ?? `"${rejected}" không hợp lệ`)
          : null,
      );
      setDraft('');
      if (accepted.length > 0) {
        onChange([...value, ...accepted]);
      }
    },
    [invalidTokenMessage, onChange, validateToken, value],
  );

  const removeTag = React.useCallback(
    (token: string) => {
      onChange(value.filter((existing) => existing !== token));
    },
    [onChange, value],
  );

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter' || event.key === ',' || event.key === ';') {
      event.preventDefault();
      commit(draft);
      return;
    }
    if (event.key === 'Backspace' && draft.length === 0 && value.length > 0) {
      event.preventDefault();
      removeTag(value[value.length - 1]);
    }
  };

  return (
    <div className={cn('space-y-2', className)}>
      <div
        className={cn(
          'flex min-h-10 w-full flex-wrap items-center gap-1.5 rounded-lg border border-input bg-card p-2 shadow-xs',
          'focus-within:outline focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-brand',
          error && 'border-utility-error-300',
          disabled && 'cursor-not-allowed opacity-50',
        )}
      >
        {value.map((token) => (
          <span
            key={token}
            className="inline-flex h-7 max-w-full items-center gap-1 rounded-md border border-border/60 bg-muted px-2 py-0.5 text-[13px] font-medium text-foreground"
          >
            <span className="truncate">{token}</span>
            <button
              type="button"
              disabled={disabled}
              onClick={() => removeTag(token)}
              aria-label={`Xoá ${token}`}
              className="relative -mr-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-sm text-muted-foreground transition-colors hover:text-destructive focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand"
            >
              <X className="h-3 w-3" aria-hidden="true" />
            </button>
          </span>
        ))}
        <Input
          id={id}
          type="text"
          value={draft}
          disabled={disabled}
          placeholder={value.length === 0 ? (placeholder ?? 'Nhập rồi nhấn Enter') : ''}
          onChange={(event) => {
            setDraft(event.target.value);
            setError(null);
          }}
          onKeyDown={handleKeyDown}
          onBlur={() => commit(draft)}
          className="h-7 min-h-7 w-full min-w-40 flex-1 border-0 bg-transparent px-0.5 text-[13px] shadow-none focus-visible:outline-none"
        />
      </div>
      {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  );
}
