import { useQuery } from '@tanstack/react-query';
import { useLocalSearchParams } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { statusKind } from '@/app/admin/index';
import { ThemedText } from '@/components/themed-text';
import { Banner, Card, Screen, SectionTitle, StatTile, StatusPill } from '@/components/ui/kit';
import { Spacing } from '@/constants/theme';
import { describeError, useApi } from '@/lib/api-context';

const ACTIVE_STATUSES = new Set(['pending', 'sending']);

export default function NewsletterDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { client, hasApiKey } = useApi();

  const query = useQuery({
    queryKey: ['newsletter-send', id],
    queryFn: () => client!.getNewsletterSend(id),
    enabled: Boolean(client) && hasApiKey && Boolean(id),
    // Keep polling while the worker is still draining this send's queue jobs.
    refetchInterval: (q) =>
      q.state.data && ACTIVE_STATUSES.has(q.state.data.status) ? 3000 : false,
  });

  if (query.isPending) {
    return (
      <Screen>
        <Banner kind="info" message="Loading newsletter…" />
      </Screen>
    );
  }

  if (query.isError) {
    return (
      <Screen>
        <Banner kind="critical" message={describeError(query.error)} />
      </Screen>
    );
  }

  const send = query.data;

  return (
    <Screen>
      <View style={styles.header}>
        <ThemedText type="subtitle" style={styles.subject}>
          {send.subject}
        </ThemedText>
        <StatusPill kind={statusKind(send.status)} label={send.status} />
      </View>

      <View style={styles.tiles}>
        <StatTile label="Sent" value={send.sent_count} />
        <StatTile label="Failed" value={send.fail_count} />
      </View>

      <SectionTitle>Body</SectionTitle>
      <Card>
        <ThemedText type="small">{send.body}</ThemedText>
      </Card>

      <SectionTitle>Timeline</SectionTitle>
      <Card>
        <ThemedText type="small" themeColor="textSecondary">
          Created {new Date(send.created_at).toLocaleString()}
        </ThemedText>
        <ThemedText type="small" themeColor="textSecondary">
          Updated {new Date(send.updated_at).toLocaleString()}
        </ThemedText>
        <ThemedText type="code" themeColor="textSecondary">
          {send.id}
        </ThemedText>
      </Card>
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    gap: Spacing.two,
  },
  subject: {
    fontSize: 24,
    lineHeight: 30,
  },
  tiles: {
    flexDirection: 'row',
    gap: Spacing.three,
  },
});
