import { z } from 'zod';
import type { UserRole } from '@/types/modal-config.types';

/**
 * General Domain Modal Registry
 * Contains all general-purpose modals with schemas and metadata
 */

export const GENERAL_MODAL_IDS = {
  FILE_UPLOAD: 'file_upload',
  NOTIFICATION_SHEET: 'notification_sheet',
} as const;

export type GeneralModalId = typeof GENERAL_MODAL_IDS[keyof typeof GENERAL_MODAL_IDS];

// Zod schemas for general modals
export const generalModalSchemas = {
  [GENERAL_MODAL_IDS.FILE_UPLOAD]: z.object({}),
  [GENERAL_MODAL_IDS.NOTIFICATION_SHEET]: z.object({}),
} as const;

// Modal metadata with permissions
interface GeneralModalMetadata {
  id: GeneralModalId;
  name: string;
  description: string;
  category: 'general' | 'report';
  requiresAuth: boolean;
  roles: UserRole[];
  componentPath: string;
  hasParams: boolean;
  schema?: keyof typeof generalModalSchemas;
  permissions: {
    action: 'read' | 'create' | 'update' | 'delete';
    subject: 'all' | 'Report' | 'PayrollData';
  };
}

export const generalModals: Record<GeneralModalId, GeneralModalMetadata> = {
  [GENERAL_MODAL_IDS.FILE_UPLOAD]: {
    id: GENERAL_MODAL_IDS.FILE_UPLOAD,
    name: 'Tải lên tập tin',
    description: 'Tải lên tập tin',
    category: 'general',
    requiresAuth: true,
    roles: ['admin', 'quanly'],
    componentPath: 'FileUploadModal',
    hasParams: false,
    permissions: {
      action: 'create',
      subject: 'all',
    },
  },
  [GENERAL_MODAL_IDS.NOTIFICATION_SHEET]: {
    id: GENERAL_MODAL_IDS.NOTIFICATION_SHEET,
    name: 'Thông báo',
    description: 'Xem thông báo',
    category: 'general',
    requiresAuth: true,
    roles: ['admin', 'quanly'],
    componentPath: 'NotificationSheet',
    hasParams: false,
    permissions: {
      action: 'read',
      subject: 'all',
    },
  },
} as const;
