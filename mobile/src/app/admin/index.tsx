import { useQuery } from '@tanstack/react-query';
import { Link, useRouter } from 'expo-router';
import { Pressable, RefreshControl, StyleSheet, View } from 'react-native';

import { ThemedText } from '@/components/themed-text';
import { Banner, Button, Card, Screen, StatusPill } from '@/components/ui/kit';
import { Spacing } from '@/constants/theme';
import { describeError, useApi } from '@/lib/api-context';
import type { NewsletterSendRecord } from '@/lib/sdk';

export default function AdminScreen() {
  const { client, hasApiKey, ready } = useApi();
  const router = useRouter();

  const query = useQuery({
    queryKey: ['newsletter-sends'],
    queryFn: () => client!.listNewsletterSends({ limit: 50 }),
    enabled: Boolean(client) && hasApiKey,
  });

  if (ready && (!client || !hasApiKey)) {
    return (
      <Screen>
        <Banner
          kind="info"
          message={
            client
              ? 'Set your admin API key in Settings to manage newsletters.'
              : 'No API base URL configured yet.'
          }
        />
        <Button title="Open Settings" onPress={() => router.push('/settings')} />
      </Screen>
    );
  }

  return (
    <Screen
      refreshControl={
        <RefreshControl refreshing={query.isRefetching} onRefresh={() => query.refetch()} />
      }>
      <Button title="Compose newsletter" onPress={() => router.push('/admin/compose')} />

      {query.isPending && <Banner kind="info" message="Loading newsletters…" />}
      {query.isError && <Banner kind="critical" message={describeError(query.error)} />}

      {query.data?.items.length === 0 && (
        <Banner kind="info" message="No newsletters sent yet." />
      )}

      {query.data?.items.map((item) => (
        <SendRow key={item.id} item={item} />
      ))}
    </Screen>
  );
}

function SendRow({ item }: { item: NewsletterSendRecord }) {
  return (
    <Link href={{ pathname: '/admin/[id]', params: { id: item.id } }} asChild>
      <Pressable>
        <Card>
          <View style={styles.rowTop}>
            <ThemedText type="smallBold" numberOfLines={1} style={styles.subject}>
              {item.subject}
            </ThemedText>
            <StatusPill kind={statusKind(item.status)} label={item.status} />
          </View>
          <ThemedText type="small" themeColor="textSecondary">
            {item.sent_count.toLocaleString()} sent · {item.fail_count.toLocaleString()} failed ·{' '}
            {new Date(item.created_at).toLocaleString()}
          </ThemedText>
        </Card>
      </Pressable>
    </Link>
  );
}

export function statusKind(status: string): 'good' | 'warning' | 'critical' {
  if (status === 'done') {
    return 'good';
  }
  if (status === 'failed') {
    return 'critical';
  }
  return 'warning';
}

const styles = StyleSheet.create({
  rowTop: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: Spacing.two,
  },
  subject: {
    flexShrink: 1,
  },
});
