import { useEffect, useState } from 'react';
import { getPipelineStats, type PipelineStats } from '../api/pipelineApi';

interface UsePipelineStatsOptions {
  window?: '7d' | '30d' | '60d' | '90d';
  enabled?: boolean;
  refetchInterval?: number; // ms
}

const CACHE_TTL = 5 * 60 * 1000; // 5 minutes
let cachedData: PipelineStats | null = null;
let cacheTime = 0;

export function usePipelineStats(options: UsePipelineStatsOptions = {}) {
  const { window = '30d', enabled = true, refetchInterval = 0 } = options;
  const [stats, setStats] = useState<PipelineStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!enabled) return;

    const fetchStats = async () => {
      const now = Date.now();

      // Use cache if available and fresh
      if (cachedData && now - cacheTime < CACHE_TTL) {
        setStats(cachedData);
        return;
      }

      setLoading(true);
      try {
        const data = await getPipelineStats(window);
        cachedData = data;
        cacheTime = now;
        setStats(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Failed to fetch stats'));
      } finally {
        setLoading(false);
      }
    };

    fetchStats();

    // Setup auto-refresh if interval is set
    if (refetchInterval > 0) {
      const interval = setInterval(fetchStats, refetchInterval);
      return () => clearInterval(interval);
    }
  }, [window, enabled, refetchInterval]);

  return { stats, loading, error };
}

// Manual cache invalidation
export function invalidatePipelineStatsCache() {
  cachedData = null;
  cacheTime = 0;
}
