import { Download } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { downloadAssetFile } from '@/utils/file-download';

interface TransferFileLinkProps {
  assetId: number | null;
  filename: string;
  /** Typography for the surrounding surface (table cell vs mobile card title). */
  className?: string;
}

/**
 * Sổ cái file name (PAYSUB-24): downloads the customer's original transfer
 * result workbook. Uploads from before workbook storage render as plain text.
 */
export function TransferFileLink({ assetId, filename, className = '' }: TransferFileLinkProps) {
  if (assetId == null) {
    return (
      <span className={`block break-all ${className}`} title={filename}>
        {filename}
      </span>
    );
  }
  return (
    <Button
      variant="link"
      className={`min-h-11 max-w-full gap-1.5 p-0 sm:min-h-9 ${className}`}
      onClick={() => void downloadAssetFile(assetId, filename)}
      aria-label={`Tải xuống ${filename}`}
    >
      <Download className="size-3 shrink-0" aria-hidden="true" />
      <span className="break-all">{filename}</span>
    </Button>
  );
}
