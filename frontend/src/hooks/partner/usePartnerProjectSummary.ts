import { useQuery } from '@tanstack/react-query';
import { projectService } from '@/services/api/project.service';
import { authManager } from '@/lib/auth';

const QUERY_KEYS = {
  partnerSummary: ['projects', 'partner-summary'] as const,
};

export const usePartnerProjectSummary = () => {
  const userRole = authManager.getUserRole();
  // Mirror the useProjects gate: admin, quanly, giamdoc. ketoan's project
  // list is deliberately scoped out, so its summary stays off too — no new
  // data exposure beyond the visible list.
  const hasPermission =
    (userRole === 'admin' || userRole === 'quanly' || userRole === 'giamdoc') && userRole !== null;

  return useQuery({
    queryKey: QUERY_KEYS.partnerSummary,
    queryFn: () => projectService.getPartnerSummary(),
    enabled: hasPermission,
    retry: false, // Don't retry permission errors
    refetchOnWindowFocus: false,
  });
};