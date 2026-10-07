import { customInstance } from '../../api/custom-instance';

export interface ClusterStatus {
  initialized: boolean;
  version: string;
}

let cachedClusterStatus: ClusterStatus | null = null;

export function setCachedClusterStatus(status: ClusterStatus | null) {
  cachedClusterStatus = status;
}

export async function fetchClusterStatus(): Promise<ClusterStatus> {
  try {
    return await customInstance<ClusterStatus>({ url: '/auth/status' });
  } catch {
    return { initialized: true, version: '1.3.0' };
  }
}

export async function getClusterStatus(forceRefresh = false): Promise<ClusterStatus> {
  if (!forceRefresh && cachedClusterStatus !== null) {
    return cachedClusterStatus;
  }
  const status = await fetchClusterStatus();
  cachedClusterStatus = status;
  return status;
}
