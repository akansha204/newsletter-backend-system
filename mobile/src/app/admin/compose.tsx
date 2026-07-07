import { useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';

import { ThemedText } from '@/components/themed-text';
import { Banner, Button, Card, Field, Screen } from '@/components/ui/kit';
import { describeError, newIdempotencyKey, useApi } from '@/lib/api-context';
import { APIError } from '@/lib/sdk';

export default function ComposeScreen() {
  const { client, hasApiKey } = useApi();
  const queryClient = useQueryClient();
  const router = useRouter();

  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSend = Boolean(client) && hasApiKey && subject.trim() !== '' && body.trim() !== '';

  const handleSend = async () => {
    if (!client) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const response = await client.sendNewsletter(
        { subject: subject.trim(), body: body.trim() },
        { idempotencyKey: newIdempotencyKey() }
      );
      queryClient.invalidateQueries({ queryKey: ['newsletter-sends'] });
      queryClient.invalidateQueries({ queryKey: ['stats'] });
      if (response.id) {
        router.replace({ pathname: '/admin/[id]', params: { id: response.id } });
      } else {
        router.back();
      }
    } catch (err) {
      setError(composeErrorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen>
      <Card>
        <ThemedText type="small" themeColor="textSecondary">
          Sends to every confirmed subscriber.
        </ThemedText>
        <Field placeholder="Subject" value={subject} onChangeText={setSubject} />
        <Field
          placeholder="Write your newsletter…"
          value={body}
          onChangeText={setBody}
          multiline
          numberOfLines={8}
          style={{ minHeight: 160, textAlignVertical: 'top' }}
        />
        <Button title="Send newsletter" onPress={handleSend} disabled={!canSend} busy={busy} />
        {error && <Banner kind="critical" message={error} />}
      </Card>
    </Screen>
  );
}

function composeErrorMessage(error: unknown): string {
  if (error instanceof APIError) {
    if (error.statusCode === 401 || error.statusCode === 403) {
      return 'admin API key was rejected — check it in Settings';
    }
    if (error.statusCode === 409) {
      return 'a send with this idempotency key is already in progress — try again';
    }
  }
  return describeError(error);
}
