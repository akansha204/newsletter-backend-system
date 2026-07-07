import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createContext,
  PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';

import { loadApiKey, loadBaseUrl, saveApiKey, saveBaseUrl } from '@/lib/settings-storage';
import { NewsletterClient } from '@/lib/sdk';

interface ApiContextValue {
  ready: boolean;
  baseUrl: string | null;
  apiKey: string | null;
  client: NewsletterClient | null;
  hasApiKey: boolean;
  updateBaseUrl: (baseUrl: string) => Promise<void>;
  updateApiKey: (apiKey: string) => Promise<void>;
}

const ApiContext = createContext<ApiContextValue | null>(null);

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 5_000,
    },
  },
});

export function ApiProvider({ children }: PropsWithChildren) {
  const [ready, setReady] = useState(false);
  const [baseUrl, setBaseUrl] = useState<string | null>(null);
  const [apiKey, setApiKey] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([loadBaseUrl(), loadApiKey()])
      .then(([storedBaseUrl, storedApiKey]) => {
        if (cancelled) {
          return;
        }
        setBaseUrl(storedBaseUrl);
        setApiKey(storedApiKey);
      })
      .finally(() => {
        if (!cancelled) {
          setReady(true);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const updateBaseUrl = useCallback(async (value: string) => {
    await saveBaseUrl(value);
    setBaseUrl(value.trim() || null);
    queryClient.clear();
  }, []);

  const updateApiKey = useCallback(async (value: string) => {
    await saveApiKey(value);
    setApiKey(value.trim() || null);
    queryClient.clear();
  }, []);

  const client = useMemo(() => {
    if (!baseUrl) {
      return null;
    }
    try {
      return new NewsletterClient({ baseUrl, apiKey: apiKey ?? undefined });
    } catch {
      return null;
    }
  }, [baseUrl, apiKey]);

  const value = useMemo<ApiContextValue>(
    () => ({
      ready,
      baseUrl,
      apiKey,
      client,
      hasApiKey: Boolean(apiKey),
      updateBaseUrl,
      updateApiKey,
    }),
    [ready, baseUrl, apiKey, client, updateBaseUrl, updateApiKey]
  );

  return (
    <ApiContext.Provider value={value}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </ApiContext.Provider>
  );
}

export function useApi(): ApiContextValue {
  const context = useContext(ApiContext);
  if (!context) {
    throw new Error('useApi must be used within ApiProvider');
  }
  return context;
}

export function describeError(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return 'something went wrong';
}

export function newIdempotencyKey(): string {
  const random = Array.from({ length: 16 }, () =>
    Math.floor(Math.random() * 16).toString(16)
  ).join('');
  return `mobile-${Date.now()}-${random}`;
}
