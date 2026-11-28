import { useQuery } from '@tanstack/react-query';
import { QueryKeys } from '@/lib/queryKeys';
import { bankService } from '@/services/api/bank.service';

/**
 * Seeded bank branch reference list for the bank capture UI. The list is a
 * frozen snapshot managed outside the product, so it never goes stale.
 */
export const useBanks = () => {
  return useQuery({
    queryKey: QueryKeys.banks.lists(),
    queryFn: () => bankService.listBanks(),
    staleTime: Infinity,
  });
};
