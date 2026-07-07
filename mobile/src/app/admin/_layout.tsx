import { Stack } from 'expo-router';
import { useColorScheme } from 'react-native';

import { Colors } from '@/constants/theme';

export default function AdminLayout() {
  const scheme = useColorScheme();
  const colors = Colors[scheme === 'dark' ? 'dark' : 'light'];

  return (
    <Stack
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerTintColor: colors.text,
        contentStyle: { backgroundColor: colors.background },
      }}>
      <Stack.Screen name="index" options={{ title: 'Newsletters' }} />
      <Stack.Screen name="compose" options={{ title: 'Compose', presentation: 'modal' }} />
      <Stack.Screen name="[id]" options={{ title: 'Newsletter' }} />
    </Stack>
  );
}
