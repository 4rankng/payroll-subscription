// Bank related types

export interface Bank {
  id: number;
  branch_name: string;
  created_at?: string;
  updated_at?: string;
}

/** Body of GET /banks — the read-only seeded branch reference list. */
export interface BanksListResponse {
  banks: Bank[];
}
