import { useEffect, useState } from 'react';

import { ThemedText } from '@/components/themed-text';
import { Banner, Button, Card, Field, Screen, SectionTitle } from '@/components/ui/kit';
import { describeError, useApi } from '@/lib/api-context';
import { NewsletterClient } from '@/lib/sdk';

type Feedback = { kind: 'good' | 'critical'; message: string } | null;

export default function SettingsScreen() {
  const { baseUrl, apiKey, updateBaseUrl, updateApiKey, ready } = useApi();

  const [baseUrlDraft, setBaseUrlDraft] = useState('');
  const [apiKeyDraft, setApiKeyDraft] = useState('');
  const [saveFeedback, setSaveFeedback] = useState<Feedback>(null);
  const [testFeedback, setTestFeedback] = useState<Feedback>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (ready) {
      setBaseUrlDraft(baseUrl ?? '');
      setApiKeyDraft(apiKey ?? '');
    }
    // Seed drafts once storage has loaded.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready]);

  const handleSave = async () => {
    setBusy(true);
    setSaveFeedback(null);
    setTestFeedback(null);
    try {
      await updateBaseUrl(baseUrlDraft);
      await updateApiKey(apiKeyDraft);
      setSaveFeedback({ kind: 'good', message: 'settings saved' });
    } catch (error) {
      setSaveFeedback({ kind: 'critical', message: describeError(error) });
    } finally {
      setBusy(false);
    }
  };

  const handleTest = async () => {
    setBusy(true);
    setTestFeedback(null);
    try {
      const trimmed = baseUrlDraft.trim();
      if (!trimmed) {
        throw new Error('enter a base URL first');
      }
      const probe = new NewsletterClient({ baseUrl: trimmed });
      const health = await probe.health();
      setTestFeedback({ kind: 'good', message: `connected — API status: ${health.status}` });
    } catch (error) {
      setTestFeedback({ kind: 'critical', message: describeError(error) });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen>
      <ThemedText type="subtitle">Settings</ThemedText>

      <SectionTitle>API server</SectionTitle>
      <Card>
        <ThemedText type="small" themeColor="textSecondary">
          Use your machine&apos;s LAN IP on a physical device (for example
          http://192.168.1.20:3001). Android emulators reach the host via http://10.0.2.2:3001.
        </ThemedText>
        <Field
          placeholder="http://192.168.1.20:3001"
          keyboardType="url"
          value={baseUrlDraft}
          onChangeText={setBaseUrlDraft}
        />
      </Card>

      <SectionTitle>Admin API key</SectionTitle>
      <Card>
        <ThemedText type="small" themeColor="textSecondary">
          Required for the Admin and Dashboard tabs. Stored securely on this device.
        </ThemedText>
        <Field
          placeholder="admin API key"
          secureTextEntry
          value={apiKeyDraft}
          onChangeText={setApiKeyDraft}
        />
      </Card>

      <Button title="Save" onPress={handleSave} busy={busy} />
      {saveFeedback && <Banner kind={saveFeedback.kind} message={saveFeedback.message} />}

      <Button title="Test connection" onPress={handleTest} variant="secondary" busy={busy} />
      {testFeedback && <Banner kind={testFeedback.kind} message={testFeedback.message} />}
    </Screen>
  );
}
