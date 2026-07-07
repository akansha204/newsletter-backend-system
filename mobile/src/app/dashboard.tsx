import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { RefreshControl, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import {
  Banner,
  Button,
  Card,
  Screen,
  SectionTitle,
  StatTile,
  StatusPill,
} from '@/components/ui/kit';
import { Spacing } from '@/constants/theme';
import { describeError, useApi } from '@/lib/api-context';
import { APIError, type HealthResponse } from '@/lib/sdk';

const POLL_INTERVAL_MS = 10_000;

// /health responds 503 with the same payload when a dependency is down;
// recover the per-dependency checks from the thrown APIError.
function healthFromError(error: unknown): HealthResponse | null {
  if (error instanceof APIError && error.data && typeof error.data === 'object') {
    const data = error.data as Partial<HealthResponse>;
    if (data.checks) {
      return data as HealthResponse;
    }
  }
  return null;
}

export default function DashboardScreen() {
  const { client, hasApiKey, ready } = useApi();
  const router = useRouter();

  const statsQuery = useQuery({
    queryKey: ['stats'],
    queryFn: () => client!.stats(),
    enabled: Boolean(client) && hasApiKey,
    refetchInterval: POLL_INTERVAL_MS,
  });

  const healthQuery = useQuery({
    queryKey: ['health'],
    queryFn: () => client!.health(),
    enabled: Boolean(client),
    refetchInterval: POLL_INTERVAL_MS,
    // The API returns 503 when a dependency is down; surface the payload instead of failing.
    retry: false,
  });

  if (ready && (!client || !hasApiKey)) {
    return (
      <Screen>
        <Banner
          kind="info"
          message={
            client
              ? 'Set your admin API key in Settings to view the dashboard.'
              : 'No API base URL configured yet.'
          }
        />
        <Button title="Open Settings" onPress={() => router.push('/settings')} />
      </Screen>
    );
  }

  const stats = statsQuery.data;
  const lastSend = stats?.newsletters.last_send;

  const refreshing = statsQuery.isRefetching || healthQuery.isRefetching;
  const onRefresh = () => {
    statsQuery.refetch();
    healthQuery.refetch();
  };

  return (
    <Screen refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} />}>
      <ThemedText type="subtitle">Dashboard</ThemedText>

      {statsQuery.isError && <Banner kind="critical" message={describeError(statsQuery.error)} />}

      {stats && (
        <>
          <SectionTitle>Subscribers</SectionTitle>
          <View style={styles.tiles}>
            <StatTile label="Total" value={stats.subscribers.total} />
            <StatTile label="Confirmed" value={stats.subscribers.confirmed} />
          </View>

          <SectionTitle>Email delivery</SectionTitle>
          <View style={styles.tiles}>
            <StatTile label="Emails sent" value={stats.newsletters.sent_total} />
            <StatTile label="Emails failed" value={stats.newsletters.fail_total} />
            <StatTile label="Newsletters" value={stats.newsletters.total} />
          </View>

          {lastSend && (
            <>
              <SectionTitle>Last newsletter</SectionTitle>
              <Card>
                <ThemedText type="smallBold" numberOfLines={1}>
                  {lastSend.subject}
                </ThemedText>
                <ThemedText type="small" themeColor="textSecondary">
                  {lastSend.sent_count.toLocaleString()} sent ·{' '}
                  {lastSend.fail_count.toLocaleString()} failed · {lastSend.status}
                </ThemedText>
                <ThemedText type="small" themeColor="textSecondary">
                  {new Date(lastSend.created_at).toLocaleString()}
                </ThemedText>
              </Card>
            </>
          )}
        </>
      )}

      <SectionTitle>Backend health</SectionTitle>
      <Card>
        <HealthChecks
          health={healthQuery.data ?? healthFromError(healthQuery.error)}
          error={healthQuery.isError ? healthQuery.error : null}
        />
      </Card>
    </Screen>
  );
}

function HealthChecks({
  health,
  error,
}: {
  health: HealthResponse | null;
  error: unknown | null;
}) {
  if (health) {
    return (
      <>
        {Object.entries(health.checks).map(([name, check]) => (
          <View key={name} style={styles.healthRow}>
            <ThemedText type="small">{name}</ThemedText>
            <StatusPill kind={check.status === 'up' ? 'good' : 'critical'} label={check.status} />
          </View>
        ))}
      </>
    );
  }

  if (error) {
    return <Banner kind="critical" message={describeError(error)} />;
  }

  return (
    <ThemedText type="small" themeColor="textSecondary">
      Checking…
    </ThemedText>
  );
}

const styles = StyleSheet.create({
  tiles: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: Spacing.three,
  },
  healthRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
});
