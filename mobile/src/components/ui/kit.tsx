import { PropsWithChildren, type ReactElement } from 'react';
import {
  ActivityIndicator,
  Pressable,
  type RefreshControlProps,
  ScrollView,
  StyleSheet,
  TextInput,
  type TextInputProps,
  View,
  useColorScheme,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';

import { ThemedText } from '@/components/themed-text';
import { ThemedView } from '@/components/themed-view';
import {
  BottomTabInset,
  MaxContentWidth,
  PrimaryColor,
  Spacing,
  StatusColors,
  type StatusKind,
} from '@/constants/theme';
import { useTheme } from '@/hooks/use-theme';

export function useStatusColors() {
  const scheme = useColorScheme();
  return StatusColors[scheme === 'dark' ? 'dark' : 'light'];
}

export function Screen({
  children,
  refreshControl,
}: PropsWithChildren<{ refreshControl?: ReactElement<RefreshControlProps> }>) {
  return (
    <ThemedView style={styles.screen}>
      <SafeAreaView edges={['top']} style={styles.safeArea}>
        <ScrollView
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          refreshControl={refreshControl}>
          {children}
        </ScrollView>
      </SafeAreaView>
    </ThemedView>
  );
}

export function Card({ children, style }: PropsWithChildren<{ style?: object }>) {
  return (
    <ThemedView type="backgroundElement" style={[styles.card, style]}>
      {children}
    </ThemedView>
  );
}

export function Field(props: TextInputProps) {
  const theme = useTheme();
  return (
    <TextInput
      placeholderTextColor={theme.textSecondary}
      autoCapitalize="none"
      autoCorrect={false}
      {...props}
      style={[
        styles.field,
        {
          color: theme.text,
          backgroundColor: theme.backgroundElement,
          borderColor: theme.backgroundSelected,
        },
        props.style,
      ]}
    />
  );
}

export function Button({
  title,
  onPress,
  disabled,
  busy,
  variant = 'primary',
}: {
  title: string;
  onPress: () => void;
  disabled?: boolean;
  busy?: boolean;
  variant?: 'primary' | 'secondary';
}) {
  const theme = useTheme();
  const inactive = disabled || busy;
  const background = variant === 'primary' ? PrimaryColor : theme.backgroundSelected;
  const label = variant === 'primary' ? '#ffffff' : theme.text;

  return (
    <Pressable
      onPress={onPress}
      disabled={inactive}
      style={({ pressed }) => [
        styles.button,
        { backgroundColor: background },
        (pressed || inactive) && styles.buttonDim,
      ]}>
      {busy ? (
        <ActivityIndicator size="small" color={label} />
      ) : (
        <ThemedText type="smallBold" style={{ color: label }}>
          {title}
        </ThemedText>
      )}
    </Pressable>
  );
}

export function Banner({ kind, message }: { kind: StatusKind | 'info'; message: string }) {
  const status = useStatusColors();
  const theme = useTheme();
  const color = kind === 'info' ? theme.textSecondary : status[kind];
  const prefix = kind === 'good' ? '✓' : kind === 'info' ? 'ℹ' : '!';

  return (
    <Card style={{ borderLeftWidth: 3, borderLeftColor: color }}>
      <ThemedText type="small" style={{ color }}>
        {prefix} {message}
      </ThemedText>
    </Card>
  );
}

export function StatTile({ label, value }: { label: string; value: string | number }) {
  return (
    <Card style={styles.statTile}>
      <ThemedText type="small" themeColor="textSecondary">
        {label}
      </ThemedText>
      <ThemedText type="subtitle" numberOfLines={1} adjustsFontSizeToFit>
        {formatValue(value)}
      </ThemedText>
    </Card>
  );
}

export function StatusPill({ kind, label }: { kind: StatusKind; label: string }) {
  const status = useStatusColors();
  return (
    <View style={styles.pill}>
      <View style={[styles.dot, { backgroundColor: status[kind] }]} />
      <ThemedText type="smallBold" style={{ color: status[kind] }}>
        {label}
      </ThemedText>
    </View>
  );
}

export function SectionTitle({ children }: PropsWithChildren) {
  return (
    <ThemedText type="smallBold" themeColor="textSecondary" style={styles.sectionTitle}>
      {children}
    </ThemedText>
  );
}

function formatValue(value: string | number): string {
  if (typeof value === 'number') {
    return value.toLocaleString();
  }
  return value;
}

const styles = StyleSheet.create({
  screen: {
    flex: 1,
    flexDirection: 'row',
    justifyContent: 'center',
  },
  safeArea: {
    flex: 1,
    maxWidth: MaxContentWidth,
  },
  scrollContent: {
    padding: Spacing.three,
    gap: Spacing.three,
    paddingBottom: BottomTabInset + Spacing.four,
  },
  card: {
    borderRadius: Spacing.three,
    padding: Spacing.three,
    gap: Spacing.two,
  },
  field: {
    borderRadius: Spacing.two,
    borderWidth: 1,
    paddingHorizontal: Spacing.three,
    paddingVertical: Spacing.two + Spacing.one,
    fontSize: 16,
  },
  button: {
    borderRadius: Spacing.two,
    paddingVertical: Spacing.two + Spacing.one,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 44,
  },
  buttonDim: {
    opacity: 0.6,
  },
  statTile: {
    flexGrow: 1,
    flexBasis: '45%',
  },
  pill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: Spacing.two,
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
  },
  sectionTitle: {
    textTransform: 'uppercase',
    marginTop: Spacing.two,
  },
});
