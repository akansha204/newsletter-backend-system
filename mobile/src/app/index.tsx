import { useRouter } from 'expo-router';
import { useState } from 'react';

import { ThemedText } from '@/components/themed-text';
import { Banner, Button, Card, Field, Screen, SectionTitle } from '@/components/ui/kit';
import { describeError, useApi } from '@/lib/api-context';
import { APIError } from '@/lib/sdk';

type Feedback = { kind: 'good' | 'warning' | 'critical'; message: string } | null;

export default function SubscribeScreen() {
  const { client, ready } = useApi();
  const router = useRouter();

  const [email, setEmail] = useState('');
  const [subscribeBusy, setSubscribeBusy] = useState(false);
  const [subscribeFeedback, setSubscribeFeedback] = useState<Feedback>(null);

  const [token, setToken] = useState('');
  const [confirmBusy, setConfirmBusy] = useState(false);
  const [confirmFeedback, setConfirmFeedback] = useState<Feedback>(null);

  const handleSubscribe = async () => {
    if (!client) {
      return;
    }
    setSubscribeBusy(true);
    setSubscribeFeedback(null);
    try {
      const response = await client.subscribe(email.trim());
      setSubscribeFeedback({
        kind: 'good',
        message: response.message ?? 'check your email to confirm your subscription',
      });
      setEmail('');
    } catch (error) {
      setSubscribeFeedback(subscribeErrorFeedback(error));
    } finally {
      setSubscribeBusy(false);
    }
  };

  const handleConfirm = async () => {
    if (!client) {
      return;
    }
    setConfirmBusy(true);
    setConfirmFeedback(null);
    try {
      const response = await client.confirm(token.trim());
      setConfirmFeedback({
        kind: 'good',
        message: response.message ?? 'subscription confirmed',
      });
      setToken('');
    } catch (error) {
      setConfirmFeedback({ kind: 'critical', message: describeError(error) });
    } finally {
      setConfirmBusy(false);
    }
  };

  if (ready && !client) {
    return (
      <Screen>
        <ThemedText type="subtitle">Newsletter</ThemedText>
        <Banner kind="info" message="No API base URL configured yet." />
        <Button title="Open Settings" onPress={() => router.push('/settings')} />
      </Screen>
    );
  }

  return (
    <Screen>
      <ThemedText type="subtitle">Newsletter</ThemedText>

      <SectionTitle>Subscribe</SectionTitle>
      <Card>
        <Field
          placeholder="you@example.com"
          keyboardType="email-address"
          value={email}
          onChangeText={setEmail}
        />
        <Button
          title="Subscribe"
          onPress={handleSubscribe}
          disabled={!email.trim()}
          busy={subscribeBusy}
        />
        {subscribeFeedback && (
          <Banner kind={subscribeFeedback.kind} message={subscribeFeedback.message} />
        )}
      </Card>

      <SectionTitle>Confirm subscription</SectionTitle>
      <Card>
        <ThemedText type="small" themeColor="textSecondary">
          Paste the token from your confirmation email.
        </ThemedText>
        <Field placeholder="confirmation token" value={token} onChangeText={setToken} />
        <Button
          title="Confirm"
          onPress={handleConfirm}
          disabled={!token.trim()}
          busy={confirmBusy}
          variant="secondary"
        />
        {confirmFeedback && (
          <Banner kind={confirmFeedback.kind} message={confirmFeedback.message} />
        )}
      </Card>
    </Screen>
  );
}

function subscribeErrorFeedback(error: unknown): Feedback {
  if (error instanceof APIError) {
    if (error.statusCode === 409) {
      return { kind: 'warning', message: 'this email is already subscribed' };
    }
    if (error.statusCode === 429) {
      return { kind: 'warning', message: 'too many requests — try again in a minute' };
    }
  }
  return { kind: 'critical', message: describeError(error) };
}
