import { useAsync } from 'react-use';

import { getGrafanaSearcher } from 'app/features/search/service/searcher';
import { type LocationInfo } from 'app/features/search/service/types';

/**
 *
 * @description Hook to fetch dashboard location info (folders).
 * @returns An object containing a mapping of folder UIDs to LocationInfo, loading state, and error state.
 */
export function useDashboardLocationInfo(enabled: boolean, folderUIDs?: string[]) {
  const searcher = getGrafanaSearcher();
  const {
    value: foldersByUid,
    loading,
    error,
  } = useAsync(async (): Promise<Record<string, LocationInfo>> => {
    if (!enabled) {
      return {};
    }
    return searcher.getLocationInfo(folderUIDs);
  }, [enabled, searcher, folderUIDs]);

  return {
    foldersByUid: foldersByUid ?? {},
    loading,
    error,
  };
}
