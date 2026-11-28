import { apiClient } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import { Bank, BanksListResponse } from '@/types/api/bank.types';

class BankService {
  /**
   * Seeded bank branch reference list (read-only; no pagination).
   */
  async listBanks(): Promise<Bank[]> {
    const response = await apiClient.get<BanksListResponse>(
      API_ENDPOINTS.banks.base
    );
    const body = response.data as BanksListResponse | undefined;
    return body?.banks ?? [];
  }
}

export const bankService = new BankService();
